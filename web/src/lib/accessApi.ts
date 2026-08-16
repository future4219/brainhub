import type {
  AcceptedInvitation,
  CreatedInvitation,
  Invitation,
  InvitationPreview,
  IssuedClient,
  Role,
} from "@/entities/access/entity";
import { postJSON, requestJSON } from "@/lib/api";

export function createInvitation(sourceID: string, input: { email: string; role: Role; expires_at: string }): Promise<CreatedInvitation> {
  return postJSON(`/brains/${encodeURIComponent(sourceID)}/invitations`, input);
}

export function listInvitations(sourceID: string): Promise<Invitation[]> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}/invitations`);
}

export function revokeInvitation(id: string): Promise<void> {
  return requestJSON(`/invitations/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function getInvitation(token: string): Promise<InvitationPreview> {
  return requestJSON(`/invitations/${encodeURIComponent(token)}`);
}

export function acceptInvitation(token: string): Promise<AcceptedInvitation> {
  return postJSON(`/invitations/${encodeURIComponent(token)}/accept`);
}

export function issueClient(sourceID: string): Promise<IssuedClient> {
  return postJSON(`/brains/${encodeURIComponent(sourceID)}/clients`, { label: "claude-web" });
}

export function listClients(sourceID: string): Promise<IssuedClient[]> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}/clients`);
}

export function revokeClient(id: string): Promise<void> {
  return requestJSON(`/clients/${encodeURIComponent(id)}`, { method: "DELETE" });
}
