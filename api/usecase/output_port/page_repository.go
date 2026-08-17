package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type PageRepository interface {
	List(ctx context.Context, sourceID entity.SourceID) ([]entity.Page, error)
	Get(ctx context.Context, sourceID entity.SourceID, slug string) (entity.PageDetail, error)
}

type PageEditorRepository interface {
	GetEditable(ctx context.Context, brainID string, sourceID entity.SourceID, slug string) (entity.PageDetail, error)
	ListTypes(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.PageType, error)
	Put(ctx context.Context, brainID string, sourceID entity.SourceID, page entity.PageWrite) error
}
