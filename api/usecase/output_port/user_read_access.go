package output_port

import (
	"context"

	"brainhub/domain/entity"
)

// UserReadAccess maintains a user-specific read grant across visible sources.
type UserReadAccess interface {
	Status(ctx context.Context, userID string) (*entity.ReadConnectionStatus, error)
	ConnectedUsers(ctx context.Context) ([]string, error)
	Prepare(ctx context.Context, userID string, sources []entity.SourceID) error
	Reissue(ctx context.Context, userID string, sources []entity.SourceID) error
}
