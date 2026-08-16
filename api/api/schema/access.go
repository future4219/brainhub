package schema

import (
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type CreateInvitationRequest struct {
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

type InvitationResponse struct {
	ID         string     `json:"id"`
	Email      *string    `json:"email"`
	Role       string     `json:"role"`
	State      string     `json:"state"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	AcceptedAt *time.Time `json:"accepted_at"`
	AcceptedBy *string    `json:"accepted_by"`
}

type CreatedInvitationResponse struct {
	InvitationResponse
	Token string `json:"token"`
}

func InvitationResponseFromEntity(invitation entity.Invitation) InvitationResponse {
	return InvitationResponse{
		ID: invitation.ID, Email: invitation.Email, Role: string(invitation.Role), State: string(invitation.State),
		ExpiresAt: invitation.ExpiresAt, CreatedAt: invitation.CreatedAt,
		AcceptedAt: invitation.AcceptedAt, AcceptedBy: invitation.AcceptedBy,
	}
}

type InvitationPreviewResponse struct {
	BrainName     string `json:"brain_name"`
	InvitedByName string `json:"invited_by_name"`
}

func InvitationPreviewResponseFromInput(preview input_port.InvitationPreview) InvitationPreviewResponse {
	return InvitationPreviewResponse{BrainName: preview.BrainName, InvitedByName: preview.InvitedByName}
}

type AcceptedInvitationResponse struct {
	SourceID  string `json:"source_id"`
	BrainName string `json:"brain_name"`
	Role      string `json:"role"`
}

type CreateClientRequest struct {
	Label string `json:"label"`
}

type IssuedClientResponse struct {
	ID             string     `json:"id"`
	ClientID       *string    `json:"client_id"`
	Label          string     `json:"label"`
	WriteSourceID  *string    `json:"write_source_id"`
	ReadSourceIDs  []string   `json:"read_source_ids"`
	Scopes         []string   `json:"scopes"`
	State          string     `json:"state"`
	StateReason    string     `json:"state_reason"`
	IssuedAt       time.Time  `json:"issued_at"`
	LastVerifiedAt *time.Time `json:"last_verified_at"`
	RevokedAt      *time.Time `json:"revoked_at"`
}

type IssuedClientErrorResponse struct {
	Error  string               `json:"error"`
	Client IssuedClientResponse `json:"client"`
}

func IssuedClientResponseFromEntity(client entity.IssuedClient) IssuedClientResponse {
	readSourceIDs := make([]string, len(client.ReadSourceIDs))
	for i, sourceID := range client.ReadSourceIDs {
		readSourceIDs[i] = sourceID.String()
	}
	var writeSourceID *string
	if client.WriteSourceID != nil {
		value := client.WriteSourceID.String()
		writeSourceID = &value
	}
	return IssuedClientResponse{
		ID: client.ID, ClientID: client.GBrainClientID, Label: client.Label,
		WriteSourceID: writeSourceID, ReadSourceIDs: readSourceIDs, Scopes: client.Scopes,
		State: string(client.State), StateReason: client.StateReason, IssuedAt: client.IssuedAt,
		LastVerifiedAt: client.LastVerifiedAt, RevokedAt: client.RevokedAt,
	}
}
