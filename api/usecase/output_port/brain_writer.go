package output_port

import (
	"context"
	"time"

	"brainhub/domain/entity"
)

type BrainWriterClientRepository interface {
	CreateBrainWriterClient(context.Context, entity.BrainWriterClient) error
	FindBrainWriterClient(context.Context, string) (entity.BrainWriterClient, error)
	ListIssuingBrainWriterClients(context.Context) ([]entity.BrainWriterClient, error)
	ActivateBrainWriterClient(context.Context, string, string, []byte, time.Time) (entity.BrainWriterClient, error)
	MarkBrainWriterClientOrphan(context.Context, string, *string, string) (entity.BrainWriterClient, error)
	ResetBrainWriterClient(context.Context, string) error
}

type BrainWriter interface {
	Provision(context.Context, string, entity.SourceID) error
	Reissue(context.Context, string, entity.SourceID) error
}

// MCPWriter only issues tokens for an already authorized, source-bound write.
type MCPWriter interface {
	AccessToken(context.Context, string, entity.SourceID) (string, error)
}
