CREATE TABLE mcp_clients (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name           TEXT NOT NULL,
    redirect_uris  TEXT[] NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    revoked_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX mcp_clients_active_user_name_idx
    ON mcp_clients (user_id, name)
    WHERE revoked_at IS NULL;

CREATE TABLE mcp_authorization_codes (
    id              TEXT PRIMARY KEY,
    code_hash       TEXT NOT NULL UNIQUE,
    client_id       TEXT NOT NULL REFERENCES mcp_clients(id) ON DELETE RESTRICT,
    user_id         TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    redirect_uri    TEXT NOT NULL,
    code_challenge  TEXT NOT NULL,
    resource        TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ
);

CREATE INDEX mcp_authorization_codes_client_user_idx
    ON mcp_authorization_codes (client_id, user_id, expires_at);

CREATE TABLE mcp_tokens (
    id          TEXT PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,
    token_type  TEXT NOT NULL CHECK (token_type IN ('access', 'refresh')),
    client_id   TEXT NOT NULL REFERENCES mcp_clients(id) ON DELETE RESTRICT,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX mcp_tokens_client_user_type_idx
    ON mcp_tokens (client_id, user_id, token_type, expires_at);

CREATE TABLE brainhub_reader_clients (
    id                        TEXT PRIMARY KEY,
    user_id                   TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    gbrain_client_id          TEXT UNIQUE,
    client_secret_ciphertext  BYTEA,
    federated_read            TEXT[] NOT NULL DEFAULT '{}',
    state                     TEXT NOT NULL CHECK (state IN ('issuing', 'active', 'orphan')),
    state_reason              TEXT NOT NULL DEFAULT '',
    issued_at                 TIMESTAMPTZ,
    last_verified_at          TIMESTAMPTZ,
    CHECK (
        (state = 'active' AND gbrain_client_id IS NOT NULL AND client_secret_ciphertext IS NOT NULL)
        OR state <> 'active'
    )
);

-- Existing issued_clients intentionally remain untouched. They are historical
-- GBrain OAuth clients; the new /mcp path accepts only mcp_tokens.
