package input_port

import (
	"context"
	"errors"

	"brainhub/domain/entity"
)

var (
	ErrPageNotFound = errors.New("page not found")
)

type PageUseCase interface {
	List(context.Context, entity.SourceID, string) ([]entity.Page, error)
	Get(context.Context, entity.SourceID, string, string) (entity.PageDetail, error)
}
