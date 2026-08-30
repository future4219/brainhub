import type {
  MCPClient,
  MCPConnection,
  OAuthConsent,
} from "@/entities/mcp/entity";
import { postJSON, requestJSON } from "@/lib/api";

export function getMCPConnection(): Promise<MCPConnection> {
  return requestJSON("/mcp/connection");
}

export function issueMCPClient(): Promise<MCPClient> {
  return postJSON("/mcp/client");
}

export function reissueMCPReader(): Promise<void> {
  return postJSON("/mcp/reader/reissue");
}

export function getOAuthConsent(search: string): Promise<OAuthConsent> {
  return requestJSON(`/oauth/authorization${search}`);
}

export function decideOAuth(
  search: string,
  decision: "approve" | "deny",
): Promise<{ redirect_uri: string }> {
  const params = new URLSearchParams(search);
  params.set("decision", decision);
  return requestJSON("/oauth/authorization", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params.toString(),
  });
}
