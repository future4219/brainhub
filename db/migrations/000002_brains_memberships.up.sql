CREATE TABLE brains (
    id           TEXT PRIMARY KEY,
    source_id    TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    visibility   TEXT NOT NULL CHECK (visibility IN ('public', 'private')),
    owner_id     TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    state        TEXT NOT NULL CHECK (state IN ('provisioning', 'ready', 'degraded', 'failed', 'archived')),
    state_reason TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    archived_at  TIMESTAMPTZ
);

CREATE TABLE memberships (
    id         TEXT PRIMARY KEY,
    brain_id   TEXT NOT NULL REFERENCES brains(id) ON DELETE RESTRICT,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'reader')),
    invited_by TEXT REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    UNIQUE (brain_id, user_id)
);

CREATE INDEX memberships_active_user_idx
    ON memberships (user_id, brain_id)
    WHERE revoked_at IS NULL;
