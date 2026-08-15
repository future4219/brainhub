package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type PageRepository interface {
	List(ctx context.Context, sourceID entity.SourceID) ([]entity.Page, error)
}
