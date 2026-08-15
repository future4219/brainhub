package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type SourceCatalog interface {
	Exists(context.Context, entity.SourceID) (bool, error)
}
