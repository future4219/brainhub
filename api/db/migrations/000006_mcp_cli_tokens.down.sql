DROP INDEX mcp_tokens_cli_user_created_idx;

DELETE FROM mcp_tokens WHERE token_type = 'cli';

ALTER TABLE mcp_tokens
    DROP CONSTRAINT mcp_tokens_shape_check,
    DROP CONSTRAINT mcp_tokens_token_type_check,
    DROP COLUMN label,
    ALTER COLUMN client_id SET NOT NULL;

ALTER TABLE mcp_tokens
    ADD CONSTRAINT mcp_tokens_token_type_check
        CHECK (token_type IN ('access', 'refresh'));
