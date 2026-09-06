package entity

import (
	"net/url"
	"slices"
	"strconv"
	"time"
)

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

// Native OAuth clients choose a local listener port at login (RFC 8252).
// Only the registered Codex loopback callback permits a variable port.
func (c MCPClient) AllowsRedirectURI(raw string) bool {
	if slices.Contains(c.RedirectURIs, raw) {
		return true
	}
	if c.Name != "codex" || !slices.Contains(c.RedirectURIs, "http://127.0.0.1/callback") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil ||
		u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port <= 0 || port > 65535 || u.Host != "127.0.0.1:"+u.Port() {
		return false
	}
	u.Host = "127.0.0.1"
	return slices.Contains(c.RedirectURIs, u.String())
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
