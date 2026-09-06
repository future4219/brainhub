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
