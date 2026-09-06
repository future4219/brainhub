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

// ReadConnectionStatus is the non-secret state shown in connection settings.
type ReadConnectionStatus struct {
	State       ReaderClientState
	StateReason string
}
