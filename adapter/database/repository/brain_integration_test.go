package repository_test

import (
	"context"
	"errors"
	"strings"
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

func TestBrainRepositoryWithPostgres(t *testing.T) {
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

	ids := ulid.Generator{}
	now := time.Now().UTC()
	store := repository.New(pool, clock.Clock{}, ids)
	user := entity.User{
		ID: ids.New(), Email: ids.New() + "@example.com", Name: "Brain Repository Test",
		State: entconst.UserStateActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	brain := entity.Brain{
		ID: ids.New(), SourceID: entity.SourceID("repo-test-" + strings.ToLower(ids.New()[20:])), Name: "Repository Test",
		Visibility: entconst.VisibilityPrivate, OwnerID: user.ID, State: entconst.BrainStateProvisioning,
		CreatedAt: now, UpdatedAt: now,
	}
	membership := entity.Membership{
		ID: ids.New(), BrainID: brain.ID, UserID: user.ID, Role: entity.RoleOwner,
		CreatedAt: now, UpdatedAt: now,
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM memberships WHERE brain_id = $1", brain.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM brains WHERE id = $1", brain.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})

	if err := store.WithinBrainTransaction(ctx, func(repositories output_port.BrainRepositories) error {
		if err := repositories.CreateBrain(ctx, brain); err != nil {
			return err
		}
		return repositories.CreateMembership(ctx, membership)
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := store.FindBrainBySourceID(ctx, brain.SourceID)
	if err != nil || stored.State != entconst.BrainStateProvisioning {
		t.Fatalf("stored brain = %+v %v", stored, err)
	}
	memberships, err := store.ListActiveMembershipsByUser(ctx, user.ID)
	if err != nil || len(memberships) != 1 || memberships[0].Role != entity.RoleOwner {
		t.Fatalf("memberships = %+v %v", memberships, err)
	}
	ready, err := store.TransitionBrain(ctx, brain.ID, entconst.BrainStateReady, "", now.Add(time.Second))
	if err != nil || ready.State != entconst.BrainStateReady {
		t.Fatalf("ready = %+v %v", ready, err)
	}
	if _, err := store.TransitionBrain(ctx, brain.ID, entconst.BrainStateFailed, "late failure", now.Add(2*time.Second)); !errors.Is(err, output_port.ErrConflict) {
		t.Fatalf("second transition error = %v", err)
	}
}
