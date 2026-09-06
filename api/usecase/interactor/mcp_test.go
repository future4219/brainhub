package interactor_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/interactor"
	"brainhub/usecase/output_port"
)

type mcpRepositoryMock struct {
	clients   map[string]entity.MCPClient
	code      entity.MCPAuthorizationCode
	rawCode   string
	consumed  bool
	token     entity.MCPToken
	cliTokens []entity.MCPToken
	visible   []entity.MCPVisibleBrain
}

func newMCPRepositoryMock() *mcpRepositoryMock {
	return &mcpRepositoryMock{clients: make(map[string]entity.MCPClient), rawCode: "raw-authorization-code"}
}

func (r *mcpRepositoryMock) CreateMCPClient(_ context.Context, client entity.MCPClient) error {
	for _, existing := range r.clients {
		if existing.UserID == client.UserID && existing.Name == client.Name && existing.RevokedAt == nil {
			return output_port.ErrConflict
		}
	}
	r.clients[client.ID] = client
	return nil
}

func (r *mcpRepositoryMock) FindMCPClientByID(_ context.Context, id string) (entity.MCPClient, error) {
	client, ok := r.clients[id]
	if !ok {
		return entity.MCPClient{}, output_port.ErrNotFound
	}
	return client, nil
}

func (r *mcpRepositoryMock) FindActiveMCPClientByUserName(_ context.Context, userID, name string) (entity.MCPClient, error) {
	for _, client := range r.clients {
		if client.UserID == userID && client.Name == name && client.RevokedAt == nil {
			return client, nil
		}
	}
	return entity.MCPClient{}, output_port.ErrNotFound
}

func (r *mcpRepositoryMock) CreateMCPAuthorizationCode(_ context.Context, code entity.MCPAuthorizationCode) (string, error) {
	r.code = code
	r.consumed = false
	return r.rawCode, nil
}

func (r *mcpRepositoryMock) ConsumeMCPAuthorizationCode(_ context.Context, raw, clientID, redirectURI, challenge string, now time.Time) (entity.MCPAuthorizationCode, error) {
	if r.consumed || raw != r.rawCode || clientID != r.code.ClientID || redirectURI != r.code.RedirectURI || challenge != r.code.CodeChallenge || !now.Before(r.code.ExpiresAt) {
		return entity.MCPAuthorizationCode{}, output_port.ErrNotFound
	}
	r.consumed = true
	return r.code, nil
}

func (r *mcpRepositoryMock) CreateMCPTokenPair(_ context.Context, clientID, userID string, writeAllowed bool, accessExpiry, _ time.Time) (string, string, error) {
	r.token = entity.MCPToken{ID: "token-id", WriteAllowed: writeAllowed, Type: entity.MCPTokenAccess, ClientID: &clientID, UserID: userID, ExpiresAt: accessExpiry}
	return "brainhub-access", "brainhub-refresh", nil
}

func (r *mcpRepositoryMock) CreateMCPCLIToken(_ context.Context, token entity.MCPToken) (string, error) {
	r.token = token
	r.cliTokens = append(r.cliTokens, token)
	return "brainhub-cli", nil
}

func (r *mcpRepositoryMock) ListMCPCLITokens(_ context.Context, userID string) ([]entity.MCPToken, error) {
	tokens := make([]entity.MCPToken, 0)
	for _, token := range r.cliTokens {
		if token.UserID == userID && token.RevokedAt == nil {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (r *mcpRepositoryMock) RevokeMCPCLIToken(_ context.Context, id, userID string, now time.Time) error {
	for i := range r.cliTokens {
		if r.cliTokens[i].ID == id && r.cliTokens[i].UserID == userID && r.cliTokens[i].RevokedAt == nil {
			r.cliTokens[i].RevokedAt = &now
			if r.token.ID == id {
				r.token.RevokedAt = &now
			}
			return nil
		}
	}
	return output_port.ErrNotFound
}

func (r *mcpRepositoryMock) RotateMCPRefreshToken(context.Context, string, string, time.Time, time.Time, time.Time) (entity.MCPToken, string, string, error) {
	return r.token, "rotated-access", "rotated-refresh", nil
}

func (r *mcpRepositoryMock) VerifyMCPAccessToken(_ context.Context, raw string, now time.Time) (entity.MCPToken, error) {
	validRaw := raw == "brainhub-access" && r.token.Type == entity.MCPTokenAccess || raw == "brainhub-cli" && r.token.Type == entity.MCPTokenCLI
	if validRaw && r.token.Type == entity.MCPTokenCLI && !now.Before(r.token.ExpiresAt) && r.token.RevokedAt == nil {
		return entity.MCPToken{}, output_port.ErrTokenExpired
	}
	if !validRaw || !now.Before(r.token.ExpiresAt) || r.token.RevokedAt != nil {
		return entity.MCPToken{}, output_port.ErrNotFound
	}
	return r.token, nil
}

func (r *mcpRepositoryMock) RevokeMCPToken(context.Context, string, string, time.Time) error {
	return nil
}

func (r *mcpRepositoryMock) ListMCPVisibleBrains(context.Context, string) ([]entity.MCPVisibleBrain, error) {
	return append([]entity.MCPVisibleBrain(nil), r.visible...), nil
}

type brainReaderMock struct {
	users   []string
	sources []entity.SourceID
	calls   int
	err     error
}

func (r *brainReaderMock) AccessToken(_ context.Context, _ string, sources []entity.SourceID) (string, error) {
	r.calls++
	r.sources = append([]entity.SourceID(nil), sources...)
	return "gbrain-reader-token", r.err
}
func (r *brainReaderMock) Status(context.Context, string) (*entity.ReadConnectionStatus, error) {
	return nil, nil
}
func (r *brainReaderMock) ConnectedUsers(context.Context) ([]string, error) { return r.users, nil }
func (r *brainReaderMock) Prepare(ctx context.Context, user string, sources []entity.SourceID) error {
	_, err := r.AccessToken(ctx, user, sources)
	return err
}
func (r *brainReaderMock) Reissue(context.Context, string, []entity.SourceID) error { return nil }

func TestMCPOAuthPKCEIsOneTimeAndIssuesHashedRepositoryTokens(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	reader := &brainReaderMock{}
	useCase, err := interactor.NewMCPUseCase(repository, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	if err != nil {
		t.Fatal(err)
	}
	client, err := useCase.IssueClient(context.Background(), "user-1", "claude-web")
	if err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("a", 43)
	digest := sha256.Sum256([]byte(verifier))
	request := input_port.OAuthAuthorizationRequest{
		ClientID: client.ID, RedirectURI: "https://claude.ai/api/mcp/auth_callback", ResponseType: "code",
		CodeChallenge: base64.RawURLEncoding.EncodeToString(digest[:]), CodeChallengeMethod: "S256",
		Resource: "https://brainhub.example/mcp",
	}
	code, err := useCase.ApproveAuthorization(context.Background(), request, "user-1")
	if err != nil || code != repository.rawCode {
		t.Fatalf("approve = %q %v", code, err)
	}
	pair, err := useCase.ExchangeAuthorizationCode(context.Background(), client.ID, code, request.RedirectURI, verifier)
	if err != nil || pair.AccessToken != "brainhub-access" || pair.RefreshToken != "brainhub-refresh" {
		t.Fatalf("exchange = %+v %v", pair, err)
	}
	if _, err := useCase.ExchangeAuthorizationCode(context.Background(), client.ID, code, request.RedirectURI, verifier); !errors.Is(err, input_port.ErrOAuthInvalidGrant) {
		t.Fatalf("second exchange error = %v; want invalid_grant", err)
	}
}

func TestMCPCLITokenLifecycleUsesNinetyDayExpiry(t *testing.T) {
	now := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	useCase, _ := interactor.NewMCPUseCase(repository, &brainReaderMock{}, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	issued, err := useCase.IssueCLIToken(context.Background(), "user-1", "  codex  ")
	if err != nil || issued.RawToken != "brainhub-cli" || issued.Token.Type != entity.MCPTokenCLI {
		t.Fatalf("issued = %+v %v", issued, err)
	}
	if issued.Token.Label == nil || *issued.Token.Label != "codex" || !issued.Token.ExpiresAt.Equal(now.Add(90*24*time.Hour)) {
		t.Fatalf("token metadata = %+v", issued.Token)
	}
	connection, err := useCase.Connection(context.Background(), "user-1")
	if err != nil || len(connection.CLITokens) != 1 || connection.CLITokens[0].ID != issued.Token.ID {
		t.Fatalf("connection tokens = %+v %v", connection.CLITokens, err)
	}
	if err := useCase.RevokeCLIToken(context.Background(), issued.Token.ID, "user-2"); !errors.Is(err, input_port.ErrMCPTokenNotFound) {
		t.Fatalf("foreign revoke error = %v", err)
	}
	if err := useCase.RevokeCLIToken(context.Background(), issued.Token.ID, "user-1"); err != nil {
		t.Fatal(err)
	}
	connection, _ = useCase.Connection(context.Background(), "user-1")
	if len(connection.CLITokens) != 0 {
		t.Fatalf("revoked tokens = %+v", connection.CLITokens)
	}
	for _, label := range []string{"   ", strings.Repeat("あ", 65)} {
		if _, err := useCase.IssueCLIToken(context.Background(), "user-1", label); !errors.Is(err, input_port.ErrMCPInvalidTokenLabel) {
			t.Fatalf("label %q error = %v", label, err)
		}
	}
}

func TestMCPOAuthRejectsNonASCIIAndNonS256PKCE(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	useCase, _ := interactor.NewMCPUseCase(repository, &brainReaderMock{}, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	client, err := useCase.IssueClient(context.Background(), "user-1", "claude-web")
	if err != nil {
		t.Fatal(err)
	}
	base := input_port.OAuthAuthorizationRequest{
		ClientID: client.ID, RedirectURI: "https://claude.ai/api/mcp/auth_callback", ResponseType: "code",
		CodeChallenge: strings.Repeat("a", 43), CodeChallengeMethod: "S256", Resource: "https://brainhub.example/mcp",
	}
	base.CodeChallenge = strings.Repeat("é", 43)
	if _, err := useCase.ValidateAuthorization(context.Background(), base, "user-1"); !errors.Is(err, input_port.ErrOAuthInvalidRequest) {
		t.Fatalf("non-ASCII PKCE error = %v", err)
	}
	base.CodeChallenge = strings.Repeat("a", 43)
	base.CodeChallengeMethod = "plain"
	if _, err := useCase.ValidateAuthorization(context.Background(), base, "user-1"); !errors.Is(err, input_port.ErrOAuthInvalidRequest) {
		t.Fatalf("plain PKCE error = %v", err)
	}
}

func TestMCPAuthorizationUsesCurrentVisibilityAndRejectsForeignSourceBeforeGBrain(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	repository.token = entity.MCPToken{Type: entity.MCPTokenAccess, UserID: "user-1", ExpiresAt: now.Add(time.Hour)}
	repository.visible = []entity.MCPVisibleBrain{
		{SourceID: "private-own", Role: "owner"},
		{SourceID: "public-other", Role: "public"},
	}
	reader := &brainReaderMock{}
	useCase, _ := interactor.NewMCPUseCase(repository, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	foreign := "private-foreign"
	if _, err := useCase.AuthorizeCall(context.Background(), "brainhub-access", "get_page", &foreign); !errors.Is(err, input_port.ErrMCPForbidden) {
		t.Fatalf("foreign source error = %v; want forbidden", err)
	}
	if reader.calls != 0 {
		t.Fatal("GBrain reader must not be reached for a foreign source")
	}
	authorized, err := useCase.AuthorizeCall(context.Background(), "brainhub-access", "query", nil)
	if err != nil || authorized.SourceID == nil || *authorized.SourceID != "__all__" {
		t.Fatalf("authorized = %+v %v", authorized, err)
	}
	if len(authorized.ReadableSources) != 2 || authorized.ReadableSources[1] != "public-other" {
		t.Fatalf("reader sources = %v", authorized.ReadableSources)
	}
}

func TestMCPCLITokenMatchesOAuthAuthorizationAndReflectsMembershipRevokeImmediately(t *testing.T) {
	now := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	visible := []entity.MCPVisibleBrain{
		{SourceID: "private-own", Role: "owner"},
		{SourceID: "public-other", Role: "public"},
	}
	for _, tc := range []struct {
		name  string
		type_ entity.MCPTokenType
		raw   string
	}{
		{name: "oauth", type_: entity.MCPTokenAccess, raw: "brainhub-access"},
		{name: "cli", type_: entity.MCPTokenCLI, raw: "brainhub-cli"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository := newMCPRepositoryMock()
			repository.token = entity.MCPToken{Type: tc.type_, UserID: "user-1", ExpiresAt: now.Add(time.Hour)}
			repository.visible = append([]entity.MCPVisibleBrain(nil), visible...)
			reader := &brainReaderMock{}
			useCase, _ := interactor.NewMCPUseCase(repository, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
			authorized, err := useCase.AuthorizeCall(context.Background(), tc.raw, "list_pages", nil)
			if err != nil || authorized.SourceID == nil || *authorized.SourceID != "__all__" || authorized.UserID != "user-1" {
				t.Fatalf("authorization = %+v %v", authorized, err)
			}
			if len(authorized.ReadableSources) != 2 || authorized.ReadableSources[0] != "private-own" || authorized.ReadableSources[1] != "public-other" {
				t.Fatalf("reader sources = %v", authorized.ReadableSources)
			}
			if tc.type_ == entity.MCPTokenCLI {
				repository.visible = visible[1:]
				private := "private-own"
				if _, err := useCase.AuthorizeCall(context.Background(), tc.raw, "list_pages", &private); !errors.Is(err, input_port.ErrMCPForbidden) {
					t.Fatalf("revoked membership error = %v", err)
				}
			}
		})
	}
}

func TestMCPExpiredCLITokenHasSpecificUnauthorizedReason(t *testing.T) {
	now := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	repository.token = entity.MCPToken{Type: entity.MCPTokenCLI, UserID: "user-1", ExpiresAt: now}
	useCase, _ := interactor.NewMCPUseCase(repository, &brainReaderMock{}, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	if _, err := useCase.AuthorizeCall(context.Background(), "brainhub-cli", "list_pages", nil); !errors.Is(err, input_port.ErrMCPTokenExpired) {
		t.Fatalf("expired token error = %v", err)
	}
}

func TestMCPSearchUsesCurrentVisibilityWithoutSourceInjection(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	repository.token = entity.MCPToken{Type: entity.MCPTokenAccess, UserID: "user-1", ExpiresAt: now.Add(time.Hour)}
	repository.visible = []entity.MCPVisibleBrain{{SourceID: "brain-a"}, {SourceID: "brain-b"}}
	reader := &brainReaderMock{}
	useCase, _ := interactor.NewMCPUseCase(repository, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	authorized, err := useCase.AuthorizeCall(context.Background(), "brainhub-access", "search", nil)
	if err != nil || authorized.UserID != "user-1" || authorized.SourceID != nil {
		t.Fatalf("authorization = %+v %v", authorized, err)
	}
	if reader.calls != 0 || len(authorized.ReadableSources) != 2 || authorized.ReadableSources[0] != "brain-a" || authorized.ReadableSources[1] != "brain-b" {
		t.Fatalf("reader calls/sources = %d %v", reader.calls, authorized.ReadableSources)
	}

	repository.visible = repository.visible[:1]
	authorized, err = useCase.AuthorizeCall(context.Background(), "brainhub-access", "search", nil)
	if err != nil {
		t.Fatal(err)
	}
	if reader.calls != 0 || len(authorized.ReadableSources) != 1 || authorized.ReadableSources[0] != "brain-a" {
		t.Fatalf("reader calls/sources after revoke = %d %v", reader.calls, authorized.ReadableSources)
	}
}

type mcpWriterMock struct {
	calls    int
	brainID  string
	sourceID entity.SourceID
	err      error
}

func (w *mcpWriterMock) AccessToken(_ context.Context, brainID string, source entity.SourceID) (string, error) {
	w.calls++
	w.brainID = brainID
	w.sourceID = source
	return "gbrain-writer-token", w.err
}

func TestReaderReconciliationPreparesCurrentVisibilityAndPreservesRecovery(t *testing.T) {
	repository := newMCPRepositoryMock()
	repository.visible = []entity.MCPVisibleBrain{{SourceID: "current"}}
	reader := &brainReaderMock{users: []string{"user-1"}}
	useCase, err := interactor.NewMCPUseCase(repository, reader, fixedClock{time.Now()}, &sequenceIDs{}, "https://brainhub.example/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if err := useCase.ReconcileReaders(context.Background()); err != nil || reader.calls != 1 || len(reader.sources) != 1 || reader.sources[0] != "current" {
		t.Fatalf("prepare = %v %v", reader.sources, err)
	}
	reader.err = output_port.ErrReaderNeedsReissue
	if err := useCase.ReconcileReaders(context.Background()); err != nil {
		t.Fatalf("manual recovery should remain pending: %v", err)
	}
	reader.err = errors.New("rescope unavailable")
	if err := useCase.ReconcileReaders(context.Background()); !errors.Is(err, reader.err) {
		t.Fatalf("lost startup failure: %v", err)
	}
	repository.visible = nil
	calls := reader.calls
	if err := useCase.ReconcileReaders(context.Background()); err != nil || reader.calls != calls {
		t.Fatal("empty visibility must not provision a connection at startup")
	}
}
