package gbrain

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

type writerAdminStub struct {
	registered int
	revoked    []string
}

func (a *writerAdminStub) RegisterClient(_ context.Context, input output_port.RegisterGBrainClientInput) (output_port.RegisteredGBrainClient, error) {
	a.registered++
	if input.Source == nil || len(input.Scopes) != 2 || input.Scopes[0] != "read" || input.Scopes[1] != "write" {
		return output_port.RegisteredGBrainClient{}, errors.New("unexpected registration")
	}
	return output_port.RegisteredGBrainClient{ID: "client-" + strconv.Itoa(a.registered), Secret: "secret"}, nil
}

func (a *writerAdminStub) RevokeClient(_ context.Context, clientID string) error {
	a.revoked = append(a.revoked, clientID)
	return nil
}

type writerRepositoryStub struct {
	client entity.BrainWriterClient
}

func (r *writerRepositoryStub) CreateBrainWriterClient(context.Context, entity.BrainWriterClient) error {
	return errors.New("not used")
}
func (r *writerRepositoryStub) FindBrainWriterClient(context.Context, string) (entity.BrainWriterClient, error) {
	return r.client, nil
}
func (r *writerRepositoryStub) ListIssuingBrainWriterClients(context.Context) ([]entity.BrainWriterClient, error) {
	if r.client.State == entity.WriterClientStateIssuing {
		return []entity.BrainWriterClient{r.client}, nil
	}
	return nil, nil
}
func (r *writerRepositoryStub) ActivateBrainWriterClient(_ context.Context, _ string, clientID string, ciphertext []byte, now time.Time) (entity.BrainWriterClient, error) {
	r.client.GBrainClientID = &clientID
	r.client.ClientSecretCiphertext = ciphertext
	r.client.State = entity.WriterClientStateActive
	r.client.IssuedAt = &now
	return r.client, nil
}
func (r *writerRepositoryStub) MarkBrainWriterClientOrphan(_ context.Context, _ string, clientID *string, reason string) (entity.BrainWriterClient, error) {
	r.client.GBrainClientID = clientID
	r.client.ClientSecretCiphertext = nil
	r.client.State = entity.WriterClientStateOrphan
	r.client.StateReason = reason
	return r.client, nil
}
func (r *writerRepositoryStub) ResetBrainWriterClient(context.Context, string) error {
	r.client.GBrainClientID = nil
	r.client.ClientSecretCiphertext = nil
	r.client.State = entity.WriterClientStateIssuing
	return nil
}

type writerClockStub struct{ now time.Time }

func (c writerClockStub) Now() time.Time { return c.now }

func TestWriterServiceKeepsOnePersistentClientAndReissuesExplicitly(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	repository := &writerRepositoryStub{client: entity.BrainWriterClient{
		BrainID: "brain-id", WriteSourceID: "brainhub", State: entity.WriterClientStateIssuing,
	}}
	admin := &writerAdminStub{}
	service, err := NewWriterService("http://gbrain.test", admin, repository, writerClockStub{time.Now()}, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Provision(context.Background(), "brain-id", "brainhub"); err != nil {
		t.Fatal(err)
	}
	firstID := *repository.client.GBrainClientID
	if admin.registered != 1 || repository.client.State != entity.WriterClientStateActive || len(repository.client.ClientSecretCiphertext) == 0 {
		t.Fatalf("registered=%d client=%#v", admin.registered, repository.client)
	}
	if err := service.Provision(context.Background(), "brain-id", "brainhub"); err != nil {
		t.Fatal(err)
	}
	if admin.registered != 1 {
		t.Fatalf("active writer was registered again: %d", admin.registered)
	}
	firstClient, err := service.writerClient(context.Background(), "brain-id", "brainhub")
	if err != nil {
		t.Fatal(err)
	}
	secondClient, err := service.writerClient(context.Background(), "brain-id", "brainhub")
	if err != nil || secondClient != firstClient {
		t.Fatalf("writer client was not reused: same=%t err=%v", secondClient == firstClient, err)
	}
	if err := service.Reissue(context.Background(), "brain-id", "brainhub"); err != nil {
		t.Fatal(err)
	}
	if admin.registered != 2 || len(admin.revoked) != 1 || admin.revoked[0] != firstID || *repository.client.GBrainClientID == firstID {
		t.Fatalf("registered=%d revoked=%v client=%#v", admin.registered, admin.revoked, repository.client)
	}
	reissuedClient, err := service.writerClient(context.Background(), "brain-id", "brainhub")
	if err != nil || reissuedClient == firstClient {
		t.Fatalf("reissued writer client was not refreshed: same=%t err=%v", reissuedClient == firstClient, err)
	}
}
