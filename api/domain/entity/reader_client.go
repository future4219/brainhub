package entity

import "time"

type ReaderClientState string

const (
	ReaderClientStateIssuing ReaderClientState = "issuing"
	ReaderClientStateActive  ReaderClientState = "active"
	ReaderClientStateOrphan  ReaderClientState = "orphan"
)

type ReaderClient struct {
	ID                     string
	UserID                 string
	GBrainClientID         *string
	ClientSecretCiphertext []byte
	FederatedRead          []SourceID
	State                  ReaderClientState
	StateReason            string
	IssuedAt               *time.Time
	LastVerifiedAt         *time.Time
}
