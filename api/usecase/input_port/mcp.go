package input_port

import (
	"context"
	"errors"
	"time"

	"brainhub/domain/entity"
)

var (
	ErrOAuthInvalidRequest  = errors.New("invalid OAuth request")
	ErrOAuthInvalidClient   = errors.New("invalid OAuth client")
	ErrOAuthInvalidGrant    = errors.New("invalid OAuth grant")
	ErrOAuthAccessDenied    = errors.New("OAuth access denied")
	ErrMCPUnauthorized      = errors.New("invalid MCP access token")
	ErrMCPTokenExpired      = errors.New("MCP CLI token expired")
	ErrMCPInvalidTokenLabel = errors.New("invalid MCP CLI token label")
	ErrMCPTokenNotFound     = errors.New("MCP CLI token not found")
	ErrMCPForbidden         = errors.New("MCP source is not visible")
)

type MCPConnection struct {
	Client        *entity.MCPClient
	CLITokens     []entity.MCPToken
	VisibleBrains []entity.MCPVisibleBrain
	Reader        *entity.ReaderClient
}

type IssuedCLIToken struct {
	Token    entity.MCPToken
	RawToken string
}

type OAuthAuthorizationRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
}

type OAuthAuthorization struct {
	Client entity.MCPClient
	Input  OAuthAuthorizationRequest
}

type OAuthTokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

type MCPCallAuthorization struct {
	GBrainToken string
	SourceID    *string
}

type MCPUseCase interface {
	Connection(context.Context, string) (MCPConnection, error)
	IssueClient(context.Context, string) (entity.MCPClient, error)
	IssueCLIToken(context.Context, string, string) (IssuedCLIToken, error)
	RevokeCLIToken(context.Context, string, string) error
	ValidateAuthorization(context.Context, OAuthAuthorizationRequest, string) (OAuthAuthorization, error)
	ApproveAuthorization(context.Context, OAuthAuthorizationRequest, string) (string, error)
	ExchangeAuthorizationCode(context.Context, string, string, string, string) (OAuthTokenPair, error)
	RefreshAccessToken(context.Context, string, string) (OAuthTokenPair, error)
	RevokeToken(context.Context, string, string) error
	AuthorizeCall(context.Context, string, string, *string) (MCPCallAuthorization, error)
	ReissueReader(context.Context, string) error
	ReconcileReaders(context.Context) error
}
