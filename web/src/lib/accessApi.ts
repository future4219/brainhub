import type {
  AcceptedInvitation,
  CreatedInvitation,
  Invitation,
  InvitationPreview,
  Role,
} from "@/entities/access/entity";
import { postJSON, requestJSON } from "@/lib/api";

export function createInvitation(
  sourceID: string,
  input: { email: string; role: Role; expires_at: string },
): Promise<CreatedInvitation> {
  return postJSON(`/brains/${encodeURIComponent(sourceID)}/invitations`, input);
}

export function listInvitations(sourceID: string): Promise<Invitation[]> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}/invitations`);
}

export function revokeInvitation(id: string): Promise<void> {
  return requestJSON(`/invitations/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export function getInvitation(token: string): Promise<InvitationPreview> {
  return requestJSON(`/invitations/${encodeURIComponent(token)}`);
}

export function acceptInvitation(token: string): Promise<AcceptedInvitation> {
  return postJSON(`/invitations/${encodeURIComponent(token)}/accept`);
}
