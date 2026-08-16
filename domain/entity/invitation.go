package entity

import "time"

type InvitationState string

const (
	InvitationStatePending  InvitationState = "pending"
	InvitationStateAccepted InvitationState = "accepted"
	InvitationStateRevoked  InvitationState = "revoked"
	InvitationStateExpired  InvitationState = "expired"
)

type Invitation struct {
	ID         string
	BrainID    string
	Email      *string
	Role       Role
	TokenHash  string
	InvitedBy  string
	State      InvitationState
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AcceptedAt *time.Time
	AcceptedBy *string
}

func (i Invitation) IsAcceptable(now time.Time) bool {
	return i.State == InvitationStatePending && now.Before(i.ExpiresAt)
}
