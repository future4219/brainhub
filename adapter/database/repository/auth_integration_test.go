package repository_test

import (
	"context"
	"errors"
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

func TestSessionRepositoryWithPostgres(t *testing.T) {
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
	store := repository.New(pool, clock.Clock{}, ids)
	now := time.Now().UTC()
	user := entity.User{
		ID: ids.New(), Email: ids.New() + "@example.com", Name: "Repository Test",
		State: entconst.UserStateActive, CreatedAt: now, UpdatedAt: now,
	}
	identityHash := "not-a-real-password-hash"
	identity := entity.AuthIdentity{
		ID: ids.New(), UserID: user.ID, Provider: entconst.AuthProviderPassword,
		Identifier: user.Email, SecretHash: &identityHash, CreatedAt: now,
	}
	var session entity.Session
	var rawToken string
	if err := store.WithinTransaction(ctx, func(repositories output_port.AuthRepositories) error {
		if err := repositories.CreateUser(ctx, user); err != nil {
			return err
		}
		if err := repositories.CreateAuthIdentity(ctx, identity); err != nil {
			return err
		}
		session, rawToken, err = repositories.Create(ctx, user.ID, now.Add(time.Hour))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, statement := range []string{
			"DELETE FROM sessions WHERE user_id = $1",
			"DELETE FROM auth_identities WHERE user_id = $1",
			"DELETE FROM users WHERE id = $1",
		} {
			if _, err := pool.Exec(ctx, statement, user.ID); err != nil {
				t.Errorf("clean integration test data: %v", err)
			}
		}
	})

	var storedHash string
	if err := pool.QueryRow(ctx, "SELECT token_hash FROM sessions WHERE id = $1", session.ID).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if rawToken == "" || storedHash == rawToken || len(storedHash) != 64 {
		t.Fatalf("raw/stored token representation is unsafe: raw=%t hash_length=%d equal=%t", rawToken != "", len(storedHash), storedHash == rawToken)
	}
	verified, err := store.Verify(ctx, rawToken)
	if err != nil || verified.ID != session.ID {
		t.Fatalf("verify = %+v %v", verified, err)
	}
	if err := store.Revoke(ctx, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Verify(ctx, rawToken); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("verify revoked session error = %v", err)
	}
}
