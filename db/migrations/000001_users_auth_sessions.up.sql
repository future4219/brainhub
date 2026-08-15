CREATE TABLE users (
    id         TEXT PRIMARY KEY,
    email      TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    state      TEXT NOT NULL CHECK (state IN ('active', 'suspended')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE auth_identities (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    provider    TEXT NOT NULL CHECK (provider IN ('password', 'google', 'github')),
    identifier  TEXT NOT NULL,
    secret_hash TEXT,
    created_at  TIMESTAMPTZ NOT NULL,
    UNIQUE(provider, identifier),
    CHECK (
        (provider = 'password' AND secret_hash IS NOT NULL)
        OR (provider <> 'password' AND secret_hash IS NULL)
    )
);

CREATE INDEX auth_identities_user_id_idx ON auth_identities(user_id);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX sessions_user_id_idx ON sessions(user_id);
