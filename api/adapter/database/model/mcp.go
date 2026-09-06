package model

import "time"

type MCPClient struct {
	ID           string
	UserID       string
	Name         string
	RedirectURIs []string
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

type MCPAuthorizationCode struct {
	WriteAllowed  bool
	ID            string
	ClientID      string
	UserID        string
	RedirectURI   string
	CodeChallenge string
	Resource      string
	ExpiresAt     time.Time
	CreatedAt     time.Time
	ConsumedAt    *time.Time
}

type MCPToken struct {
	WriteAllowed bool
	ID           string
	Type         string
	ClientID     *string
	UserID       string
	Label        *string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

type ReaderClient struct {
	ID                     string
	UserID                 string
	GBrainClientID         *string
	ClientSecretCiphertext []byte
	FederatedRead          []string
	State                  string
	StateReason            string
	IssuedAt               *time.Time
	LastVerifiedAt         *time.Time
}
