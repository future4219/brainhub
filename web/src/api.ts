export type BrainState = "provisioning" | "ready" | "degraded" | "failed" | "archived";

export type Brain = {
  id: string;
  source_id: string;
  name: string;
  description: string;
  owner_id: string;
  visibility: "public" | "private";
  state: BrainState;
};

export type Role = "owner" | "editor" | "reader";

export type Invitation = {
  id: string;
  email: string | null;
  role: Role;
  state: "pending" | "accepted" | "revoked" | "expired";
  expires_at: string;
  created_at: string;
  accepted_at: string | null;
  accepted_by: string | null;
};

export type CreatedInvitation = Invitation & { token: string };

export type InvitationPreview = {
  brain_name: string;
  invited_by_name: string;
};

export type AcceptedInvitation = {
  source_id: string;
  brain_name: string;
  role: Role;
};

export type IssuedClient = {
  id: string;
  client_id: string | null;
  label: string;
  write_source_id: string | null;
  read_source_ids: string[];
  scopes: string[];
  state: "issuing" | "active" | "revoked" | "orphan";
  state_reason: string;
  issued_at: string;
  last_verified_at: string | null;
  revoked_at: string | null;
};

export type Page = {
  slug: string;
  title: string;
  type: string;
  updated_at: string;
};

export type PublicConfig = {
  mcp_url: string;
};

export type User = {
  id: string;
  email: string;
  name: string;
  state: string;
};

export type CreateBrainInput = Pick<Brain, "source_id" | "name" | "description" | "visibility">;

export class APIError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
  }
}

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      Accept: "application/json",
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const text = (await response.text()).trim();
    let message = text || response.statusText;
    try {
      const body = JSON.parse(text) as { error?: string };
      message = body.error || message;
    } catch {
      // Some API errors are intentionally plain text.
    }
    throw new APIError(response.status, message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

function postJSON<T>(path: string, body?: unknown): Promise<T> {
  return requestJSON(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

export function listBrains(): Promise<Brain[]> {
  return requestJSON("/api/brains");
}

export function getBrain(sourceID: string): Promise<Brain> {
  return requestJSON(`/api/brains/${encodeURIComponent(sourceID)}`);
}

export function listPages(sourceID: string): Promise<Page[]> {
  return requestJSON(`/api/brains/${encodeURIComponent(sourceID)}/pages`);
}

export function getPublicConfig(): Promise<PublicConfig> {
  return requestJSON("/api/config");
}

export async function getCurrentUser(): Promise<User | null> {
  try {
    return await requestJSON("/api/me");
  } catch (error) {
    if (error instanceof APIError && error.status === 401) return null;
    throw error;
  }
}

export function register(input: { email: string; password: string; name: string }): Promise<User> {
  return postJSON("/api/auth/register", input);
}

export function login(input: { email: string; password: string }): Promise<User> {
  return postJSON("/api/auth/login", input);
}

export function logout(): Promise<void> {
  return postJSON("/api/auth/logout");
}

export function createBrain(input: CreateBrainInput): Promise<Brain> {
  return postJSON("/api/brains", input);
}

export function createInvitation(sourceID: string, input: { email: string; role: Role; expires_at: string }): Promise<CreatedInvitation> {
  return postJSON(`/api/brains/${encodeURIComponent(sourceID)}/invitations`, input);
}

export function listInvitations(sourceID: string): Promise<Invitation[]> {
  return requestJSON(`/api/brains/${encodeURIComponent(sourceID)}/invitations`);
}

export function revokeInvitation(id: string): Promise<void> {
  return requestJSON(`/api/invitations/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function getInvitation(token: string): Promise<InvitationPreview> {
  return requestJSON(`/api/invitations/${encodeURIComponent(token)}`);
}

export function acceptInvitation(token: string): Promise<AcceptedInvitation> {
  return postJSON(`/api/invitations/${encodeURIComponent(token)}/accept`);
}

export function issueClient(sourceID: string): Promise<IssuedClient> {
  return postJSON(`/api/brains/${encodeURIComponent(sourceID)}/clients`, { label: "claude-web" });
}

export function listClients(sourceID: string): Promise<IssuedClient[]> {
  return requestJSON(`/api/brains/${encodeURIComponent(sourceID)}/clients`);
}

export function revokeClient(id: string): Promise<void> {
  return requestJSON(`/api/clients/${encodeURIComponent(id)}`, { method: "DELETE" });
}
