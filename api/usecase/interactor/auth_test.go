package interactor_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/interactor"
	"brainhub/usecase/output_port"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type sequenceIDs struct{ next int }

func (i *sequenceIDs) New() string {
	i.next++
	return fmt.Sprintf("id-%d", i.next)
}

type passwordHasherMock struct {
	comparedHashes []string
}

func (h *passwordHasherMock) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (h *passwordHasherMock) Matches(hash, password string) bool {
	h.comparedHashes = append(h.comparedHashes, hash)
	return hash == "hashed:"+password
}

type limiterMock struct{}

func (limiterMock) Allow(string, time.Time) bool { return true }
func (limiterMock) Reset(string)                 {}

type authRepositoriesMock struct {
	users      map[string]entity.User
	identities map[string]entity.AuthIdentity
	sessions   map[string]entity.Session
	nextToken  int
}

func newAuthRepositoriesMock() *authRepositoriesMock {
	return &authRepositoriesMock{
		users:      make(map[string]entity.User),
		identities: make(map[string]entity.AuthIdentity),
		sessions:   make(map[string]entity.Session),
	}
}

func (r *authRepositoriesMock) CreateUser(_ context.Context, user entity.User) error {
	for _, existing := range r.users {
		if existing.Email == user.Email {
			return output_port.ErrConflict
		}
	}
	r.users[user.ID] = user
	return nil
}

func (r *authRepositoriesMock) FindUserByID(_ context.Context, id string) (entity.User, error) {
	user, ok := r.users[id]
	if !ok {
		return entity.User{}, output_port.ErrNotFound
	}
	return user, nil
}

func (r *authRepositoriesMock) CreateAuthIdentity(_ context.Context, identity entity.AuthIdentity) error {
	r.identities[string(identity.Provider)+"\x00"+identity.Identifier] = identity
	return nil
}

func (r *authRepositoriesMock) FindAuthIdentity(_ context.Context, provider entconst.AuthProvider, identifier string) (entity.AuthIdentity, error) {
	identity, ok := r.identities[string(provider)+"\x00"+identifier]
	if !ok {
		return entity.AuthIdentity{}, output_port.ErrNotFound
	}
	return identity, nil
}

func (r *authRepositoriesMock) Create(_ context.Context, userID string, expiresAt time.Time) (entity.Session, string, error) {
	r.nextToken++
	raw := fmt.Sprintf("raw-%d", r.nextToken)
	session := entity.Session{ID: fmt.Sprintf("session-%d", r.nextToken), UserID: userID, ExpiresAt: expiresAt}
	r.sessions[raw] = session
	return session, raw, nil
}

func (r *authRepositoriesMock) Verify(_ context.Context, rawToken string) (entity.Session, error) {
	session, ok := r.sessions[rawToken]
	if !ok || session.RevokedAt != nil {
		return entity.Session{}, output_port.ErrNotFound
	}
	return session, nil
}

func (r *authRepositoriesMock) Revoke(_ context.Context, sessionID string) error {
	for raw, session := range r.sessions {
		if session.ID == sessionID {
			now := time.Now()
			session.RevokedAt = &now
			r.sessions[raw] = session
		}
	}
	return nil
}

func (r *authRepositoriesMock) RevokeAllByUser(_ context.Context, userID string) error {
	for raw, session := range r.sessions {
		if session.UserID == userID {
			now := time.Now()
			session.RevokedAt = &now
			r.sessions[raw] = session
		}
	}
	return nil
}

func (r *authRepositoriesMock) WithinTransaction(_ context.Context, fn func(output_port.AuthRepositories) error) error {
	return fn(r)
}

func TestAuthUseCase(t *testing.T) {
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	repositories := newAuthRepositoriesMock()
	hasher := &passwordHasherMock{}
	useCase, err := interactor.NewAuthUseCase(repositories, repositories, hasher, limiterMock{}, fixedClock{now}, &sequenceIDs{})
	if err != nil {
		t.Fatal(err)
	}

	user, session, rawToken, err := useCase.Register(context.Background(), input_port.RegisterInput{
		Email: " Alice@Example.COM ", Password: "correct-password", Name: " Alice ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "alice@example.com" || user.Name != "Alice" || user.State != entconst.UserStateActive {
		t.Fatalf("normalized user = %+v", user)
	}
	if rawToken == "" || session.ExpiresAt.Sub(now) != 30*24*time.Hour {
		t.Fatalf("session/raw token = %+v %q", session, rawToken)
	}

	loggedIn, _, loginToken, err := useCase.Login(context.Background(), input_port.LoginInput{
		Email: "ALICE@example.com", Password: "correct-password", RemoteAddr: "127.0.0.1",
	})
	if err != nil || loggedIn.ID != user.ID || loginToken == "" {
		t.Fatalf("login = %+v %q %v", loggedIn, loginToken, err)
	}

	_, _, _, wrongPasswordErr := useCase.Login(context.Background(), input_port.LoginInput{
		Email: "alice@example.com", Password: "wrong-password", RemoteAddr: "127.0.0.1",
	})
	_, _, _, missingEmailErr := useCase.Login(context.Background(), input_port.LoginInput{
		Email: "missing@example.com", Password: "wrong-password", RemoteAddr: "127.0.0.1",
	})
	if !errors.Is(wrongPasswordErr, input_port.ErrInvalidCredentials) || !errors.Is(missingEmailErr, input_port.ErrInvalidCredentials) {
		t.Fatalf("login errors differ: %v / %v", wrongPasswordErr, missingEmailErr)
	}
	if hasher.comparedHashes[len(hasher.comparedHashes)-1] != "" {
		t.Fatal("missing email did not use the dummy-hash path")
	}
}
