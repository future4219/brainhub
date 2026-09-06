package output_port

import (
	"context"

	"brainhub/domain/entity"
)

// SourceAccess prepares or repairs a source connection. Credentials stay in the adapter.
type SourceAccess interface {
	Provision(ctx context.Context, brainID string, sourceID entity.SourceID) error
	Reissue(ctx context.Context, brainID string, sourceID entity.SourceID) error
}
