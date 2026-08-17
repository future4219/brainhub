package repository_test

import (
	"context"
	"testing"
	"time"

	"brainhub/adapter/clock"
	"brainhub/adapter/database"
	"brainhub/adapter/database/repository"
	"brainhub/adapter/ulid"
	"brainhub/config"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

func TestAccessRepositoryWithPostgres(t *testing.T) {
	databaseURL, err := config.DatabaseURL()
	if err != nil {
		t.Skip("BRAINHUB_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	ids, now := ulid.Generator{}, time.Now().UTC()
	store := repository.New(pool, clock.Clock{}, ids)
	owner := entity.User{ID: ids.New(), Email: ids.New() + "@example.com", Name: "Access Owner", State: entconst.UserStateActive, CreatedAt: now, UpdatedAt: now}
	reader := entity.User{ID: ids.New(), Email: ids.New() + "@example.com", Name: "Access Reader", State: entconst.UserStateActive, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateUser(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, reader); err != nil {
		t.Fatal(err)
	}
	brain := entity.Brain{ID: ids.New(), SourceID: entity.SourceID("access-" + ids.New()[20:]), Name: "Access Test", Visibility: entconst.VisibilityPrivate, OwnerID: owner.ID, State: entconst.BrainStateReady, CreatedAt: now, UpdatedAt: now}
	ownerMembership := entity.Membership{ID: ids.New(), BrainID: brain.ID, UserID: owner.ID, Role: entity.RoleOwner, CreatedAt: now, UpdatedAt: now}
	if err := store.WithinBrainTransaction(ctx, func(repositories output_port.BrainRepositories) error {
		if err := repositories.CreateBrain(ctx, brain); err != nil {
			return err
		}
		return repositories.CreateMembership(ctx, ownerMembership)
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM issued_clients WHERE brain_id = $1", brain.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM invitations WHERE brain_id = $1", brain.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM memberships WHERE brain_id = $1", brain.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM brains WHERE id = $1", brain.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1)", []string{owner.ID, reader.ID})
	})

	email := reader.Email
	invitation, rawToken, err := store.CreateInvitation(ctx, entity.Invitation{
		ID: ids.New(), BrainID: brain.ID, Email: &email, Role: entity.RoleReader,
		InvitedBy: owner.ID, State: entity.InvitationStatePending, ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	var storedHash string
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM invitations WHERE id = $1", invitation.ID).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if rawToken == "" || rawToken == storedHash || len(storedHash) != 64 {
		t.Fatalf("raw token was persisted: raw=%t equal=%t hash-length=%d", rawToken != "", rawToken == storedHash, len(storedHash))
	}
	readerMembership := entity.Membership{ID: ids.New(), BrainID: brain.ID, UserID: reader.ID, Role: entity.RoleReader, InvitedBy: &owner.ID, CreatedAt: now, UpdatedAt: now}
	if err := store.WithinAccessTransaction(ctx, func(repositories output_port.AccessRepositories) error {
		found, err := repositories.FindInvitationByToken(ctx, rawToken)
		if err != nil || found.ID != invitation.ID {
			return err
		}
		if err := repositories.CreateMembership(ctx, readerMembership); err != nil {
			return err
		}
		return repositories.AcceptInvitation(ctx, invitation.ID, reader.ID, now)
	}); err != nil {
		t.Fatal(err)
	}

	client := entity.IssuedClient{ID: ids.New(), UserID: reader.ID, BrainID: brain.ID, Label: "claude-web", ReadSourceIDs: []entity.SourceID{brain.SourceID}, Scopes: []string{"read"}, State: entity.ClientStateIssuing, IssuedAt: now}
	if err := store.CreateIssuedClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	orphan, err := store.MarkIssuedClientOrphan(ctx, client.ID, nil, "upstream unavailable")
	if err != nil || orphan.State != entity.ClientStateOrphan || orphan.GBrainClientID != nil || orphan.StateReason != "upstream unavailable" {
		t.Fatalf("orphan = %+v %v", orphan, err)
	}
}
