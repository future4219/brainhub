package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type PageRepository interface {
	List(ctx context.Context, sourceID entity.SourceID) ([]entity.Page, error)
	Get(ctx context.Context, sourceID entity.SourceID, slug string) (entity.PageDetail, error)
}
