-- Migration: 000005_create_organizations (up)

CREATE TABLE IF NOT EXISTS organizations (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL,
    owner_id   UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_organizations_slug_active
    ON organizations (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_organizations_owner_id  ON organizations (owner_id);
CREATE INDEX idx_organizations_created_at ON organizations (created_at DESC);

COMMENT ON TABLE organizations IS 'Named workspaces that group users around a shared set of calls.';
