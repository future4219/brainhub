package entity

import "time"

type MCPVisibleBrain struct {
	SourceID SourceID
	Name     string
	State    string
	Role     string
}

type MCPClient struct {
	ID           string
	UserID       string
	Name         string
	RedirectURIs []string
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

type MCPAuthorizationCode struct {
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

type MCPTokenType string

const (
	MCPTokenAccess  MCPTokenType = "access"
	MCPTokenRefresh MCPTokenType = "refresh"
	MCPTokenCLI     MCPTokenType = "cli"
)

type MCPToken struct {
	ID        string
	Type      MCPTokenType
	ClientID  *string
	UserID    string
	Label     *string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}
