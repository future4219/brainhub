package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type PageReader interface {
	List(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.Page, error)
	Get(ctx context.Context, brainID string, sourceID entity.SourceID, slug string) (entity.PageDetail, error)
}
