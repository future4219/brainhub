package output_port

import (
	"context"

	"brainhub/domain/entity"
)

type SourceRepository interface {
	ListSources(context.Context) ([]entity.Source, error)
}
