package interactor_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/interactor"
	"brainhub/usecase/output_port"
)

type accessStore struct {
	brains      map[string]entity.Brain
	users       map[string]entity.User
	invitations map[string]entity.Invitation
	tokens      map[string]string
	memberships map[string]entity.Membership
	clients     map[string]entity.IssuedClient
}

func newAccessStore() *accessStore {
	return &accessStore{
		brains: map[string]entity.Brain{}, users: map[string]entity.User{}, invitations: map[string]entity.Invitation{},
		tokens: map[string]string{}, memberships: map[string]entity.Membership{}, clients: map[string]entity.IssuedClient{},
	}
}

func (s *accessStore) FindBrainByID(_ context.Context, id string) (entity.Brain, error) {
	for _, brain := range s.brains {
		if brain.ID == id && brain.ArchivedAt == nil {
			return brain, nil
		}
	}
	return entity.Brain{}, output_port.ErrNotFound
}

func (s *accessStore) FindBrainBySourceID(_ context.Context, sourceID entity.SourceID) (entity.Brain, error) {
	brain, ok := s.brains[sourceID.String()]
	if !ok || brain.ArchivedAt != nil {
		return entity.Brain{}, output_port.ErrNotFound
	}
	return brain, nil
}

func (s *accessStore) ArchiveBrain(_ context.Context, id string, now time.Time) error {
	for sourceID, brain := range s.brains {
		if brain.ID == id && brain.ArchivedAt == nil {
			brain.State, brain.StateReason, brain.UpdatedAt, brain.ArchivedAt = entconst.BrainStateArchived, "", now, &now
			s.brains[sourceID] = brain
			return nil
		}
	}
	return output_port.ErrConflict
}

func (s *accessStore) CreateUser(_ context.Context, user entity.User) error {
	s.users[user.ID] = user
	return nil
}

func (s *accessStore) FindUserByID(_ context.Context, id string) (entity.User, error) {
	user, ok := s.users[id]
	if !ok {
		return entity.User{}, output_port.ErrNotFound
	}
	return user, nil
}

func (s *accessStore) CreateInvitation(_ context.Context, invitation entity.Invitation) (entity.Invitation, string, error) {
	raw := "raw-" + invitation.ID
	invitation.TokenHash = "hash-only"
	s.invitations[invitation.ID] = invitation
	s.tokens[raw] = invitation.ID
	return invitation, raw, nil
}

func (s *accessStore) FindInvitationByID(_ context.Context, id string) (entity.Invitation, error) {
	invitation, ok := s.invitations[id]
	if !ok {
		return entity.Invitation{}, output_port.ErrNotFound
	}
	return invitation, nil
}

func (s *accessStore) FindInvitationByToken(_ context.Context, raw string) (entity.Invitation, error) {
	id, ok := s.tokens[raw]
	if !ok {
		return entity.Invitation{}, output_port.ErrNotFound
	}
	return s.FindInvitationByID(context.Background(), id)
}

func (s *accessStore) ListInvitationsByBrain(_ context.Context, brainID string) ([]entity.Invitation, error) {
	var invitations []entity.Invitation
	for _, invitation := range s.invitations {
		if invitation.BrainID == brainID {
			invitations = append(invitations, invitation)
		}
	}
	return invitations, nil
}

func (s *accessStore) AcceptInvitation(_ context.Context, id, userID string, now time.Time) error {
	invitation := s.invitations[id]
	if invitation.State != entity.InvitationStatePending || !now.Before(invitation.ExpiresAt) {
		return output_port.ErrConflict
	}
	invitation.State, invitation.AcceptedAt, invitation.AcceptedBy = entity.InvitationStateAccepted, &now, &userID
	s.invitations[id] = invitation
	return nil
}

func (s *accessStore) RevokeInvitation(_ context.Context, id string) error {
	invitation := s.invitations[id]
	invitation.State = entity.InvitationStateRevoked
	s.invitations[id] = invitation
	return nil
}

func (s *accessStore) RevokePendingInvitationsByBrain(_ context.Context, brainID string) error {
	for id, invitation := range s.invitations {
		if invitation.BrainID == brainID && invitation.State == entity.InvitationStatePending {
			invitation.State = entity.InvitationStateRevoked
			s.invitations[id] = invitation
		}
	}
	return nil
}

func membershipKey(brainID, userID string) string { return brainID + "\x00" + userID }

func (s *accessStore) CreateMembership(_ context.Context, membership entity.Membership) error {
	key := membershipKey(membership.BrainID, membership.UserID)
	if _, ok := s.memberships[key]; ok {
		return output_port.ErrConflict
	}
	s.memberships[key] = membership
	return nil
}

func (s *accessStore) FindMembership(_ context.Context, brainID, userID string) (entity.Membership, error) {
	membership, ok := s.memberships[membershipKey(brainID, userID)]
	if !ok {
		return entity.Membership{}, output_port.ErrNotFound
	}
	return membership, nil
}

func (s *accessStore) FindActiveMembership(ctx context.Context, brainID, userID string) (entity.Membership, error) {
	membership, err := s.FindMembership(ctx, brainID, userID)
	if err != nil || membership.RevokedAt != nil {
		return entity.Membership{}, output_port.ErrNotFound
	}
	return membership, nil
}

func (s *accessStore) RevokeMembership(_ context.Context, id string, now time.Time) error {
	for key, membership := range s.memberships {
		if membership.ID == id && membership.RevokedAt == nil {
			membership.RevokedAt, membership.UpdatedAt = &now, now
			s.memberships[key] = membership
			return nil
		}
	}
	return output_port.ErrConflict
}

func (s *accessStore) CreateIssuedClient(_ context.Context, client entity.IssuedClient) error {
	s.clients[client.ID] = client
	return nil
}

func (s *accessStore) FindIssuedClientByID(_ context.Context, id string) (entity.IssuedClient, error) {
	client, ok := s.clients[id]
	if !ok {
		return entity.IssuedClient{}, output_port.ErrNotFound
	}
	return client, nil
}

func (s *accessStore) ListIssuedClientsByUserBrain(_ context.Context, userID, brainID string) ([]entity.IssuedClient, error) {
	var clients []entity.IssuedClient
	for _, client := range s.clients {
		if client.UserID == userID && client.BrainID == brainID {
			clients = append(clients, client)
		}
	}
	return clients, nil
}

func (s *accessStore) ListRevocableIssuedClientsByUserBrain(ctx context.Context, userID, brainID string) ([]entity.IssuedClient, error) {
	clients, _ := s.ListIssuedClientsByUserBrain(ctx, userID, brainID)
	var revocable []entity.IssuedClient
	for _, client := range clients {
		if client.GBrainClientID != nil && (client.State == entity.ClientStateActive || client.State == entity.ClientStateOrphan) {
			revocable = append(revocable, client)
		}
	}
	return revocable, nil
}

func (s *accessStore) ListRevocableIssuedClientsByBrain(_ context.Context, brainID string) ([]entity.IssuedClient, error) {
	var clients []entity.IssuedClient
	for _, client := range s.clients {
		if client.BrainID == brainID && client.GBrainClientID != nil && (client.State == entity.ClientStateActive || client.State == entity.ClientStateOrphan) {
			clients = append(clients, client)
		}
	}
	return clients, nil
}

func (s *accessStore) ActivateIssuedClient(_ context.Context, id, gbrainClientID string, now time.Time) (entity.IssuedClient, error) {
	client := s.clients[id]
	client.GBrainClientID, client.State, client.LastVerifiedAt = &gbrainClientID, entity.ClientStateActive, &now
	s.clients[id] = client
	return client, nil
}

func (s *accessStore) MarkIssuedClientOrphan(_ context.Context, id string, gbrainClientID *string, reason string) (entity.IssuedClient, error) {
	client := s.clients[id]
	if gbrainClientID != nil {
		client.GBrainClientID = gbrainClientID
	}
	client.State, client.StateReason = entity.ClientStateOrphan, reason
	s.clients[id] = client
	return client, nil
}

func (s *accessStore) MarkIssuedClientRevoked(_ context.Context, id string, now time.Time) (entity.IssuedClient, error) {
	client := s.clients[id]
	client.State, client.StateReason, client.RevokedAt = entity.ClientStateRevoked, "", &now
	s.clients[id] = client
	return client, nil
}

func (s *accessStore) WithinAccessTransaction(ctx context.Context, fn func(output_port.AccessRepositories) error) error {
	return fn(s)
}

type accessAdmin struct {
	registerInput output_port.RegisterGBrainClientInput
	registerErr   error
	revokeErr     error
	revoked       []string
}

func (a *accessAdmin) RegisterClient(_ context.Context, input output_port.RegisterGBrainClientInput) (output_port.RegisteredGBrainClient, error) {
	a.registerInput = input
	if a.registerErr != nil {
		return output_port.RegisteredGBrainClient{}, a.registerErr
	}
	return output_port.RegisteredGBrainClient{ID: "gbrain-client-id"}, nil
}

func (a *accessAdmin) RevokeClient(_ context.Context, clientID string) error {
	a.revoked = append(a.revoked, clientID)
	return a.revokeErr
}

type accessClock struct{ now time.Time }

func (c accessClock) Now() time.Time { return c.now }

type accessIDs struct{ next int }

func (g *accessIDs) New() string {
	g.next++
	return fmt.Sprintf("id-%d", g.next)
}

func accessFixture(t *testing.T) (*accessStore, *accessAdmin, input_port.AccessUseCase, accessClock) {
	t.Helper()
	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	store, admin := newAccessStore(), &accessAdmin{}
	store.brains["brainhub"] = entity.Brain{ID: "brain-id", SourceID: "brainhub", Name: "Brainhub", State: entconst.BrainStateReady}
	store.users["owner"] = entity.User{ID: "owner", Email: "owner@example.com", Name: "Owner"}
	store.users["reader"] = entity.User{ID: "reader", Email: "reader@example.com", Name: "Reader"}
	store.memberships[membershipKey("brain-id", "owner")] = entity.Membership{ID: "owner-membership", BrainID: "brain-id", UserID: "owner", Role: entity.RoleOwner, CreatedAt: now, UpdatedAt: now}
	clock := accessClock{now: now}
	useCase, err := interactor.NewAccessUseCase(store, store, store, store, store, store, admin, clock, &accessIDs{})
	if err != nil {
		t.Fatal(err)
	}
	return store, admin, useCase, clock
}

func TestInvitationAcceptanceRules(t *testing.T) {
	store, _, useCase, clock := accessFixture(t)
	invitation, rawToken, err := useCase.CreateInvitation(context.Background(), "brainhub", "owner", input_port.CreateInvitationInput{
		Email: "READER@example.com", Role: entity.RoleReader, ExpiresAt: clock.now.Add(time.Hour),
	})
	if err != nil || rawToken == "" || invitation.TokenHash == rawToken || invitation.Email == nil || *invitation.Email != "reader@example.com" {
		t.Fatalf("CreateInvitation = %+v %q %v", invitation, rawToken, err)
	}
	preview, err := useCase.PreviewInvitation(context.Background(), rawToken)
	if err != nil || preview.BrainName != "Brainhub" || preview.InvitedByName != "Owner" {
		t.Fatalf("PreviewInvitation = %+v %v", preview, err)
	}
	wrongUser := store.users["reader"]
	wrongUser.Email = "other@example.com"
	if _, err := useCase.AcceptInvitation(context.Background(), rawToken, wrongUser); !errors.Is(err, input_port.ErrInvitationEmail) {
		t.Fatalf("wrong email error = %v", err)
	}
	accepted, err := useCase.AcceptInvitation(context.Background(), rawToken, store.users["reader"])
	if err != nil || accepted.Membership.Role != entity.RoleReader || accepted.SourceID != "brainhub" {
		t.Fatalf("AcceptInvitation = %+v %v", accepted, err)
	}
	if _, err := useCase.AcceptInvitation(context.Background(), rawToken, store.users["reader"]); !errors.Is(err, input_port.ErrInvitationAccepted) {
		t.Fatalf("second accept error = %v", err)
	}
	expired := entity.Invitation{ID: "expired", BrainID: "brain-id", Role: entity.RoleReader, InvitedBy: "owner", State: entity.InvitationStatePending, ExpiresAt: clock.now, CreatedAt: clock.now.Add(-time.Hour)}
	store.invitations[expired.ID], store.tokens["expired-token"] = expired, expired.ID
	if _, err := useCase.AcceptInvitation(context.Background(), "expired-token", store.users["reader"]); !errors.Is(err, input_port.ErrInvitationNotFound) {
		t.Fatalf("expired accept error = %v", err)
	}
}

func TestClientIssueAndMembershipRevocation(t *testing.T) {
	store, admin, useCase, clock := accessFixture(t)
	store.memberships[membershipKey("brain-id", "reader")] = entity.Membership{ID: "reader-membership", BrainID: "brain-id", UserID: "reader", Role: entity.RoleReader, CreatedAt: clock.now, UpdatedAt: clock.now}
	client, err := useCase.IssueClient(context.Background(), "brainhub", "reader", "claude-web")
	if err != nil || client.State != entity.ClientStateActive || client.GBrainClientID == nil {
		t.Fatalf("IssueClient = %+v %v", client, err)
	}
	if admin.registerInput.Source != nil || !reflect.DeepEqual(admin.registerInput.Scopes, []string{"read"}) || !reflect.DeepEqual(admin.registerInput.FederatedRead, []string{"brainhub"}) {
		t.Fatalf("reader register input = %+v", admin.registerInput)
	}
	if err := useCase.RevokeMembership(context.Background(), "brainhub", "owner", "owner"); !errors.Is(err, input_port.ErrCannotRevokeSelf) {
		t.Fatalf("owner self-revoke error = %v", err)
	}
	if err := useCase.RevokeMembership(context.Background(), "brainhub", "owner", "reader"); err != nil {
		t.Fatal(err)
	}
	stored := store.clients[client.ID]
	if stored.State != entity.ClientStateRevoked || !reflect.DeepEqual(admin.revoked, []string{"gbrain-client-id"}) {
		t.Fatalf("revoked client/admin = %+v %v", stored, admin.revoked)
	}
}

func TestClientFailuresRemainOrphaned(t *testing.T) {
	store, admin, useCase, clock := accessFixture(t)
	store.memberships[membershipKey("brain-id", "reader")] = entity.Membership{ID: "reader-membership", BrainID: "brain-id", UserID: "reader", Role: entity.RoleReader, CreatedAt: clock.now, UpdatedAt: clock.now}
	admin.registerErr = errors.New("gbrain unavailable")
	client, err := useCase.IssueClient(context.Background(), "brainhub", "reader", "claude-web")
	if !errors.Is(err, input_port.ErrGBrainAdmin) || client.State != entity.ClientStateOrphan || client.StateReason != "gbrain unavailable" || client.GBrainClientID != nil {
		t.Fatalf("failed issue = %+v %v", client, err)
	}
	admin.registerErr = nil
	active, err := useCase.IssueClient(context.Background(), "brainhub", "reader", "claude-web")
	if err != nil {
		t.Fatal(err)
	}
	admin.revokeErr = errors.New("revoke unavailable")
	if err := useCase.RevokeMembership(context.Background(), "brainhub", "owner", "reader"); !errors.Is(err, input_port.ErrGBrainAdmin) {
		t.Fatalf("membership revoke error = %v", err)
	}
	if stored := store.clients[active.ID]; stored.State != entity.ClientStateOrphan || stored.StateReason != "revoke unavailable" {
		t.Fatalf("failed revoke state = %+v", stored)
	}
}

func TestOnlyOwnerCanArchiveBrain(t *testing.T) {
	store, admin, useCase, clock := accessFixture(t)
	store.memberships[membershipKey("brain-id", "reader")] = entity.Membership{ID: "reader-membership", BrainID: "brain-id", UserID: "reader", Role: entity.RoleReader}
	clientID := "reader-client"
	store.clients[clientID] = entity.IssuedClient{ID: clientID, UserID: "reader", BrainID: "brain-id", GBrainClientID: stringPointer("gbrain-reader"), State: entity.ClientStateActive}
	store.invitations["pending"] = entity.Invitation{ID: "pending", BrainID: "brain-id", State: entity.InvitationStatePending}

	if err := useCase.ArchiveBrain(context.Background(), "brainhub", "reader"); !errors.Is(err, input_port.ErrForbidden) {
		t.Fatalf("reader archive error = %v", err)
	}
	if err := useCase.ArchiveBrain(context.Background(), "brainhub", "owner"); err != nil {
		t.Fatal(err)
	}
	brain := store.brains["brainhub"]
	if brain.State != entconst.BrainStateArchived || brain.ArchivedAt == nil || !brain.ArchivedAt.Equal(clock.now) {
		t.Fatalf("archived brain = %+v", brain)
	}
	if store.clients[clientID].State != entity.ClientStateRevoked || store.invitations["pending"].State != entity.InvitationStateRevoked || !reflect.DeepEqual(admin.revoked, []string{"gbrain-reader"}) {
		t.Fatalf("client/invitation/admin = %+v %+v %v", store.clients[clientID], store.invitations["pending"], admin.revoked)
	}
}

func TestArchiveWaitsForClientRevocation(t *testing.T) {
	store, admin, useCase, _ := accessFixture(t)
	store.clients["active"] = entity.IssuedClient{ID: "active", BrainID: "brain-id", GBrainClientID: stringPointer("gbrain-active"), State: entity.ClientStateActive}
	admin.revokeErr = errors.New("revoke unavailable")

	if err := useCase.ArchiveBrain(context.Background(), "brainhub", "owner"); !errors.Is(err, input_port.ErrGBrainAdmin) {
		t.Fatalf("archive error = %v", err)
	}
	if brain := store.brains["brainhub"]; brain.ArchivedAt != nil {
		t.Fatalf("brain archived before clients were revoked: %+v", brain)
	}
}

func stringPointer(value string) *string { return &value }
