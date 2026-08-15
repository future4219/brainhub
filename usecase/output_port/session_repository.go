package output_port

import (
	"context"
	"time"

	"brainhub/domain/entity"
)

type SessionRepository interface {
	// The second return value is the raw token. Callers may only put it in a cookie.
	Create(ctx context.Context, userID string, expiresAt time.Time) (entity.Session, string, error)
	Verify(ctx context.Context, rawToken string) (entity.Session, error)
	Revoke(ctx context.Context, sessionID string) error
	RevokeAllByUser(ctx context.Context, userID string) error
}
