package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"brainhub/adapter/database/model"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Store struct {
	pool    *pgxpool.Pool
	queries querier
	clock   output_port.Clock
	ids     output_port.IDGenerator
	random  io.Reader
}

func New(pool *pgxpool.Pool, clock output_port.Clock, ids output_port.IDGenerator) *Store {
	return &Store{pool: pool, queries: pool, clock: clock, ids: ids, random: rand.Reader}
}

func (s *Store) CreateUser(ctx context.Context, user entity.User) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO users (id, email, name, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.Name, string(user.State), user.CreatedAt, user.UpdatedAt,
	)
	return mapError(err)
}

func (s *Store) FindUserByID(ctx context.Context, id string) (entity.User, error) {
	var user model.User
	err := s.queries.QueryRow(ctx, `
		SELECT id, email, name, state, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.Name, &user.State, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return entity.User{}, mapError(err)
	}
	return user.Entity(), nil
}

func (s *Store) CreateAuthIdentity(ctx context.Context, identity entity.AuthIdentity) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO auth_identities (id, user_id, provider, identifier, secret_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		identity.ID, identity.UserID, string(identity.Provider), identity.Identifier, identity.SecretHash, identity.CreatedAt,
	)
	return mapError(err)
}

func (s *Store) FindAuthIdentity(ctx context.Context, provider entconst.AuthProvider, identifier string) (entity.AuthIdentity, error) {
	var identity model.AuthIdentity
	err := s.queries.QueryRow(ctx, `
		SELECT id, user_id, provider, identifier, secret_hash, created_at
		FROM auth_identities WHERE provider = $1 AND identifier = $2`,
		string(provider), identifier,
	).Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.Identifier, &identity.SecretHash, &identity.CreatedAt)
	if err != nil {
		return entity.AuthIdentity{}, mapError(err)
	}
	return identity.Entity(), nil
}

func (s *Store) Create(ctx context.Context, userID string, expiresAt time.Time) (entity.Session, string, error) {
	rawToken, err := randomToken(s.random)
	if err != nil {
		return entity.Session{}, "", err
	}
	tokenHash := hashToken(rawToken)
	session := entity.Session{
		ID:        s.ids.New(),
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: s.clock.Now(),
	}
	_, err = s.queries.Exec(ctx, `
		INSERT INTO sessions (id, token_hash, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		session.ID, tokenHash, session.UserID, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return entity.Session{}, "", mapError(err)
	}
	return session, rawToken, nil
}

func (s *Store) Verify(ctx context.Context, rawToken string) (entity.Session, error) {
	var session model.Session
	err := s.queries.QueryRow(ctx, `
		SELECT id, user_id, expires_at, created_at, revoked_at
		FROM sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2`,
		hashToken(rawToken), s.clock.Now(),
	).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt, &session.RevokedAt)
	if err != nil {
		return entity.Session{}, mapError(err)
	}
	return session.Entity(), nil
}

func (s *Store) Revoke(ctx context.Context, sessionID string) error {
	_, err := s.queries.Exec(ctx, `
		UPDATE sessions SET revoked_at = $1
		WHERE id = $2 AND revoked_at IS NULL`, s.clock.Now(), sessionID)
	return mapError(err)
}

func (s *Store) RevokeAllByUser(ctx context.Context, userID string) error {
	_, err := s.queries.Exec(ctx, `
		UPDATE sessions SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL`, s.clock.Now(), userID)
	return mapError(err)
}

func (s *Store) WithinTransaction(ctx context.Context, fn func(output_port.AuthRepositories) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	transactionStore := &Store{
		pool:    s.pool,
		queries: tx,
		clock:   s.clock,
		ids:     s.ids,
		random:  s.random,
	}
	if err := fn(transactionStore); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func randomToken(reader io.Reader) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(reader, tokenBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return output_port.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return output_port.ErrConflict
	}
	return err
}
