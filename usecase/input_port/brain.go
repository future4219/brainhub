package input_port

import (
	"context"
	"errors"

	"brainhub/domain/entity"
)

var ErrSourceNotFound = errors.New("source not found")

type BrainUseCase interface {
	List(context.Context) ([]entity.Source, error)
}
