package interactor

import (
	"context"
	"errors"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

type brainUseCase struct {
	repository output_port.SourceRepository
}

func NewBrainUseCase(repository output_port.SourceRepository) (input_port.BrainUseCase, error) {
	if repository == nil {
		return nil, errors.New("source repository is required")
	}
	return &brainUseCase{repository: repository}, nil
}

func (u *brainUseCase) List(ctx context.Context) ([]entity.Source, error) {
	return u.repository.ListSources(ctx)
}
