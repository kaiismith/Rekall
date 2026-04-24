CREATE TABLE IF NOT EXISTS meeting_messages (
    id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    meeting_id UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    body       TEXT        NOT NULL,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT meeting_messages_pkey        PRIMARY KEY (id),
    CONSTRAINT meeting_messages_meeting_fk  FOREIGN KEY (meeting_id) REFERENCES meetings(id) ON DELETE CASCADE,
    CONSTRAINT meeting_messages_user_fk     FOREIGN KEY (user_id)    REFERENCES users(id)    ON DELETE CASCADE,
    CONSTRAINT meeting_messages_body_length CHECK (char_length(body) BETWEEN 1 AND 2000)
);

-- Supports chronological read (ORDER BY sent_at DESC LIMIT 50) and the
-- cursor-pagination query (sent_at < $before). Partial on deleted_at so
-- future soft-delete rows are skipped without a table scan.
CREATE INDEX meeting_messages_room_time_idx
    ON meeting_messages(meeting_id, sent_at DESC)
    WHERE deleted_at IS NULL;
