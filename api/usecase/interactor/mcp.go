package interactor

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

const (
	claudeMCPClientName  = "claude-web"
	claudeRedirectURI    = "https://claude.ai/api/mcp/auth_callback"
	authorizationCodeTTL = 5 * time.Minute
	mcpAccessTokenTTL    = time.Hour
	mcpRefreshTokenTTL   = 30 * 24 * time.Hour
)

type mcpUseCase struct {
	repository output_port.MCPRepository
	readers    output_port.ReaderClientRepository
	reader     output_port.BrainReader
	clock      output_port.Clock
	ids        output_port.IDGenerator
	mcpURL     string
}

func NewMCPUseCase(repository output_port.MCPRepository, readers output_port.ReaderClientRepository, reader output_port.BrainReader, clock output_port.Clock, ids output_port.IDGenerator, mcpURL string) (input_port.MCPUseCase, error) {
	if repository == nil || readers == nil || reader == nil || clock == nil || ids == nil || mcpURL == "" {
		return nil, errors.New("all MCP dependencies are required")
	}
	return &mcpUseCase{repository: repository, readers: readers, reader: reader, clock: clock, ids: ids, mcpURL: mcpURL}, nil
}

func (u *mcpUseCase) Connection(ctx context.Context, userID string) (input_port.MCPConnection, error) {
	connection := input_port.MCPConnection{}
	client, err := u.repository.FindActiveMCPClientByUserName(ctx, userID, claudeMCPClientName)
	if err == nil {
		connection.Client = &client
	} else if !errors.Is(err, output_port.ErrNotFound) {
		return connection, fmt.Errorf("find MCP client: %w", err)
	}
	connection.VisibleBrains, err = u.repository.ListMCPVisibleBrains(ctx, userID)
	if err != nil {
		return connection, fmt.Errorf("list MCP-visible brains: %w", err)
	}
	reader, err := u.readers.FindReaderClientByUser(ctx, userID)
	if err == nil {
		connection.Reader = &reader
	} else if !errors.Is(err, output_port.ErrNotFound) {
		return connection, fmt.Errorf("find reader client: %w", err)
	}
	return connection, nil
}

func (u *mcpUseCase) IssueClient(ctx context.Context, userID string) (entity.MCPClient, error) {
	client, err := u.repository.FindActiveMCPClientByUserName(ctx, userID, claudeMCPClientName)
	if err == nil {
		return client, nil
	}
	if !errors.Is(err, output_port.ErrNotFound) {
		return entity.MCPClient{}, fmt.Errorf("find MCP client: %w", err)
	}
	client = entity.MCPClient{
		ID: u.ids.New(), UserID: userID, Name: claudeMCPClientName,
		RedirectURIs: []string{claudeRedirectURI}, CreatedAt: u.clock.Now(),
	}
	if err := u.repository.CreateMCPClient(ctx, client); errors.Is(err, output_port.ErrConflict) {
		return u.repository.FindActiveMCPClientByUserName(ctx, userID, claudeMCPClientName)
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
	if !slices.Contains(client.RedirectURIs, input.RedirectURI) {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidRequest
	}
	if input.Resource == "" {
		input.Resource = u.mcpURL
	}
	if input.Resource != u.mcpURL {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidRequest
	}
	return input_port.OAuthAuthorization{Client: client, Input: input}, nil
}

func (u *mcpUseCase) ApproveAuthorization(ctx context.Context, input input_port.OAuthAuthorizationRequest, userID string) (string, error) {
	authorization, err := u.ValidateAuthorization(ctx, input, userID)
	if err != nil {
		return "", err
	}
	now := u.clock.Now()
	return u.repository.CreateMCPAuthorizationCode(ctx, entity.MCPAuthorizationCode{
		ID: u.ids.New(), ClientID: authorization.Client.ID, UserID: userID,
		RedirectURI: authorization.Input.RedirectURI, CodeChallenge: authorization.Input.CodeChallenge,
		Resource: authorization.Input.Resource, ExpiresAt: now.Add(authorizationCodeTTL), CreatedAt: now,
	})
}

func (u *mcpUseCase) ExchangeAuthorizationCode(ctx context.Context, clientID, rawCode, redirectURI, verifier string) (input_port.OAuthTokenPair, error) {
	client, err := u.activeClient(ctx, clientID)
	if err != nil {
		return input_port.OAuthTokenPair{}, err
	}
	if !slices.Contains(client.RedirectURIs, redirectURI) || !validPKCEValue(verifier) {
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
	return u.createTokenPair(ctx, code.ClientID, code.UserID)
}

func (u *mcpUseCase) RefreshAccessToken(ctx context.Context, clientID, rawRefreshToken string) (input_port.OAuthTokenPair, error) {
	if _, err := u.activeClient(ctx, clientID); err != nil {
		return input_port.OAuthTokenPair{}, err
	}
	now := u.clock.Now()
	_, access, refresh, err := u.repository.RotateMCPRefreshToken(
		ctx, rawRefreshToken, clientID, now, now.Add(mcpAccessTokenTTL), now.Add(mcpRefreshTokenTTL),
	)
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.OAuthTokenPair{}, input_port.ErrOAuthInvalidGrant
	}
	if err != nil {
		return input_port.OAuthTokenPair{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	return input_port.OAuthTokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: mcpAccessTokenTTL}, nil
}

func (u *mcpUseCase) RevokeToken(ctx context.Context, clientID, rawToken string) error {
	if _, err := u.activeClient(ctx, clientID); err != nil {
		return err
	}
	return u.repository.RevokeMCPToken(ctx, rawToken, clientID, u.clock.Now())
}

func (u *mcpUseCase) AuthorizeCall(ctx context.Context, rawToken, toolName string, requestedSource *string) (input_port.MCPCallAuthorization, error) {
	token, err := u.repository.VerifyMCPAccessToken(ctx, rawToken, u.clock.Now())
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
	if toolName == "search" {
		return input_port.MCPCallAuthorization{}, input_port.ErrMCPUnsupportedTool
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
	upstreamToken, err := u.reader.AccessToken(ctx, token.UserID, sources)
	if err != nil {
		return input_port.MCPCallAuthorization{}, fmt.Errorf("prepare GBrain reader: %w", err)
	}
	return input_port.MCPCallAuthorization{GBrainToken: upstreamToken, SourceID: injected}, nil
}

func (u *mcpUseCase) ReissueReader(ctx context.Context, userID string) error {
	visible, err := u.repository.ListMCPVisibleBrains(ctx, userID)
	if err != nil {
		return err
	}
	return u.reader.Reissue(ctx, userID, sourceIDs(visible))
}

func (u *mcpUseCase) ReconcileReaders(ctx context.Context) error {
	clients, err := u.readers.ListReaderClients(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, client := range clients {
		visible, err := u.repository.ListMCPVisibleBrains(ctx, client.UserID)
		if err == nil && len(visible) > 0 {
			_, err = u.reader.AccessToken(ctx, client.UserID, sourceIDs(visible))
		}
		if err != nil && !errors.Is(err, output_port.ErrReaderNeedsReissue) {
			failures = append(failures, fmt.Errorf("%s: %w", client.UserID, err))
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

func (u *mcpUseCase) createTokenPair(ctx context.Context, clientID, userID string) (input_port.OAuthTokenPair, error) {
	now := u.clock.Now()
	access, refresh, err := u.repository.CreateMCPTokenPair(ctx, clientID, userID, now.Add(mcpAccessTokenTTL), now.Add(mcpRefreshTokenTTL))
	if err != nil {
		return input_port.OAuthTokenPair{}, fmt.Errorf("create MCP token pair: %w", err)
	}
	return input_port.OAuthTokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: mcpAccessTokenTTL}, nil
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
