package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type PageEditorRepository interface {
	List(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.Page, error)
	GetEditable(ctx context.Context, brainID string, sourceID entity.SourceID, slug string) (entity.PageDetail, error)
	ListTypes(ctx context.Context, brainID string, sourceID entity.SourceID) ([]entity.PageType, error)
	Put(ctx context.Context, brainID string, sourceID entity.SourceID, page entity.PageWrite) error
}
