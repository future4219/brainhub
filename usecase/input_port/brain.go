package input_port

import (
	"context"
	"errors"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
)

var (
	ErrBrainNotFound      = errors.New("brain not found")
	ErrBrainAlreadyExists = errors.New("brain already exists")
	ErrProvisioningFailed = errors.New("brain provisioning failed")
	ErrSourceNotFound     = errors.New("GBrain source not found")
	ErrSourceLookupFailed = errors.New("GBrain source lookup failed")
)

type CreateBrainInput struct {
	SourceID    string
	Name        string
	Description string
	Visibility  entconst.Visibility
}

type AdoptBrainInput struct {
	Name        string
	Description string
	Visibility  entconst.Visibility
}

type BrainUseCase interface {
	Create(context.Context, string, CreateBrainInput) (entity.Brain, error)
	Adopt(context.Context, string, entity.SourceID, AdoptBrainInput) (entity.Brain, error)
	List(context.Context, string) ([]entity.Brain, error)
	Get(context.Context, entity.SourceID, string) (entity.Brain, error)
}
