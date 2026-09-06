package gbrain

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"

	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

type UserReadAccessService struct {
	baseURL    string
	admin      output_port.GBrainReaderAdmin
	repository output_port.ReaderClientRepository
	cipher     *credentialCipher
	clock      output_port.Clock
	ids        output_port.IDGenerator

	// ponytail: one API replica currently serializes low-volume reader setup;
	// replace with a DB lease before running multiple replicas.
	mu      sync.Mutex
	clients map[string]*Client
}

func NewUserReadAccessService(baseURL string, admin output_port.GBrainReaderAdmin, repository output_port.ReaderClientRepository, clock output_port.Clock, ids output_port.IDGenerator, encodedKey string) (*UserReadAccessService, error) {
	if baseURL == "" || admin == nil || repository == nil || clock == nil || ids == nil {
		return nil, errors.New("all reader service dependencies are required")
	}
	cipher, err := newCredentialCipher(encodedKey)
	if err != nil {
		return nil, err
	}
	return &UserReadAccessService{
		baseURL: baseURL, admin: admin, repository: repository, cipher: cipher,
		clock: clock, ids: ids, clients: make(map[string]*Client),
	}, nil
}

func (s *UserReadAccessService) AccessToken(ctx context.Context, userID string, sources []entity.SourceID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sources = normalizedSources(sources)
	if userID == "" || len(sources) == 0 {
		return "", output_port.ErrNoVisibleSources
	}
	stored, err := s.ensure(ctx, userID, sources)
	if err != nil {
		return "", err
	}
	client, err := s.clientFor(ctx, stored)
	if err != nil {
		return "", err
	}
	return client.BearerToken(ctx)
}

func (s *UserReadAccessService) ensure(ctx context.Context, userID string, sources []entity.SourceID) (entity.ReaderClient, error) {
	stored, err := s.repository.FindReaderClientByUser(ctx, userID)
	if errors.Is(err, output_port.ErrNotFound) {
		stored = entity.ReaderClient{
			ID: s.ids.New(), UserID: userID, State: entity.ReaderClientStateIssuing,
		}
		if err := s.repository.CreateReaderClient(ctx, stored); err != nil {
			return entity.ReaderClient{}, fmt.Errorf("create reader client: %w", err)
		}
	} else if err != nil {
		return entity.ReaderClient{}, fmt.Errorf("find reader client: %w", err)
	}

	switch stored.State {
	case entity.ReaderClientStateIssuing:
		return s.provision(ctx, stored, sources)
	case entity.ReaderClientStateOrphan:
		return entity.ReaderClient{}, output_port.ErrReaderNeedsReissue
	case entity.ReaderClientStateActive:
		if slices.Equal(stored.FederatedRead, sources) {
			return stored, nil
		}
		if stored.GBrainClientID == nil {
			return entity.ReaderClient{}, s.markOrphan(ctx, userID, nil, "active reader has no GBrain client ID")
		}
		if err := s.admin.RescopeClient(ctx, *stored.GBrainClientID, sourceStrings(sources)); err != nil {
			markErr := s.markOrphan(ctx, userID, stored.GBrainClientID, "rescope reader client: "+err.Error())
			// SECURITY: never continue with the previously wider GBrain grant.
			// Memberships are brainhub's source of truth; a failed rescope must
			// fail closed before any upstream token or request is forwarded.
			return entity.ReaderClient{}, errors.Join(fmt.Errorf("rescope reader client: %w", err), markErr)
		}
		if cached := s.clients[userID]; cached != nil {
			// GBrain access tokens may carry a snapshot of the old grant. Force a
			// fresh token after every successful rescope, especially on shrink.
			cached.clearBearerToken()
		}
		updated, err := s.repository.UpdateReaderClientScope(ctx, userID, sources, s.clock.Now())
		if err != nil {
			// GBrain now has the new scope but brainhub cannot prove/persist it.
			// Fail closed and require reissue instead of forwarding ambiguously.
			markErr := s.markOrphan(ctx, userID, stored.GBrainClientID, "persist reader scope: "+err.Error())
			return entity.ReaderClient{}, errors.Join(fmt.Errorf("persist reader scope: %w", err), markErr)
		}
		return updated, nil
	default:
		return entity.ReaderClient{}, s.markOrphan(ctx, userID, stored.GBrainClientID, "reader client has an invalid state")
	}
}

func (s *UserReadAccessService) provision(ctx context.Context, stored entity.ReaderClient, sources []entity.SourceID) (entity.ReaderClient, error) {
	primary := sources[0].String()
	registered, err := s.admin.RegisterClient(ctx, output_port.RegisterGBrainClientInput{
		Name: "brainhub-reader-" + stored.UserID, Scopes: []string{"read"}, Source: &primary,
		FederatedRead: sourceStrings(sources), GrantTypes: []string{"client_credentials"},
		TokenEndpointAuthMethod: "client_secret_post",
	})
	if err != nil {
		markErr := s.markOrphan(ctx, stored.UserID, nil, "register reader client: "+err.Error())
		return entity.ReaderClient{}, errors.Join(fmt.Errorf("register reader client: %w", err), markErr)
	}
	if registered.Secret == "" {
		revokeErr := s.admin.RevokeClient(ctx, registered.ID)
		clientID := &registered.ID
		if revokeErr == nil {
			clientID = nil
		}
		markErr := s.markOrphan(ctx, stored.UserID, clientID, "GBrain did not return a reader client secret")
		return entity.ReaderClient{}, errors.Join(errors.New("GBrain did not return a reader client secret"), revokeErr, markErr)
	}
	ciphertext, err := s.cipher.encrypt("reader:"+stored.UserID, registered.ID, registered.Secret)
	if err == nil {
		var active entity.ReaderClient
		active, err = s.repository.ActivateReaderClient(ctx, stored.UserID, registered.ID, ciphertext, sources, s.clock.Now())
		if err == nil {
			return active, nil
		}
	}
	revokeErr := s.admin.RevokeClient(ctx, registered.ID)
	clientID := &registered.ID
	if revokeErr == nil {
		clientID = nil
	}
	markErr := s.markOrphan(ctx, stored.UserID, clientID, "persist reader credentials: "+err.Error())
	return entity.ReaderClient{}, errors.Join(fmt.Errorf("persist reader credentials: %w", err), revokeErr, markErr)
}

func (s *UserReadAccessService) Reissue(ctx context.Context, userID string, sources []entity.SourceID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sources = normalizedSources(sources)
	if len(sources) == 0 {
		return output_port.ErrNoVisibleSources
	}
	stored, err := s.repository.FindReaderClientByUser(ctx, userID)
	if err != nil {
		return err
	}
	if stored.State != entity.ReaderClientStateOrphan {
		return output_port.ErrConflict
	}
	delete(s.clients, userID)
	if stored.GBrainClientID != nil {
		if err := s.admin.RevokeClient(ctx, *stored.GBrainClientID); err != nil {
			_, markErr := s.repository.MarkReaderClientOrphan(ctx, userID, stored.GBrainClientID, "revoke reader client: "+err.Error())
			return errors.Join(err, markErr)
		}
	}
	if err := s.repository.ResetReaderClient(ctx, userID); err != nil {
		return err
	}
	reset, err := s.repository.FindReaderClientByUser(ctx, userID)
	if err != nil {
		return err
	}
	_, err = s.provision(ctx, reset, sources)
	return err
}

func (s *UserReadAccessService) clientFor(ctx context.Context, stored entity.ReaderClient) (*Client, error) {
	if stored.GBrainClientID == nil || len(stored.ClientSecretCiphertext) == 0 {
		return nil, output_port.ErrReaderNeedsReissue
	}
	if cached := s.clients[stored.UserID]; cached != nil {
		return cached, nil
	}
	secret, err := s.cipher.decrypt("reader:"+stored.UserID, *stored.GBrainClientID, stored.ClientSecretCiphertext)
	if err != nil {
		_, markErr := s.repository.MarkReaderClientOrphan(ctx, stored.UserID, stored.GBrainClientID, "decrypt reader credential: "+err.Error())
		return nil, errors.Join(output_port.ErrReaderNeedsReissue, err, markErr)
	}
	client, err := newClient(s.baseURL, *stored.GBrainClientID, secret, "read")
	if err != nil {
		return nil, err
	}
	s.clients[stored.UserID] = client
	return client, nil
}

func (s *UserReadAccessService) markOrphan(ctx context.Context, userID string, clientID *string, reason string) error {
	delete(s.clients, userID)
	_, err := s.repository.MarkReaderClientOrphan(ctx, userID, clientID, reason)
	return err
}

func normalizedSources(sources []entity.SourceID) []entity.SourceID {
	values := append([]entity.SourceID(nil), sources...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return slices.Compact(values)
}

func sourceStrings(sources []entity.SourceID) []string {
	values := make([]string, len(sources))
	for i, source := range sources {
		values[i] = source.String()
	}
	return values
}
