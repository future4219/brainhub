package input_port

import (
	"context"
	"errors"

	"brainhub/domain/entity"
)

var (
	ErrInvalidInput           = errors.New("invalid input")
	ErrEmailAlreadyRegistered = errors.New("email already registered")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrRateLimited            = errors.New("too many login attempts")
)

type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

type LoginInput struct {
	Email      string
	Password   string
	RemoteAddr string
}

type AuthUseCase interface {
	Register(context.Context, RegisterInput) (entity.User, entity.Session, string, error)
	Login(context.Context, LoginInput) (entity.User, entity.Session, string, error)
	Authenticate(context.Context, string) (entity.User, entity.Session, error)
	Logout(context.Context, string) error
}
