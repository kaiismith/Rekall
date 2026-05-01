package ws

import "github.com/google/uuid"

// KatLifecycle is the slice of KatNotesService the WS hub depends on. We
// declare it here as a local interface (not a domain port) to keep the
// dependency direction clean: ws -> KatLifecycle (interface) <- services
// (concrete impl). The hub does not import application/services.
//
// All methods are no-ops when Kat is disabled; the hub sets the field to nil
// in that case and skips the calls.
//
// The late-join replay is the service's responsibility (it knows the ring
// buffer): on OnParticipantJoined the service walks the buffer and uses its
// own WSBroadcaster.SendToUser to fan the historical notes out to the joiner.
// The hub does not handle the replay payload.
type KatLifecycle interface {
	OnParticipantJoined(meetingID uuid.UUID, userID uuid.UUID, hasActiveASR bool)
	OnParticipantLeft(meetingID uuid.UUID, isLast bool)
	OnMeetingEnded(meetingID uuid.UUID)

	// SetKatEnabled gates whether the scheduler will fire Foundry/OpenAI
	// calls for the meeting. The cohort entry stays alive (transcription
	// segments still flow into the ring buffer for replay), but ticks become
	// no-ops while the flag is false. This lets per-user "AI notes" toggles
	// stop the cost without tearing the cohort down.
	SetKatEnabled(meetingID uuid.UUID, enabled bool)
}
