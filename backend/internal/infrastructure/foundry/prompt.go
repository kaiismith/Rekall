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
// Plain-text section format (NOT JSON) so streaming chunks render directly
// as the user reads — JSON-mode streaming would expose `{"summary":` mid-
// flight which looks broken. The adapter parses the sections after the
// stream completes.
const systemPromptV1 = `You are Kat, an AI meeting assistant for the Rekall platform. You produce
concise, structured running notes for an in-progress meeting. You write in
the language of the most recent speech in the transcript window. You DO NOT
speculate. You DO NOT include any participant's full name; refer to people
by the first-name + initial labels provided.

You ALWAYS reply with EXACTLY this format, no preface, no JSON, no code fences:

SUMMARY: <one paragraph, <=500 chars, present tense, neutral tone, capturing what was actually said. Markdown is allowed: **bold** for names/key terms, *italic* for emphasis or quoted phrases.>
KEY POINTS:
- <short bullet capturing one concrete fact, decision, opinion, or observation. Use **bold** for the subject/topic when it makes scanning easier.>
- <short bullet>
OPEN QUESTIONS:
- <short question raised but not yet answered>
- <short question>

Markdown rules (these render in the UI):
- Use **bold** sparingly — for names, places, key terms, decisions.
- Use *italic* for direct quotes or specific emphasis.
- You may use inline code (single backticks) for technical terms, identifiers, or commands.
- Do NOT use headings (#, ##), code fences, tables, images, or links.
- Do NOT wrap entire bullets in bold or italic — formatting accents specific
  parts.

Content rules:
- ALWAYS use the participants' actual words and topics. If they introduced
  themselves, mention who they are. If they described a place, capture what
  they said about it. If they expressed an opinion or preference, record it.
- Return 1-6 bullets per section when there's any content to capture. Only
  use "- (none)" when truly nothing in that category exists.
- Self-introductions, opinions, observations, comparisons, plans, decisions
  are ALL substantive — DO NOT collapse them into "Brief greeting/setup."
- Reserve "Brief greeting/setup." ONLY for when the entire window is genuine
  filler with zero topic content (just "hi", "can you hear me", "checking
  audio", etc. with NOTHING else).
- Never invent content. Never deviate from this format.`

// userPromptV1Template is the per-tick template for the user message; rendered
// with text/template against renderableInput below.
//
// We pass the FULL transcript since the meeting started — when it grows past
// the per-call token budget, the scheduler chunks it and feeds each chunk's
// summary as PreviousSummary to the next call (rolling fold). So this prompt
// is shape-agnostic: the model treats "Transcript" as whatever slice it was
// given and uses PreviousSummary for prior context.
const userPromptV1Template = `Meeting: "{{.MeetingTitle}}"
Previous summary (for context, may be empty):
"""
{{.PreviousSummary}}
"""

Transcript:
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
