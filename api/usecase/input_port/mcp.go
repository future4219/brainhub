package input_port

import (
	"context"
	"errors"
	"time"

	"brainhub/domain/entity"
)

var (
	ErrOAuthInvalidScope    = errors.New("invalid OAuth scope")
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
	CodexClient   *entity.MCPClient
	CLITokens     []entity.MCPToken
	VisibleBrains []entity.MCPVisibleBrain
	Reader        *entity.ReadConnectionStatus
}

type IssuedCLIToken struct {
	Token    entity.MCPToken
	RawToken string
}

type OAuthAuthorizationRequest struct {
	Scope               string
	GrantedScope        string
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
	WriteAllowed bool
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// MCPCallAuthorization contains current Brainhub permissions, never upstream secrets.
type MCPCallAuthorization struct {
	UserID          string
	ToolName        string
	ReadableSources []entity.SourceID
	WritableSources []string
	SourceID        *string
	WriteTarget     *MCPWriteTarget
}

type MCPWriteTarget struct {
	BrainID  string
	SourceID entity.SourceID
}

type MCPUseCase interface {
	Connection(context.Context, string) (MCPConnection, error)
	IssueClient(context.Context, string, string) (entity.MCPClient, error)
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

// AuthorizedMCPRequest binds the parsed request to its server-side authorization.
// It is passed directly to the GBrain adapter, never accepted from HTTP headers.
type AuthorizedMCPRequest struct {
	Request       map[string]any
	Authorization MCPCallAuthorization
}

type mcpRequestContextKey struct{}

func WithAuthorizedMCPRequest(ctx context.Context, request AuthorizedMCPRequest) context.Context {
	return context.WithValue(ctx, mcpRequestContextKey{}, request)
}

func MCPRequestFromContext(ctx context.Context) (AuthorizedMCPRequest, bool) {
	request, ok := ctx.Value(mcpRequestContextKey{}).(AuthorizedMCPRequest)
	return request, ok
}
