export type MCPClient = {
  id: string;
  name: string;
};

export type MCPVisibleBrain = {
  source_id: string;
  name: string;
  state: string;
  role: string;
};

export type MCPReader = {
  state: string;
  state_reason: string;
};

export type MCPConnection = {
  client: MCPClient | null;
  visible_brains: MCPVisibleBrain[];
  reader: MCPReader | null;
};

export type OAuthConsent = {
  client_name: string;
  client_id: string;
};
