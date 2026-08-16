package entity

import "time"

type ClientState string

const (
	ClientStateIssuing ClientState = "issuing"
	ClientStateActive  ClientState = "active"
	ClientStateRevoked ClientState = "revoked"
	ClientStateOrphan  ClientState = "orphan"
)

type IssuedClient struct {
	ID             string
	UserID         string
	BrainID        string
	GBrainClientID *string
	Label          string
	WriteSourceID  *SourceID
	ReadSourceIDs  []SourceID
	Scopes         []string
	State          ClientState
	StateReason    string
	IssuedAt       time.Time
	LastVerifiedAt *time.Time
	RevokedAt      *time.Time
}
