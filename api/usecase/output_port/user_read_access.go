package output_port

import (
	"context"

	"brainhub/domain/entity"
)

// UserReadAccess maintains a user-specific read grant across visible sources.
type UserReadAccess interface {
	AccessToken(ctx context.Context, userID string, sources []entity.SourceID) (string, error)
	Reissue(ctx context.Context, userID string, sources []entity.SourceID) error
}
