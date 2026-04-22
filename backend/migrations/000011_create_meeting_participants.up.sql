CREATE TABLE IF NOT EXISTS meeting_participants (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    meeting_id  UUID        NOT NULL,
    user_id     UUID        NOT NULL,
    role        TEXT        NOT NULL DEFAULT 'participant',
    invited_by  UUID,
    joined_at   TIMESTAMPTZ,
    left_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT meeting_participants_pkey        PRIMARY KEY (id),
    CONSTRAINT meeting_participants_role_check  CHECK (role IN ('host', 'participant')),
    CONSTRAINT meeting_participants_meeting_fk  FOREIGN KEY (meeting_id) REFERENCES meetings(id)  ON DELETE CASCADE,
    CONSTRAINT meeting_participants_user_fk     FOREIGN KEY (user_id)    REFERENCES users(id)     ON DELETE CASCADE,
    CONSTRAINT meeting_participants_inviter_fk  FOREIGN KEY (invited_by) REFERENCES users(id),
    CONSTRAINT meeting_participants_unique      UNIQUE (meeting_id, user_id)
);

CREATE INDEX meeting_participants_meeting_idx ON meeting_participants(meeting_id);
CREATE INDEX meeting_participants_user_idx    ON meeting_participants(user_id);

-- Partial index for fast active-participant count queries.
CREATE INDEX meeting_participants_active_idx ON meeting_participants(meeting_id)
    WHERE left_at IS NULL;
