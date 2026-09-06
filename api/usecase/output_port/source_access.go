package output_port

import (
	"context"

	"brainhub/domain/entity"
)

// SourceAccess manages credentials bound to one source. Callers check membership
// and consent before requesting a token; this interface does not authorize users.
type SourceAccess interface {
	Provision(ctx context.Context, brainID string, sourceID entity.SourceID) error
	Reissue(ctx context.Context, brainID string, sourceID entity.SourceID) error
	AccessToken(ctx context.Context, brainID string, sourceID entity.SourceID) (string, error)
}
