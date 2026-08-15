package input_port

import (
	"context"

	"brainhub/domain/entity"
)

type PageUseCase interface {
	List(context.Context, entity.SourceID) ([]entity.Page, error)
}
