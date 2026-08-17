package output_port

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	New() string
}

type AuthRepositories interface {
	UserRepository
	AuthIdentityRepository
	SessionRepository
}

type TransactionManager interface {
	WithinTransaction(context.Context, func(AuthRepositories) error) error
}
