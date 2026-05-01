package foundry

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"

	"github.com/rekall/backend/internal/domain/ports"
	"github.com/rekall/backend/pkg/logger/catalog"
)

// noteResponseSchema is the JSON shape Kat asks the model to emit. The
// adapter parses this directly; a corrective retry kicks in on shape mismatch.
type noteResponseSchema struct {
	Summary       string   `json:"summary"`
	KeyPoints     []string `json:"key_points"`
	OpenQuestions []string `json:"open_questions"`
}

// NoteGenerator implements ports.NoteGenerator over the Foundry Client.
//
// Resilience policy is intentionally narrow:
//   - one corrective retry on bad JSON
//   - one retry on 429 (honouring Retry-After clamped to <=30s)
//   - one retry on 5xx (1-second jittered sleep)
//   - any second failure surfaces a foundry sentinel and the scheduler enters
//     error-cooldown
//
// No exponential-backoff loops here — that's the scheduler's job, not the
// per-request path.
type NoteGenerator struct {
	client        *Client
	windowSeconds int
	log           *zap.Logger
	rng           *rand.Rand
}

// NewNoteGenerator constructs a NoteGenerator from a Foundry Client. The
// windowSeconds is rendered into the user prompt so the model knows the
// temporal scope it's summarizing.
func NewNoteGenerator(client *Client, windowSeconds int, log *zap.Logger) *NoteGenerator {
	if windowSeconds <= 0 {
		windowSeconds = 120
	}
	return &NoteGenerator{
		client:        client,
		windowSeconds: windowSeconds,
		log:           log,
		// Per-instance RNG so tests with a deterministic seed remain stable.
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ModelID returns the configured deployment name; "" when not configured.
func (g *NoteGenerator) ModelID() string {
	if g.client == nil {
		return ""
	}
	return g.client.deployment
}

// AuthMode returns the boot-selected auth strategy.
func (g *NoteGenerator) AuthMode() string {
	if g.client == nil {
		return AuthModeNone
	}
	return g.client.authMode
}

// Provider returns the selected backend ("foundry" | "openai" | "").
func (g *NoteGenerator) Provider() string {
	if g.client == nil {
		return ""
	}
	return g.client.Provider()
}

// IsConfigured returns false when no auth strategy could be picked at boot.
// In that case Generate short-circuits with ErrFoundryUnconfigured.
func (g *NoteGenerator) IsConfigured() bool {
	return g.client != nil && g.client.configured
}

// Generate renders the kat-v1 prompt, calls Foundry / OpenAI with streaming,
// emits incremental chunks via onChunk, and returns the final parsed output.
// onChunk may be nil (non-streaming consumers).
func (g *NoteGenerator) Generate(
	ctx context.Context,
	in ports.NoteGeneratorInput,
	onChunk ports.StreamCallback,
) (*ports.NoteGeneratorOutput, error) {
	if !g.IsConfigured() {
		return nil, ErrFoundryUnconfigured
	}

	system, user, err := renderPrompt(in, g.windowSeconds)
	if err != nil {
		return nil, err
	}

	timeout := g.client.timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	parsed, usage, err := g.streamOnce(callCtx, system, user, onChunk)
	latency := time.Since(start)

	if err != nil {
		// Retriable transport / 5xx / 429 error: try once more (without
		// streaming chunks during retry — keeps the panel from flickering
		// the same partial twice).
		retryParsed, retryUsage, sentinel := g.classifyAndRetry(callCtx, err, system, user)
		if sentinel != nil {
			return nil, sentinel
		}
		if retryParsed == nil {
			return nil, ErrFoundryUnavailable
		}
		parsed, usage = retryParsed, retryUsage
		latency = time.Since(start)
	}

	out := &ports.NoteGeneratorOutput{
		Summary:       parsed.Summary,
		KeyPoints:     parsed.KeyPoints,
		OpenQuestions: parsed.OpenQuestions,
		ModelID:       g.client.deployment,
		LatencyMs:     int32(latency / time.Millisecond),
	}
	if usage != nil {
		pt := int32(usage.PromptTokens)
		ct := int32(usage.CompletionTokens)
		out.PromptTokens = &pt
		out.CompletionTokens = &ct
	}
	return out, nil
}

// classifyAndRetry applies the per-status retry policy.
//
// Returns one of three shapes:
//   - (parsed, usage, nil)  — retry succeeded; caller should use the result
//   - (nil,    nil,   sentinel) — terminal failure; caller surfaces sentinel
//   - (nil,    nil,   nil) — non-retriable error (e.g. 4xx other than 429);
//     caller treats as Unavailable.
//
// Called only on a non-bad-JSON error; the bad-JSON corrective retry happens
// in Generate.
func (g *NoteGenerator) classifyAndRetry(
	ctx context.Context,
	err error,
	system, user string,
) (*noteResponseSchema, *openai.CompletionUsage, error) {
	// Context deadline -> Unavailable + dedicated log.
	if errors.Is(err, context.DeadlineExceeded) {
		catalog.KatFoundryTimeout.Warn(g.log)
		return nil, nil, ErrFoundryUnavailable
	}

	apiErr := asAPIError(err)
	if apiErr == nil {
		// Transport-layer failure (DNS, TLS, connection reset) — same retry
		// posture as a 5xx: try once more, then give up.
		parsed, usage, retryErr := g.retryAfter(ctx, time.Second+jitter(g.rng, 250*time.Millisecond), system, user, false)
		if retryErr == nil {
			return parsed, usage, nil
		}
		if errors.Is(retryErr, context.DeadlineExceeded) {
			catalog.KatFoundryTimeout.Warn(g.log)
			return nil, nil, ErrFoundryUnavailable
		}
		catalog.KatFoundryUnavailable.Warn(g.log, zap.Error(err))
		return nil, nil, ErrFoundryUnavailable
	}

	switch {
	case apiErr.StatusCode == http.StatusTooManyRequests:
		retryAfter := min(parseRetryAfter(apiErr.Response, g.rng), 30*time.Second)
		parsed, usage, retryErr := g.retryAfter(ctx, retryAfter, system, user, false)
		if retryErr == nil {
			return parsed, usage, nil
		}
		catalog.KatFoundryRateLimited.Warn(g.log)
		return nil, nil, ErrFoundryRateLimited
	case apiErr.StatusCode >= 500 && apiErr.StatusCode < 600:
		parsed, usage, retryErr := g.retryAfter(ctx, time.Second+jitter(g.rng, 250*time.Millisecond), system, user, false)
		if retryErr == nil {
			return parsed, usage, nil
		}
		catalog.KatFoundryUnavailable.Warn(g.log, zap.Int("status", apiErr.StatusCode))
		return nil, nil, ErrFoundryUnavailable
	default:
		// 4xx other than 429 — not retryable; surface as Unavailable for the
		// scheduler (cooldown), with the status_code in the log for forensics.
		catalog.KatFoundryUnavailable.Warn(g.log, zap.Int("status", apiErr.StatusCode))
		return nil, nil, ErrFoundryUnavailable
	}
}

// retryAfter sleeps for d (respecting ctx) and then runs streamOnce a single
// time. Returns the parsed result if successful or the underlying error.
// Retry path runs without streaming chunks — keeps the panel from rendering
// the same partial twice if the first attempt mid-streamed before failing.
func (g *NoteGenerator) retryAfter(
	ctx context.Context,
	d time.Duration,
	system, user string,
	_corrective bool,
) (*noteResponseSchema, *openai.CompletionUsage, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-timer.C:
	}
	return g.streamOnce(ctx, system, user, nil)
}

// streamOnce performs one Foundry / OpenAI chat-completions request with
// streaming enabled. As tokens arrive it concatenates them into a buffer and
// invokes onChunk (when non-nil) with the running text. When the stream ends
// it parses the plain-text section format defined in prompt.go (kat-v1).
func (g *NoteGenerator) streamOnce(
	ctx context.Context,
	system, user string,
	onChunk ports.StreamCallback,
) (*noteResponseSchema, *openai.CompletionUsage, error) {
	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(g.client.deployment),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		Temperature: openai.Float(0.2),
	}

	stream := g.client.oai.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()

	var (
		buf      strings.Builder
		lastEmit time.Time
		// Throttle WS broadcasts. 250ms gives ~4 fps of partials — fast
		// enough to feel live, slow enough that the React renderer doesn't
		// drop intermediate states and the user sees the typing animation
		// progress visibly. The frontend-side typewriter further smooths
		// gaps for short responses where OpenAI ships everything in <1s.
		emitEvery = 250 * time.Millisecond
		usage     *openai.CompletionUsage
	)

	for stream.Next() {
		chunk := stream.Current()
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				buf.WriteString(delta)
				if onChunk != nil && time.Since(lastEmit) >= emitEvery {
					onChunk(buf.String())
					lastEmit = time.Now()
				}
			}
		}
		// Some providers emit `usage` in a final chunk; capture if present.
		if chunk.Usage.TotalTokens > 0 {
			u := chunk.Usage
			usage = &u
		}
	}
	if err := stream.Err(); err != nil {
		return nil, nil, err
	}

	final := buf.String()
	if final == "" {
		g.log.Warn("kat: foundry returned empty stream",
			zap.String("model", g.client.deployment),
		)
		return nil, nil, ErrFoundryParseFailed
	}
	// Final emit so the frontend gets the last few tokens that fell inside
	// the throttle window.
	if onChunk != nil {
		onChunk(final)
	}

	// Log the raw model output so we can debug "Brief greeting/setup" /
	// section-marker / empty-bullets cases without re-running the meeting.
	// This fires on EVERY successful generation — Info level. Trim very
	// long responses to keep log lines bounded.
	rawForLog := final
	if len(rawForLog) > 2000 {
		rawForLog = rawForLog[:2000] + "...[truncated]"
	}
	g.log.Info("kat: foundry raw response",
		zap.String("model", g.client.deployment),
		zap.Int("raw_len", len(final)),
		zap.String("raw", rawForLog),
	)

	parsed, err := parseSectionedResponse(final)
	if err != nil {
		catalog.KatFoundryParseFailed.Warn(g.log,
			zap.Error(err),
			zap.String("raw", rawForLog),
		)
		return nil, nil, ErrFoundryParseFailed
	}

	// Log the parsed structured output for diagnostic visibility into what
	// the panel will eventually render.
	g.log.Info("kat: foundry parsed response",
		zap.String("summary", parsed.Summary),
		zap.Int("summary_len", len(parsed.Summary)),
		zap.Strings("key_points", parsed.KeyPoints),
		zap.Strings("open_questions", parsed.OpenQuestions),
	)

	return parsed, usage, nil
}

// parseSectionedResponse turns the kat-v1 plain-text format into a
// noteResponseSchema. Tolerant of casing, extra whitespace, missing trailing
// sections, and the literal "(none)" placeholder.
func parseSectionedResponse(text string) (*noteResponseSchema, error) {
	out := &noteResponseSchema{
		KeyPoints:     []string{},
		OpenQuestions: []string{},
	}

	// Section markers: SUMMARY: ... KEY POINTS: ... OPEN QUESTIONS: ...
	upper := strings.ToUpper(text)
	idxSummary := strings.Index(upper, "SUMMARY:")
	idxKey := strings.Index(upper, "KEY POINTS:")
	idxOpen := strings.Index(upper, "OPEN QUESTIONS:")

	// summary: everything between SUMMARY: and KEY POINTS: (or end)
	if idxSummary >= 0 {
		startSum := idxSummary + len("SUMMARY:")
		endSum := len(text)
		if idxKey > idxSummary {
			endSum = idxKey
		} else if idxOpen > idxSummary {
			endSum = idxOpen
		}
		out.Summary = strings.TrimSpace(text[startSum:endSum])
	} else {
		// Defensive fallback: treat the whole response as the summary.
		out.Summary = strings.TrimSpace(text)
	}

	// key_points: bullets between KEY POINTS: and OPEN QUESTIONS: (or end)
	if idxKey >= 0 {
		startKP := idxKey + len("KEY POINTS:")
		endKP := len(text)
		if idxOpen > idxKey {
			endKP = idxOpen
		}
		out.KeyPoints = parseBullets(text[startKP:endKP])
	}
	// open_questions: bullets after OPEN QUESTIONS: to end
	if idxOpen >= 0 {
		out.OpenQuestions = parseBullets(text[idxOpen+len("OPEN QUESTIONS:"):])
	}

	if out.Summary == "" {
		return nil, errors.New("empty summary")
	}
	return out, nil
}

// parseBullets extracts "- ..." lines, drops "(none)" placeholders, returns
// a (possibly empty) slice.
func parseBullets(s string) []string {
	out := []string{}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Accept "-", "*", "•" as bullet markers.
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "•") {
			line = strings.TrimSpace(strings.TrimLeft(line, "-*• "))
		}
		if line == "" {
			continue
		}
		// Skip "(none)" / "none" placeholders.
		lower := strings.ToLower(line)
		if lower == "(none)" || lower == "none" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// asAPIError unwraps an openai.Error if present.
func asAPIError(err error) *openai.Error {
	if err == nil {
		return nil
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

// parseRetryAfter reads the Retry-After header and returns its value as a
// duration. Honours both delta-seconds and HTTP-date formats. Falls back to a
// jittered 1-second wait when the header is absent or unparseable.
func parseRetryAfter(resp *http.Response, rng *rand.Rand) time.Duration {
	const fallback = time.Second
	if resp == nil {
		return fallback + jitter(rng, 250*time.Millisecond)
	}
	h := resp.Header.Get("Retry-After")
	if h == "" {
		return fallback + jitter(rng, 250*time.Millisecond)
	}
	if secs, err := strconv.Atoi(h); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = fallback
		}
		return d
	}
	return fallback + jitter(rng, 250*time.Millisecond)
}

// jitter returns a small randomised offset in [0, max) used to spread retry
// timings across cohorts so a flapping Foundry doesn't get hammered in lockstep.
func jitter(rng *rand.Rand, max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rng.Int63n(int64(max)))
}
