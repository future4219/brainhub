ALTER TABLE mcp_tokens DROP CONSTRAINT mcp_cli_read_only;
ALTER TABLE mcp_tokens DROP COLUMN write_allowed;
ALTER TABLE mcp_authorization_codes DROP COLUMN write_allowed;
