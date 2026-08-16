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
