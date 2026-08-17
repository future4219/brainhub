CREATE TABLE brain_writer_clients (
    brain_id                  TEXT PRIMARY KEY REFERENCES brains(id) ON DELETE RESTRICT,
    write_source_id           TEXT NOT NULL UNIQUE,
    gbrain_client_id          TEXT UNIQUE,
    client_secret_ciphertext  BYTEA,
    state                     TEXT NOT NULL CHECK (state IN ('issuing', 'active', 'orphan')),
    state_reason              TEXT NOT NULL DEFAULT '',
    issued_at                 TIMESTAMPTZ,
    last_verified_at          TIMESTAMPTZ,
    CHECK (
        (state = 'active' AND gbrain_client_id IS NOT NULL AND client_secret_ciphertext IS NOT NULL)
        OR state <> 'active'
    )
);

-- Existing brains predate persistent writer clients. Startup backfill claims
-- these issuing rows and provisions one source-bound client per brain.
INSERT INTO brain_writer_clients (brain_id, write_source_id, state)
SELECT id, source_id, 'issuing'
FROM brains;

-- No key_version column in v1: key rotation is intentionally out of scope.
-- Adding a nullable key_version later is backward-compatible; until then every
-- ciphertext is encrypted by the single BRAINHUB_WRITER_CREDENTIAL_KEY.
