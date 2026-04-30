package foundry

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/rekall/backend/internal/domain/ports"
)

// PromptVersionV1 is the canonical first prompt for Kat. Stamped onto every
// note via the WS payload so a future kat-v2 stays distinguishable.
const PromptVersionV1 = "kat-v1"

// systemPromptV1 is the static system message used for the kat-v1 prompt.
// It instructs the model to refer to itself as "Kat", to redact participant
// full names (relying on the speaker labels we provide), and to always reply
// with the structured JSON object the adapter parses.
const systemPromptV1 = `You are Kat, an AI meeting assistant for the Rekall platform. You produce concise,
structured running notes for an in-progress meeting. You write in the language of
the most recent speech in the transcript window. You DO NOT speculate. You DO NOT
include any participant's full name; refer to people by the first-name + initial
labels provided. You ALWAYS reply with a single JSON object matching this schema:

{
  "summary":         "<paragraph, <=500 chars, present tense, neutral tone>",
  "key_points":      ["<short bullet>", ...],
  "open_questions":  ["<short question raised but not yet answered>", ...]
}

Return between 0 and 6 entries in each list. If the window contains nothing
substantive (filler, hellos), return short summary like "Brief greeting/setup."
with empty lists. Never invent content. Never write more than the JSON.`

// userPromptV1Template is the per-tick template for the user message; rendered
// with text/template against renderableInput below.
const userPromptV1Template = `Meeting: "{{.MeetingTitle}}"
Previous summary (for context, may be empty):
"""
{{.PreviousSummary}}
"""

Recent transcript (last {{.WindowSeconds}} seconds):
{{- range .Segments }}
[{{.SpeakerLabel}}] {{.Text}}
{{- end }}`

type renderableInput struct {
	MeetingTitle    string
	PreviousSummary string
	WindowSeconds   int
	Segments        []ports.TranscriptSegmentLite
}

var compiledUserV1 = template.Must(template.New("kat-v1-user").Parse(userPromptV1Template))

// renderPrompt produces (systemMessage, userMessage) for the given prompt
// version + input. Returns an error for an unknown promptVersion.
func renderPrompt(in ports.NoteGeneratorInput, windowSeconds int) (system, user string, err error) {
	switch in.PromptVersion {
	case PromptVersionV1, "":
		var buf bytes.Buffer
		if err := compiledUserV1.Execute(&buf, renderableInput{
			MeetingTitle:    in.MeetingTitle,
			PreviousSummary: in.PreviousSummary,
			WindowSeconds:   windowSeconds,
			Segments:        in.Segments,
		}); err != nil {
			return "", "", fmt.Errorf("foundry: render user prompt: %w", err)
		}
		return systemPromptV1, buf.String(), nil
	default:
		return "", "", fmt.Errorf("foundry: unknown prompt_version %q", in.PromptVersion)
	}
}
