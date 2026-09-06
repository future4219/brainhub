package gbrain

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

type WriterService struct {
	admin       output_port.GBrainAdmin
	credentials output_port.BrainWriterClientRepository
	cipher      *credentialCipher
	clock       output_port.Clock
	baseURL     string

	// ponytail: one brainhub process currently handles low-volume writer setup;
	// replace this with a DB lease before running multiple API replicas.
	mu      sync.Mutex
	clients map[string]*Client
}

func NewWriterService(baseURL string, admin output_port.GBrainAdmin, credentials output_port.BrainWriterClientRepository, clock output_port.Clock, encodedKey string) (*WriterService, error) {
	if baseURL == "" || admin == nil || credentials == nil || clock == nil {
		return nil, errors.New("all writer service dependencies are required")
	}
	cipher, err := newCredentialCipher(encodedKey)
	if err != nil {
		return nil, err
	}
	return &WriterService{
		baseURL: baseURL, admin: admin, credentials: credentials, cipher: cipher, clock: clock,
		clients: make(map[string]*Client),
	}, nil
}

func (s *WriterService) Provision(ctx context.Context, brainID string, sourceID entity.SourceID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.provision(ctx, brainID, sourceID)
}

func (s *WriterService) provision(ctx context.Context, brainID string, sourceID entity.SourceID) error {
	stored, err := s.credentials.FindBrainWriterClient(ctx, brainID)
	if err != nil {
		return fmt.Errorf("find brain writer client: %w", err)
	}
	if stored.WriteSourceID != sourceID {
		return errors.New("brain writer source does not match brain source")
	}
	if stored.State == entity.WriterClientStateActive {
		return nil
	}
	if stored.State != entity.WriterClientStateIssuing {
		return errors.New("brain writer client requires manual reissue")
	}

	source := sourceID.String()
	registered, err := s.admin.RegisterClient(ctx, output_port.RegisterGBrainClientInput{
		Name: "brainhub-writer-" + source, Scopes: []string{"read", "write"}, Source: &source,
		FederatedRead: []string{source}, GrantTypes: []string{"client_credentials"},
		TokenEndpointAuthMethod: "client_secret_post",
	})
	if err != nil {
		_, markErr := s.credentials.MarkBrainWriterClientOrphan(ctx, brainID, nil, err.Error())
		return errors.Join(fmt.Errorf("register brain writer client: %w", err), markErr)
	}
	if registered.Secret == "" {
		revokeErr := s.admin.RevokeClient(ctx, registered.ID)
		orphanID := &registered.ID
		reason := "GBrain did not return a client secret"
		if revokeErr == nil {
			orphanID = nil
		} else {
			reason += "; revoke: " + revokeErr.Error()
		}
		_, markErr := s.credentials.MarkBrainWriterClientOrphan(ctx, brainID, orphanID, reason)
		return errors.Join(errors.New(reason), markErr)
	}
	ciphertext, err := s.cipher.encrypt(brainID, registered.ID, registered.Secret)
	if err == nil {
		_, err = s.credentials.ActivateBrainWriterClient(ctx, brainID, registered.ID, ciphertext, s.clock.Now())
	}
	if err == nil {
		return nil
	}
	revokeErr := s.admin.RevokeClient(ctx, registered.ID)
	reason := fmt.Sprintf("persist writer credentials: %v", err)
	orphanID := &registered.ID
	if revokeErr != nil {
		reason += "; revoke: " + revokeErr.Error()
	} else {
		orphanID = nil
	}
	_, markErr := s.credentials.MarkBrainWriterClientOrphan(ctx, brainID, orphanID, reason)
	return errors.Join(errors.New(reason), markErr)
}

func (s *WriterService) Reissue(ctx context.Context, brainID string, sourceID entity.SourceID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, err := s.credentials.FindBrainWriterClient(ctx, brainID)
	if err != nil {
		return fmt.Errorf("find brain writer client: %w", err)
	}
	if stored.WriteSourceID != sourceID {
		return errors.New("brain writer source does not match brain source")
	}
	delete(s.clients, brainID+":"+sourceID.String())
	if stored.GBrainClientID != nil {
		if err := s.admin.RevokeClient(ctx, *stored.GBrainClientID); err != nil {
			_, markErr := s.credentials.MarkBrainWriterClientOrphan(ctx, brainID, stored.GBrainClientID, "revoke writer client: "+err.Error())
			return errors.Join(fmt.Errorf("revoke writer client: %w", err), markErr)
		}
	}
	if err := s.credentials.ResetBrainWriterClient(ctx, brainID); err != nil {
		return fmt.Errorf("reset brain writer client: %w", err)
	}
	return s.provision(ctx, brainID, sourceID)
}

func (s *WriterService) Backfill(ctx context.Context) error {
	clients, err := s.credentials.ListIssuingBrainWriterClients(ctx)
	if err != nil {
		return fmt.Errorf("list writer clients for backfill: %w", err)
	}
	var failures []error
	for _, client := range clients {
		if err := s.Provision(ctx, client.BrainID, client.WriteSourceID); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", client.WriteSourceID, err))
		}
	}
	return errors.Join(failures...)
}

func (s *WriterService) credentialsFor(ctx context.Context, brainID string, sourceID entity.SourceID) (string, string, error) {
	stored, err := s.credentials.FindBrainWriterClient(ctx, brainID)
	if err != nil {
		return "", "", fmt.Errorf("find brain writer client: %w", err)
	}
	if stored.State != entity.WriterClientStateActive || stored.GBrainClientID == nil || len(stored.ClientSecretCiphertext) == 0 {
		return "", "", errors.New("brain writer client is not active; an owner must reissue it")
	}
	if stored.WriteSourceID != sourceID {
		return "", "", errors.New("brain writer source does not match brain source")
	}
	secret, err := s.cipher.decrypt(brainID, *stored.GBrainClientID, stored.ClientSecretCiphertext)
	if err != nil {
		return "", "", fmt.Errorf("%w; an owner must reissue the writer client", err)
	}
	return *stored.GBrainClientID, secret, nil
}

func (s *WriterService) AccessToken(ctx context.Context, brainID string, sourceID entity.SourceID) (string, error) {
	client, err := s.writerClient(ctx, brainID, sourceID)
	if err != nil {
		return "", err
	}
	return client.BearerToken(ctx)
}
