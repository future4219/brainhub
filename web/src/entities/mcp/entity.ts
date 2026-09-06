export type MCPClient = {
  id: string;
  name: string;
};

export type MCPVisibleBrain = {
  can_write: boolean;
  source_id: string;
  name: string;
  state: string;
  role: string;
};

export type MCPReader = {
  state: string;
  state_reason: string;
};

export type MCPCLIToken = {
  id: string;
  label: string;
  expires_at: string;
  created_at: string;
};

export type CreatedMCPCLIToken = MCPCLIToken & {
  token: string;
};

export type MCPConnection = {
  client: MCPClient | null;
  codex_client: MCPClient | null;
  cli_tokens: MCPCLIToken[];
  visible_brains: MCPVisibleBrain[];
  reader: MCPReader | null;
};

export type OAuthConsent = {
  scope: "read" | "read write";
  client_name: string;
  client_id: string;
  visible_brains: MCPVisibleBrain[];
};
