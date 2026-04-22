-- Migration: 000007_create_invitations (up)

CREATE TABLE IF NOT EXISTS invitations (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invited_by  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email       TEXT        NOT NULL,
    token_hash  TEXT        NOT NULL UNIQUE,
    role        TEXT        NOT NULL DEFAULT 'member'
                            CHECK (role IN ('admin', 'member')),
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invitations_org_id     ON invitations (org_id);
CREATE INDEX idx_invitations_email      ON invitations (email);
CREATE INDEX idx_invitations_token_hash ON invitations (token_hash);

COMMENT ON TABLE invitations IS 'Pending email invitations to join an organization.';
