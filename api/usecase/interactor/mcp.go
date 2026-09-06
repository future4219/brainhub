package interactor

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

const (
	claudeMCPClientName  = "claude-web"
	claudeRedirectURI    = "https://claude.ai/api/mcp/auth_callback"
	codexMCPClientName   = "codex"
	codexRedirectURI     = "http://127.0.0.1/callback"
	authorizationCodeTTL = 5 * time.Minute
	mcpAccessTokenTTL    = time.Hour
	mcpRefreshTokenTTL   = 30 * 24 * time.Hour
	mcpCLITokenTTL       = 90 * 24 * time.Hour
	mcpCLITokenLabelMax  = 64
)

type mcpUseCase struct {
	repository     output_port.MCPRepository
	userReadAccess output_port.UserReadAccess
	clock          output_port.Clock
	ids            output_port.IDGenerator
	mcpURL         string
}

func NewMCPUseCase(repository output_port.MCPRepository, userReadAccess output_port.UserReadAccess, clock output_port.Clock, ids output_port.IDGenerator, mcpURL string) (input_port.MCPUseCase, error) {
	if repository == nil || userReadAccess == nil || clock == nil || ids == nil || mcpURL == "" {
		return nil, errors.New("all MCP dependencies are required")
	}
	return &mcpUseCase{repository: repository, userReadAccess: userReadAccess, clock: clock, ids: ids, mcpURL: mcpURL}, nil
}

func (u *mcpUseCase) Connection(ctx context.Context, userID string) (input_port.MCPConnection, error) {
	connection := input_port.MCPConnection{}
	client, err := u.repository.FindActiveMCPClientByUserName(ctx, userID, claudeMCPClientName)
	if err == nil {
		connection.Client = &client
	} else if !errors.Is(err, output_port.ErrNotFound) {
		return connection, fmt.Errorf("find MCP client: %w", err)
	}
	codexClient, err := u.repository.FindActiveMCPClientByUserName(ctx, userID, codexMCPClientName)
	if err == nil {
		connection.CodexClient = &codexClient
	} else if !errors.Is(err, output_port.ErrNotFound) {
		return connection, fmt.Errorf("find Codex client: %w", err)
	}
	connection.VisibleBrains, err = u.repository.ListMCPVisibleBrains(ctx, userID)
	if err != nil {
		return connection, fmt.Errorf("list MCP-visible brains: %w", err)
	}
	connection.CLITokens, err = u.repository.ListMCPCLITokens(ctx, userID)
	if err != nil {
		return connection, fmt.Errorf("list MCP CLI tokens: %w", err)
	}
	connection.Reader, err = u.userReadAccess.Status(ctx, userID)
	if err != nil {
		return connection, fmt.Errorf("find read connection: %w", err)
	}
	return connection, nil
}

func (u *mcpUseCase) IssueCLIToken(ctx context.Context, userID, label string) (input_port.IssuedCLIToken, error) {
	label = strings.TrimSpace(label)
	if label == "" || utf8.RuneCountInString(label) > mcpCLITokenLabelMax {
		return input_port.IssuedCLIToken{}, input_port.ErrMCPInvalidTokenLabel
	}
	now := u.clock.Now()
	token := entity.MCPToken{
		ID: u.ids.New(), Type: entity.MCPTokenCLI, UserID: userID, Label: &label,
		ExpiresAt: now.Add(mcpCLITokenTTL), CreatedAt: now,
	}
	raw, err := u.repository.CreateMCPCLIToken(ctx, token)
	if err != nil {
		return input_port.IssuedCLIToken{}, fmt.Errorf("create MCP CLI token: %w", err)
	}
	return input_port.IssuedCLIToken{Token: token, RawToken: raw}, nil
}

func (u *mcpUseCase) RevokeCLIToken(ctx context.Context, id, userID string) error {
	if err := u.repository.RevokeMCPCLIToken(ctx, id, userID, u.clock.Now()); errors.Is(err, output_port.ErrNotFound) {
		return input_port.ErrMCPTokenNotFound
	} else if err != nil {
		return fmt.Errorf("revoke MCP CLI token: %w", err)
	}
	return nil
}

func (u *mcpUseCase) IssueClient(ctx context.Context, userID, name string) (entity.MCPClient, error) {
	redirectURI := claudeRedirectURI
	switch name {
	case "", claudeMCPClientName:
		name = claudeMCPClientName
	case codexMCPClientName:
		redirectURI = codexRedirectURI
	default:
		return entity.MCPClient{}, input_port.ErrOAuthInvalidClient
	}
	client, err := u.repository.FindActiveMCPClientByUserName(ctx, userID, name)
	if err == nil {
		return client, nil
	}
	if !errors.Is(err, output_port.ErrNotFound) {
		return entity.MCPClient{}, fmt.Errorf("find MCP client: %w", err)
	}
	client = entity.MCPClient{
		ID: u.ids.New(), UserID: userID, Name: name,
		RedirectURIs: []string{redirectURI}, CreatedAt: u.clock.Now(),
	}
	if name == codexMCPClientName {
		// Codex 0.149 also binds callbacks to the MCP URL using a SHA-256 prefix.
		// Register that exact path alongside the issuer-bound /callback variant.
		digest := sha256.Sum256([]byte(u.mcpURL))
		client.RedirectURIs = append(client.RedirectURIs, codexRedirectURI+"/"+base64.RawURLEncoding.EncodeToString(digest[:9]))
	}
	if err := u.repository.CreateMCPClient(ctx, client); errors.Is(err, output_port.ErrConflict) {
		return u.repository.FindActiveMCPClientByUserName(ctx, userID, name)
	} else if err != nil {
		return entity.MCPClient{}, fmt.Errorf("create MCP client: %w", err)
	}
	return client, nil
}

func (u *mcpUseCase) ValidateAuthorization(ctx context.Context, input input_port.OAuthAuthorizationRequest, userID string) (input_port.OAuthAuthorization, error) {
	client, err := u.repository.FindMCPClientByID(ctx, input.ClientID)
	if errors.Is(err, output_port.ErrNotFound) || (err == nil && client.RevokedAt != nil) {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidClient
	}
	if err != nil {
		return input_port.OAuthAuthorization{}, fmt.Errorf("find OAuth client: %w", err)
	}
	if userID != "" && client.UserID != userID {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthAccessDenied
	}
	if input.ResponseType != "code" || input.CodeChallengeMethod != "S256" || !validPKCEValue(input.CodeChallenge) {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidRequest
	}
	if !client.AllowsRedirectURI(input.RedirectURI) {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidRequest
	}
	if input.Resource == "" {
		input.Resource = u.mcpURL
	}
	if input.Resource != u.mcpURL {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidRequest
	}
	scope, ok := normalizeMCPScope(input.Scope)
	if !ok {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidScope
	}
	input.Scope = scope
	return input_port.OAuthAuthorization{Client: client, Input: input}, nil
}

func (u *mcpUseCase) ApproveAuthorization(ctx context.Context, input input_port.OAuthAuthorizationRequest, userID string) (string, error) {
	authorization, err := u.ValidateAuthorization(ctx, input, userID)
	if err != nil {
		return "", err
	}
	if userID == "" {
		return "", input_port.ErrOAuthAccessDenied
	}
	// Missing consent never grants write (including old consent tabs).
	granted := input.GrantedScope
	if granted == "" {
		granted = "read"
	}
	granted, valid := normalizeMCPScope(granted)
	if !valid || (granted == "read write" && authorization.Input.Scope != "read write") {
		return "", input_port.ErrOAuthInvalidScope
	}
	now := u.clock.Now()
	return u.repository.CreateMCPAuthorizationCode(ctx, entity.MCPAuthorizationCode{
		ID: u.ids.New(), ClientID: authorization.Client.ID, UserID: userID, WriteAllowed: granted == "read write",
		RedirectURI: authorization.Input.RedirectURI, CodeChallenge: authorization.Input.CodeChallenge,
		Resource: authorization.Input.Resource, ExpiresAt: now.Add(authorizationCodeTTL), CreatedAt: now,
	})
}

func (u *mcpUseCase) ExchangeAuthorizationCode(ctx context.Context, clientID, rawCode, redirectURI, verifier string) (input_port.OAuthTokenPair, error) {
	client, err := u.activeClient(ctx, clientID)
	if err != nil {
		return input_port.OAuthTokenPair{}, err
	}
	if !client.AllowsRedirectURI(redirectURI) || !validPKCEValue(verifier) {
		return input_port.OAuthTokenPair{}, input_port.ErrOAuthInvalidGrant
	}
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	code, err := u.repository.ConsumeMCPAuthorizationCode(ctx, rawCode, clientID, redirectURI, challenge, u.clock.Now())
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.OAuthTokenPair{}, input_port.ErrOAuthInvalidGrant
	}
	if err != nil {
		return input_port.OAuthTokenPair{}, fmt.Errorf("consume authorization code: %w", err)
	}
	return u.createTokenPair(ctx, code.ClientID, code.UserID, code.WriteAllowed)
}

func (u *mcpUseCase) RefreshAccessToken(ctx context.Context, clientID, rawRefreshToken string) (input_port.OAuthTokenPair, error) {
	if _, err := u.activeClient(ctx, clientID); err != nil {
		return input_port.OAuthTokenPair{}, err
	}
	now := u.clock.Now()
	token, access, refresh, err := u.repository.RotateMCPRefreshToken(
		ctx, rawRefreshToken, clientID, now, now.Add(mcpAccessTokenTTL), now.Add(mcpRefreshTokenTTL),
	)
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.OAuthTokenPair{}, input_port.ErrOAuthInvalidGrant
	}
	if err != nil {
		return input_port.OAuthTokenPair{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	return input_port.OAuthTokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: mcpAccessTokenTTL, WriteAllowed: token.WriteAllowed}, nil
}

func (u *mcpUseCase) RevokeToken(ctx context.Context, clientID, rawToken string) error {
	if _, err := u.activeClient(ctx, clientID); err != nil {
		return err
	}
	return u.repository.RevokeMCPToken(ctx, rawToken, clientID, u.clock.Now())
}

func (u *mcpUseCase) AuthorizeCall(ctx context.Context, rawToken, toolName string, requestedSource *string) (input_port.MCPCallAuthorization, error) {
	token, err := u.repository.VerifyMCPAccessToken(ctx, rawToken, u.clock.Now())
	if errors.Is(err, output_port.ErrTokenExpired) {
		return input_port.MCPCallAuthorization{}, input_port.ErrMCPTokenExpired
	}
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.MCPCallAuthorization{}, input_port.ErrMCPUnauthorized
	}
	if err != nil {
		return input_port.MCPCallAuthorization{}, fmt.Errorf("verify MCP access token: %w", err)
	}
	visible, err := u.repository.ListMCPVisibleBrains(ctx, token.UserID)
	if err != nil {
		return input_port.MCPCallAuthorization{}, fmt.Errorf("list MCP-visible brains: %w", err)
	}
	sources := make([]entity.SourceID, len(visible))
	allowed := make(map[string]struct{}, len(visible))
	for i, brain := range visible {
		sources[i] = brain.SourceID
		allowed[brain.SourceID.String()] = struct{}{}
	}
	// Only page writes get an explicit write target. Source management and all
	// other operations retain read-only permissions.
	writable := make([]string, 0)
	if token.WriteAllowed {
		for _, brain := range visible {
			if brain.CanWrite() {
				writable = append(writable, brain.SourceID.String())
			}
		}
	}
	if toolName == "put_page" {
		if !token.WriteAllowed || requestedSource == nil || *requestedSource == "" || *requestedSource == "__all__" {
			return input_port.MCPCallAuthorization{}, input_port.ErrMCPForbidden
		}
		for _, brain := range visible {
			if brain.SourceID.String() == *requestedSource && brain.CanWrite() {
				return input_port.MCPCallAuthorization{
					UserID: token.UserID, ToolName: toolName,
					WriteTarget: &input_port.MCPWriteTarget{BrainID: brain.ID, SourceID: brain.SourceID},
				}, nil
			}
		}
		return input_port.MCPCallAuthorization{}, input_port.ErrMCPForbidden
	}
	var injected *string
	if toolName == "query" || toolName == "list_pages" || toolName == "get_page" {
		value := "__all__"
		if requestedSource != nil && *requestedSource != "" && *requestedSource != "__all__" {
			if _, ok := allowed[*requestedSource]; !ok {
				return input_port.MCPCallAuthorization{}, input_port.ErrMCPForbidden
			}
			value = *requestedSource
		}
		injected = &value
	}
	return input_port.MCPCallAuthorization{
		UserID: token.UserID, ToolName: toolName, ReadableSources: sources,
		SourceID: injected, WritableSources: writable,
	}, nil
}

func (u *mcpUseCase) ReissueReader(ctx context.Context, userID string) error {
	visible, err := u.repository.ListMCPVisibleBrains(ctx, userID)
	if err != nil {
		return err
	}
	return u.userReadAccess.Reissue(ctx, userID, sourceIDs(visible))
}

func (u *mcpUseCase) ReconcileReaders(ctx context.Context) error {
	users, err := u.userReadAccess.ConnectedUsers(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, userID := range users {
		visible, err := u.repository.ListMCPVisibleBrains(ctx, userID)
		if err == nil && len(visible) > 0 {
			err = u.userReadAccess.Prepare(ctx, userID, sourceIDs(visible))
		}
		if err != nil && !errors.Is(err, output_port.ErrReaderNeedsReissue) {
			failures = append(failures, fmt.Errorf("%s: %w", userID, err))
		}
	}
	return errors.Join(failures...)
}

func (u *mcpUseCase) activeClient(ctx context.Context, clientID string) (entity.MCPClient, error) {
	client, err := u.repository.FindMCPClientByID(ctx, clientID)
	if errors.Is(err, output_port.ErrNotFound) || (err == nil && client.RevokedAt != nil) {
		return entity.MCPClient{}, input_port.ErrOAuthInvalidClient
	}
	if err != nil {
		return entity.MCPClient{}, fmt.Errorf("find OAuth client: %w", err)
	}
	return client, nil
}

func (u *mcpUseCase) createTokenPair(ctx context.Context, clientID, userID string, writeAllowed bool) (input_port.OAuthTokenPair, error) {
	now := u.clock.Now()
	access, refresh, err := u.repository.CreateMCPTokenPair(ctx, clientID, userID, writeAllowed, now.Add(mcpAccessTokenTTL), now.Add(mcpRefreshTokenTTL))
	if err != nil {
		return input_port.OAuthTokenPair{}, fmt.Errorf("create MCP token pair: %w", err)
	}
	return input_port.OAuthTokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: mcpAccessTokenTTL, WriteAllowed: writeAllowed}, nil
}

func validPKCEValue(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for i := range len(value) {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '.' || c == '_' || c == '~' {
			continue
		}
		return false
	}
	return true
}

func sourceIDs(brains []entity.MCPVisibleBrain) []entity.SourceID {
	sources := make([]entity.SourceID, len(brains))
	for i, brain := range brains {
		sources[i] = brain.SourceID
	}
	return sources
}

func normalizeMCPScope(scope string) (string, bool) {
	if scope == "" {
		return "read write", true
	}
	read, write := false, false
	for _, value := range strings.Fields(scope) {
		switch value {
		case "read":
			read = true
		case "write":
			write = true
		default:
			return "", false
		}
	}
	if !read {
		return "", false
	}
	return entity.MCPScope(write), true
}
