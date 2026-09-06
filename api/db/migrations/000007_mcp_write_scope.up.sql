ALTER TABLE mcp_authorization_codes ADD COLUMN write_allowed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE mcp_tokens ADD COLUMN write_allowed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE mcp_tokens ADD CONSTRAINT mcp_cli_read_only CHECK (token_type <> 'cli' OR NOT write_allowed);
