CREATE TABLE IF NOT EXISTS meetings (
    id               UUID        NOT NULL DEFAULT gen_random_uuid(),
    code             TEXT        NOT NULL,
    title            TEXT        NOT NULL DEFAULT '',
    type             TEXT        NOT NULL DEFAULT 'open',
    scope_type       TEXT,
    scope_id         UUID,
    host_id          UUID        NOT NULL,
    status           TEXT        NOT NULL DEFAULT 'waiting',
    max_participants INTEGER     NOT NULL DEFAULT 50,
    started_at       TIMESTAMPTZ,
    ended_at         TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT meetings_pkey         PRIMARY KEY (id),
    CONSTRAINT meetings_type_check   CHECK (type IN ('open', 'private')),
    CONSTRAINT meetings_status_check CHECK (status IN ('waiting', 'active', 'ended')),
    CONSTRAINT meetings_host_fk      FOREIGN KEY (host_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX meetings_code_idx        ON meetings(code);
CREATE        INDEX meetings_host_status_idx ON meetings(host_id, status);
CREATE        INDEX meetings_created_idx     ON meetings(created_at DESC);
