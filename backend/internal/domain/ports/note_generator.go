package ports

import (
	"context"

	"github.com/google/uuid"
)

// TranscriptSegmentLite is the redacted shape passed to the LLM at the
// generation seam. Speaker UUIDs are deliberately absent — only the
// human-readable label crosses to the model.
type TranscriptSegmentLite struct {
	SpeakerLabel string
	Text         string
	StartMs      int32
	EndMs        int32
}

// NoteGeneratorInput carries everything the Kat scheduler needs to render a
// prompt for one Generation_Run.
type NoteGeneratorInput struct {
	PromptVersion   string
	MeetingTitle    string               // empty for solo calls
	SpeakerLabels   map[uuid.UUID]string // for traceability only; not sent over the wire
	Segments        []TranscriptSegmentLite
	PreviousSummary string // last successful summary; "" for first run
}

// NoteGeneratorOutput is the structured note returned by the model after
// validation against the Kat JSON schema.
type NoteGeneratorOutput struct {
	Summary          string
	KeyPoints        []string
	OpenQuestions    []string
	ModelID          string
	PromptTokens     *int32
	CompletionTokens *int32
	LatencyMs        int32
}

// NoteGenerator is the seam between the Kat application service and the
// underlying LLM provider (today: Azure AI Foundry). The implementation lives
// in backend/internal/infrastructure/foundry; the application service depends
// only on this interface.
// StreamCallback receives the running concatenated text as the LLM streams
// tokens. Invoked off the hot path (every ~few tokens), callable many times.
// `partial` is the running raw model output (plain text, not JSON). When the
// stream ends, Generate returns the final parsed result and the callback is
// not invoked again.
type StreamCallback func(partial string)

type NoteGenerator interface {
	// Generate runs one provider call and parses the response. When `onChunk`
	// is non-nil and the underlying provider supports streaming, the callback
	// receives the running raw text as tokens arrive. Either way Generate
	// returns the final parsed result on success or a foundry sentinel error.
	Generate(ctx context.Context, in NoteGeneratorInput, onChunk StreamCallback) (*NoteGeneratorOutput, error)

	// ModelID is the canonical model / deployment name; "" when not configured.
	ModelID() string

	// AuthMode is one of "api_key" | "managed_identity" | "none".
	AuthMode() string

	// Provider identifies the backend: "foundry" | "openai" | "" (unconfigured).
	Provider() string

	// IsConfigured returns false when the adapter could not construct any auth
	// strategy at boot. Frontend renders the offline state in that case.
	IsConfigured() bool
}
