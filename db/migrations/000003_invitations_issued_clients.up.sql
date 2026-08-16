CREATE TABLE invitations (
    id          TEXT PRIMARY KEY,
    brain_id    TEXT NOT NULL REFERENCES brains(id) ON DELETE RESTRICT,
    email       TEXT,
    role        TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'reader')),
    token_hash  TEXT NOT NULL UNIQUE,
    invited_by  TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    state       TEXT NOT NULL CHECK (state IN ('pending', 'accepted', 'revoked', 'expired')),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    accepted_by TEXT REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX invitations_brain_state_idx
    ON invitations (brain_id, state, created_at);

CREATE TABLE issued_clients (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    brain_id          TEXT NOT NULL REFERENCES brains(id) ON DELETE RESTRICT,
    gbrain_client_id  TEXT UNIQUE,
    label             TEXT NOT NULL,
    write_source_id   TEXT,
    read_source_ids   TEXT[] NOT NULL,
    scopes            TEXT[] NOT NULL,
    state             TEXT NOT NULL CHECK (state IN ('issuing', 'active', 'revoked', 'orphan')),
    state_reason      TEXT NOT NULL DEFAULT '',
    issued_at         TIMESTAMPTZ NOT NULL,
    last_verified_at  TIMESTAMPTZ,
    revoked_at        TIMESTAMPTZ
);

CREATE INDEX issued_clients_user_brain_state_idx
    ON issued_clients (user_id, brain_id, state, issued_at);
