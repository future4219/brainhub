package gbrain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

type readerAdminStub struct {
	rescopeErr error
	rescopes   [][]string
}

func (a *readerAdminStub) RegisterClient(context.Context, output_port.RegisterGBrainClientInput) (output_port.RegisteredGBrainClient, error) {
	return output_port.RegisteredGBrainClient{ID: "reader-id", Secret: "reader-secret"}, nil
}
func (a *readerAdminStub) RescopeClient(_ context.Context, _ string, sources []string) error {
	a.rescopes = append(a.rescopes, append([]string(nil), sources...))
	return a.rescopeErr
}
func (a *readerAdminStub) RevokeClient(context.Context, string) error { return nil }

type readerClientRepositoryStub struct {
	client      entity.ReaderClient
	orphanCalls int
	findErr     error
}

func (r *readerClientRepositoryStub) CreateReaderClient(_ context.Context, client entity.ReaderClient) error {
	r.client = client
	return nil
}
func (r *readerClientRepositoryStub) FindReaderClientByUser(context.Context, string) (entity.ReaderClient, error) {
	return r.client, r.findErr
}
func (r *readerClientRepositoryStub) ListReaderClients(context.Context) ([]entity.ReaderClient, error) {
	return []entity.ReaderClient{r.client}, nil
}
func (r *readerClientRepositoryStub) ActivateReaderClient(_ context.Context, _ string, clientID string, ciphertext []byte, sources []entity.SourceID, now time.Time) (entity.ReaderClient, error) {
	r.client.GBrainClientID = &clientID
	r.client.ClientSecretCiphertext = ciphertext
	r.client.FederatedRead = append([]entity.SourceID(nil), sources...)
	r.client.State = entity.ReaderClientStateActive
	r.client.IssuedAt = &now
	return r.client, nil
}
func (r *readerClientRepositoryStub) UpdateReaderClientScope(_ context.Context, _ string, sources []entity.SourceID, now time.Time) (entity.ReaderClient, error) {
	r.client.FederatedRead = append([]entity.SourceID(nil), sources...)
	r.client.LastVerifiedAt = &now
	return r.client, nil
}
func (r *readerClientRepositoryStub) MarkReaderClientOrphan(_ context.Context, _ string, clientID *string, reason string) (entity.ReaderClient, error) {
	r.orphanCalls++
	r.client.GBrainClientID = clientID
	r.client.ClientSecretCiphertext = nil
	r.client.State = entity.ReaderClientStateOrphan
	r.client.StateReason = reason
	return r.client, nil
}
func (r *readerClientRepositoryStub) ResetReaderClient(context.Context, string) error {
	r.client.State = entity.ReaderClientStateIssuing
	return nil
}

type readerIDStub struct{}

func (readerIDStub) New() string { return "reader-row-id" }

func TestUserReadAccessServiceFailsClosedWhenRescopeFails(t *testing.T) {
	clientID := "gbrain-reader"
	repository := &readerClientRepositoryStub{client: entity.ReaderClient{
		ID: "reader-row", UserID: "user-1", GBrainClientID: &clientID,
		ClientSecretCiphertext: []byte("not-read-after-failure"),
		FederatedRead:          []entity.SourceID{"brain-a", "brain-b"}, State: entity.ReaderClientStateActive,
	}}
	admin := &readerAdminStub{rescopeErr: errors.New("admin unavailable")}
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("r", 32)))
	service, err := NewUserReadAccessService("http://gbrain.test", admin, repository, writerClockStub{time.Now()}, readerIDStub{}, key)
	if err != nil {
		t.Fatal(err)
	}
	token, err := service.AccessToken(context.Background(), "user-1", []entity.SourceID{"brain-a"})
	if err == nil || token != "" {
		t.Fatalf("token/error = %q %v; want fail closed", token, err)
	}
	if repository.client.State != entity.ReaderClientStateOrphan || repository.orphanCalls != 1 {
		t.Fatalf("reader state/calls = %s %d", repository.client.State, repository.orphanCalls)
	}
	if len(admin.rescopes) != 1 || len(admin.rescopes[0]) != 1 || admin.rescopes[0][0] != "brain-a" {
		t.Fatalf("rescopes = %v", admin.rescopes)
	}
}

func TestUserReadAccessServiceRefreshesTokenAfterSuccessfulRescope(t *testing.T) {
	clientID := "gbrain-reader"
	repository := &readerClientRepositoryStub{client: entity.ReaderClient{
		ID: "reader-row", UserID: "user-1", GBrainClientID: &clientID,
		ClientSecretCiphertext: []byte("cached-client-skips-decryption"),
		FederatedRead:          []entity.SourceID{"brain-a", "brain-b"}, State: entity.ReaderClientStateActive,
	}}
	tokenCalls := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			_ = json.NewEncoder(w).Encode(map[string]string{"token_endpoint": server.URL + "/token"})
		case "/token":
			tokenCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh-token", "expires_in": 3600, "token_type": "Bearer"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("r", 32)))
	service, err := NewUserReadAccessService(server.URL, &readerAdminStub{}, repository, writerClockStub{time.Now()}, readerIDStub{}, key)
	if err != nil {
		t.Fatal(err)
	}
	cached, err := newClient(server.URL, clientID, "secret", "read")
	if err != nil {
		t.Fatal(err)
	}
	cached.accessToken = "stale-wide-token"
	cached.tokenExpiry = time.Now().Add(time.Hour)
	service.clients["user-1"] = cached

	token, err := service.AccessToken(context.Background(), "user-1", []entity.SourceID{"brain-a"})
	if err != nil || token != "fresh-token" || tokenCalls != 1 {
		t.Fatalf("token/calls/error = %q %d %v", token, tokenCalls, err)
	}
}

func TestReadConnectionManagementDoesNotExposeCredentials(t *testing.T) {
	repository := &readerClientRepositoryStub{client: entity.ReaderClient{
		UserID: "user-1", State: entity.ReaderClientStateOrphan, StateReason: "reissue required",
		ClientSecretCiphertext: []byte("must stay inside adapter"),
	}}
	service := &UserReadAccessService{repository: repository}
	status, err := service.Status(context.Background(), "user-1")
	if err != nil || status == nil || status.State != entity.ReaderClientStateOrphan || status.StateReason != "reissue required" {
		t.Fatalf("status = %+v %v", status, err)
	}
	users, err := service.ConnectedUsers(context.Background())
	if err != nil || len(users) != 1 || users[0] != "user-1" {
		t.Fatalf("users = %v %v", users, err)
	}
	repository.findErr = output_port.ErrNotFound
	if status, err := service.Status(context.Background(), "user-1"); err != nil || status != nil {
		t.Fatalf("missing connection = %+v %v", status, err)
	}
	repository.findErr = errors.New("database unavailable")
	if _, err := service.Status(context.Background(), "user-1"); err == nil {
		t.Fatal("database failure must not look like an unconfigured connection")
	}
}
