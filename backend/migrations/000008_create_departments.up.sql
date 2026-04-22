-- Migration: 000008_create_departments (up)

CREATE TABLE IF NOT EXISTS departments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_by  UUID        NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ

);

CREATE INDEX idx_departments_org_id    ON departments (org_id);
CREATE INDEX idx_departments_deleted_at ON departments (deleted_at) WHERE deleted_at IS NULL;

COMMENT ON TABLE departments IS 'Named sub-groups within an organization (e.g. Engineering, Sales).';
