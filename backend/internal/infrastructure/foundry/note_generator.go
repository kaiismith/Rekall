package foundry

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
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

// IsConfigured returns false when no auth strategy could be picked at boot.
// In that case Generate short-circuits with ErrFoundryUnconfigured.
func (g *NoteGenerator) IsConfigured() bool {
	return g.client != nil && g.client.configured
}

// Generate renders the kat-v1 prompt, calls Foundry, parses the response, and
// returns a structured NoteGeneratorOutput on success.
func (g *NoteGenerator) Generate(ctx context.Context, in ports.NoteGeneratorInput) (*ports.NoteGeneratorOutput, error) {
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
	parsed, usage, err := g.callOnce(callCtx, system, user, false)
	latency := time.Since(start)

	// On bad JSON, retry once with a corrective system message.
	if errors.Is(err, errBadJSON) {
		correctiveStart := time.Now()
		parsed, usage, err = g.callOnce(callCtx, system, user, true)
		latency = time.Since(correctiveStart)
		if errors.Is(err, errBadJSON) {
			catalog.KatFoundryParseFailed.Warn(g.log)
			return nil, ErrFoundryParseFailed
		}
	}

	if err != nil {
		// Retry once on retriable errors; if that succeeds, fall through to
		// the success packaging below. Otherwise map to a sentinel.
		retryParsed, retryUsage, sentinel := g.classifyAndRetry(callCtx, err, system, user)
		if sentinel != nil {
			return nil, sentinel
		}
		// classifyAndRetry may return (nil, nil, nil) when it elects not to retry;
		// in that path treat it as the original error mapped to Unavailable.
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

// retryAfter sleeps for d (respecting ctx) and then runs callOnce a single
// time. Returns the parsed result if successful or the underlying error.
func (g *NoteGenerator) retryAfter(
	ctx context.Context,
	d time.Duration,
	system, user string,
	corrective bool,
) (*noteResponseSchema, *openai.CompletionUsage, error) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-timer.C:
	}
	return g.callOnce(ctx, system, user, corrective)
}

// errBadJSON is an internal sentinel signalling a parseable HTTP success but a
// non-conforming response body. Triggers the corrective retry.
var errBadJSON = errors.New("foundry: bad json response")

// callOnce performs one Foundry chat-completions request with response_format
// set to json_object. When corrective is true a second system message tells
// the model to re-emit JSON only.
func (g *NoteGenerator) callOnce(
	ctx context.Context,
	system, user string,
	corrective bool,
) (*noteResponseSchema, *openai.CompletionUsage, error) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(system),
		openai.UserMessage(user),
	}
	if corrective {
		messages = append(messages, openai.SystemMessage(
			"Your previous response was not valid JSON. Re-emit ONLY the JSON object.",
		))
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(g.client.deployment),
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{Type: "json_object"},
		},
		Temperature: openai.Float(0.2),
	}

	resp, err := g.client.oai.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, nil, errBadJSON
	}

	content := resp.Choices[0].Message.Content
	var parsed noteResponseSchema
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, nil, errBadJSON
	}
	if parsed.KeyPoints == nil {
		parsed.KeyPoints = []string{}
	}
	if parsed.OpenQuestions == nil {
		parsed.OpenQuestions = []string{}
	}
	return &parsed, &resp.Usage, nil
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
