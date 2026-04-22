-- Migration: 000009_create_department_memberships (up)

CREATE TABLE IF NOT EXISTS department_memberships (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID        NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role          TEXT        NOT NULL DEFAULT 'member'
                              CHECK (role IN ('head', 'member')),
    joined_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_dept_memberships_dept_user UNIQUE (department_id, user_id)
);

CREATE INDEX idx_dept_memberships_dept_id ON department_memberships (department_id);
CREATE INDEX idx_dept_memberships_user_id ON department_memberships (user_id);

COMMENT ON TABLE department_memberships IS 'Junction between users and departments with head/member roles.';
