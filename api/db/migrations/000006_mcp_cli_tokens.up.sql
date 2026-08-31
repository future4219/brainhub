ALTER TABLE mcp_tokens
    DROP CONSTRAINT mcp_tokens_token_type_check,
    ALTER COLUMN client_id DROP NOT NULL,
    ADD COLUMN label TEXT;

ALTER TABLE mcp_tokens
    ADD CONSTRAINT mcp_tokens_token_type_check
        CHECK (token_type IN ('access', 'refresh', 'cli')),
    ADD CONSTRAINT mcp_tokens_shape_check CHECK (
        (token_type IN ('access', 'refresh') AND client_id IS NOT NULL AND label IS NULL)
        OR
        (token_type = 'cli' AND client_id IS NULL AND label = BTRIM(label)
            AND CHAR_LENGTH(label) BETWEEN 1 AND 64)
    );

CREATE INDEX mcp_tokens_cli_user_created_idx
    ON mcp_tokens (user_id, created_at DESC)
    WHERE token_type = 'cli';
