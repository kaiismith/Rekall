-- Adds the per-meeting toggle for the live-captions / ASR feature.
-- Default false so historic meetings stay unaffected and new meetings opt in
-- explicitly via the host's CreateMeeting form.
ALTER TABLE meetings
    ADD COLUMN IF NOT EXISTS transcription_enabled BOOLEAN NOT NULL DEFAULT FALSE;
