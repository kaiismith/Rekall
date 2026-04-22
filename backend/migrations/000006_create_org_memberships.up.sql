-- Migration: 000006_create_org_memberships (up)

CREATE TABLE IF NOT EXISTS org_memberships (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id    UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT        NOT NULL DEFAULT 'member'
                          CHECK (role IN ('owner', 'admin', 'member')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_org_memberships_org_user UNIQUE (org_id, user_id)
);

CREATE INDEX idx_org_memberships_org_id  ON org_memberships (org_id);
CREATE INDEX idx_org_memberships_user_id ON org_memberships (user_id);

COMMENT ON TABLE org_memberships IS 'Junction between users and organizations with role-based access.';
