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
	clients  map[string]entity.MCPClient
	code     entity.MCPAuthorizationCode
	rawCode  string
	consumed bool
	token    entity.MCPToken
	visible  []entity.MCPVisibleBrain
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

func (r *mcpRepositoryMock) CreateMCPTokenPair(_ context.Context, clientID, userID string, accessExpiry, _ time.Time) (string, string, error) {
	r.token = entity.MCPToken{ID: "token-id", Type: entity.MCPTokenAccess, ClientID: clientID, UserID: userID, ExpiresAt: accessExpiry}
	return "brainhub-access", "brainhub-refresh", nil
}

func (r *mcpRepositoryMock) RotateMCPRefreshToken(context.Context, string, string, time.Time, time.Time, time.Time) (entity.MCPToken, string, string, error) {
	return entity.MCPToken{}, "rotated-access", "rotated-refresh", nil
}

func (r *mcpRepositoryMock) VerifyMCPAccessToken(_ context.Context, raw string, now time.Time) (entity.MCPToken, error) {
	if raw != "brainhub-access" || r.token.Type != entity.MCPTokenAccess || !now.Before(r.token.ExpiresAt) || r.token.RevokedAt != nil {
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

type readerRepositoryMock struct{ client *entity.ReaderClient }

func (r *readerRepositoryMock) CreateReaderClient(context.Context, entity.ReaderClient) error {
	return nil
}
func (r *readerRepositoryMock) FindReaderClientByUser(context.Context, string) (entity.ReaderClient, error) {
	if r.client == nil {
		return entity.ReaderClient{}, output_port.ErrNotFound
	}
	return *r.client, nil
}
func (r *readerRepositoryMock) ListReaderClients(context.Context) ([]entity.ReaderClient, error) {
	return nil, nil
}
func (r *readerRepositoryMock) ActivateReaderClient(context.Context, string, string, []byte, []entity.SourceID, time.Time) (entity.ReaderClient, error) {
	return entity.ReaderClient{}, nil
}
func (r *readerRepositoryMock) UpdateReaderClientScope(context.Context, string, []entity.SourceID, time.Time) (entity.ReaderClient, error) {
	return entity.ReaderClient{}, nil
}
func (r *readerRepositoryMock) MarkReaderClientOrphan(context.Context, string, *string, string) (entity.ReaderClient, error) {
	return entity.ReaderClient{}, nil
}
func (r *readerRepositoryMock) ResetReaderClient(context.Context, string) error { return nil }

type brainReaderMock struct {
	sources []entity.SourceID
	calls   int
	err     error
}

func (r *brainReaderMock) AccessToken(_ context.Context, _ string, sources []entity.SourceID) (string, error) {
	r.calls++
	r.sources = append([]entity.SourceID(nil), sources...)
	return "gbrain-reader-token", r.err
}
func (r *brainReaderMock) Reissue(context.Context, string, []entity.SourceID) error { return nil }

func TestMCPOAuthPKCEIsOneTimeAndIssuesHashedRepositoryTokens(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	readerRepository := &readerRepositoryMock{}
	reader := &brainReaderMock{}
	useCase, err := interactor.NewMCPUseCase(repository, readerRepository, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	if err != nil {
		t.Fatal(err)
	}
	client, err := useCase.IssueClient(context.Background(), "user-1")
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

func TestMCPOAuthRejectsNonASCIIAndNonS256PKCE(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	useCase, _ := interactor.NewMCPUseCase(repository, &readerRepositoryMock{}, &brainReaderMock{}, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	client, err := useCase.IssueClient(context.Background(), "user-1")
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
	useCase, _ := interactor.NewMCPUseCase(repository, &readerRepositoryMock{}, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
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
	if len(reader.sources) != 2 || reader.sources[1] != "public-other" {
		t.Fatalf("reader sources = %v", reader.sources)
	}
}

func TestMCPAuthorizationFailsClosedWhenReaderRescopeFails(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	repository.token = entity.MCPToken{Type: entity.MCPTokenAccess, UserID: "user-1", ExpiresAt: now.Add(time.Hour)}
	repository.visible = []entity.MCPVisibleBrain{{SourceID: "brain-a", Role: "owner"}}
	reader := &brainReaderMock{err: errors.New("rescope failed")}
	useCase, _ := interactor.NewMCPUseCase(repository, &readerRepositoryMock{}, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	authorized, err := useCase.AuthorizeCall(context.Background(), "brainhub-access", "get_chunks", nil)
	if err == nil || authorized.GBrainToken != "" {
		t.Fatalf("authorization = %+v %v; want no upstream token", authorized, err)
	}
}

func TestMCPSearchIsRejectedWithoutCallingGBrain(t *testing.T) {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	repository := newMCPRepositoryMock()
	repository.token = entity.MCPToken{Type: entity.MCPTokenAccess, UserID: "user-1", ExpiresAt: now.Add(time.Hour)}
	repository.visible = []entity.MCPVisibleBrain{{SourceID: "brain-a"}}
	reader := &brainReaderMock{}
	useCase, _ := interactor.NewMCPUseCase(repository, &readerRepositoryMock{}, reader, fixedClock{now}, &sequenceIDs{}, "https://brainhub.example/mcp")
	if _, err := useCase.AuthorizeCall(context.Background(), "brainhub-access", "search", nil); !errors.Is(err, input_port.ErrMCPUnsupportedTool) {
		t.Fatalf("search error = %v", err)
	}
	if reader.calls != 0 {
		t.Fatal("rejected search must not reach GBrain")
	}
}
