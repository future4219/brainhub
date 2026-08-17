package interactor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"brainhub/domain/constructor"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/domain/validation"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

const sessionTTL = 30 * 24 * time.Hour

type authUseCase struct {
	repositories output_port.AuthRepositories
	transactions output_port.TransactionManager
	hasher       output_port.PasswordHasher
	limiter      output_port.RateLimiter
	clock        output_port.Clock
	ids          output_port.IDGenerator
}

func NewAuthUseCase(
	repositories output_port.AuthRepositories,
	transactions output_port.TransactionManager,
	hasher output_port.PasswordHasher,
	limiter output_port.RateLimiter,
	clock output_port.Clock,
	ids output_port.IDGenerator,
) (input_port.AuthUseCase, error) {
	if repositories == nil || transactions == nil || hasher == nil || limiter == nil || clock == nil || ids == nil {
		return nil, errors.New("all auth dependencies are required")
	}
	return &authUseCase{
		repositories: repositories,
		transactions: transactions,
		hasher:       hasher,
		limiter:      limiter,
		clock:        clock,
		ids:          ids,
	}, nil
}

func (u *authUseCase) Register(ctx context.Context, input input_port.RegisterInput) (entity.User, entity.Session, string, error) {
	if err := validation.ValidatePassword(input.Password); err != nil {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
	}
	secretHash, err := u.hasher.Hash(input.Password)
	if err != nil {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("hash password: %w", err)
	}
	now := u.clock.Now()
	user, identity, err := constructor.NewPasswordUser(u.ids.New(), u.ids.New(), input.Email, input.Name, secretHash, now)
	if err != nil {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
	}

	var session entity.Session
	var rawToken string
	err = u.transactions.WithinTransaction(ctx, func(repositories output_port.AuthRepositories) error {
		if err := repositories.CreateUser(ctx, user); err != nil {
			return err
		}
		if err := repositories.CreateAuthIdentity(ctx, identity); err != nil {
			return err
		}
		session, rawToken, err = repositories.Create(ctx, user.ID, now.Add(sessionTTL))
		return err
	})
	if errors.Is(err, output_port.ErrConflict) {
		return entity.User{}, entity.Session{}, "", input_port.ErrEmailAlreadyRegistered
	}
	if err != nil {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("register user: %w", err)
	}
	return user, session, rawToken, nil
}

func (u *authUseCase) Login(ctx context.Context, input input_port.LoginInput) (entity.User, entity.Session, string, error) {
	rateKey := input.RemoteAddr + "\x00" + strings.ToLower(strings.TrimSpace(input.Email))
	now := u.clock.Now()
	if !u.limiter.Allow(rateKey, now) {
		return entity.User{}, entity.Session{}, "", input_port.ErrRateLimited
	}

	email, emailErr := validation.NormalizeEmail(input.Email)
	var identity entity.AuthIdentity
	var findErr error
	if emailErr == nil {
		identity, findErr = u.repositories.FindAuthIdentity(ctx, entconst.AuthProviderPassword, email)
	}
	secretHash := ""
	if identity.SecretHash != nil {
		secretHash = *identity.SecretHash
	}
	matched := u.hasher.Matches(secretHash, input.Password)
	if findErr != nil && !errors.Is(findErr, output_port.ErrNotFound) {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("find auth identity: %w", findErr)
	}
	if emailErr != nil || errors.Is(findErr, output_port.ErrNotFound) || !matched {
		return entity.User{}, entity.Session{}, "", input_port.ErrInvalidCredentials
	}

	user, err := u.repositories.FindUserByID(ctx, identity.UserID)
	if err != nil && !errors.Is(err, output_port.ErrNotFound) {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("find user: %w", err)
	}
	if errors.Is(err, output_port.ErrNotFound) || user.State != entconst.UserStateActive {
		return entity.User{}, entity.Session{}, "", input_port.ErrInvalidCredentials
	}
	session, rawToken, err := u.repositories.Create(ctx, user.ID, now.Add(sessionTTL))
	if err != nil {
		return entity.User{}, entity.Session{}, "", fmt.Errorf("create session: %w", err)
	}
	u.limiter.Reset(rateKey)
	return user, session, rawToken, nil
}

func (u *authUseCase) Authenticate(ctx context.Context, rawToken string) (entity.User, entity.Session, error) {
	if rawToken == "" {
		return entity.User{}, entity.Session{}, input_port.ErrUnauthorized
	}
	session, err := u.repositories.Verify(ctx, rawToken)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.User{}, entity.Session{}, input_port.ErrUnauthorized
	}
	if err != nil {
		return entity.User{}, entity.Session{}, fmt.Errorf("verify session: %w", err)
	}
	user, err := u.repositories.FindUserByID(ctx, session.UserID)
	if err != nil && !errors.Is(err, output_port.ErrNotFound) {
		return entity.User{}, entity.Session{}, fmt.Errorf("find session user: %w", err)
	}
	if errors.Is(err, output_port.ErrNotFound) || user.State != entconst.UserStateActive {
		return entity.User{}, entity.Session{}, input_port.ErrUnauthorized
	}
	return user, session, nil
}

func (u *authUseCase) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return input_port.ErrUnauthorized
	}
	if err := u.repositories.Revoke(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
