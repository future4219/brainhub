package input_port

import (
	"context"
	"errors"

	"brainhub/domain/entity"
)

var (
	ErrPageNotFound      = errors.New("page not found")
	ErrPageAlreadyExists = errors.New("page already exists")
)

type CreatePageInput struct {
	Slug          string
	Title         string
	Type          string
	Tags          []string
	SupersededBy  *string
	CompiledTruth string
	TimelineEntry string
}

type UpdatePageInput struct {
	Title         string
	Type          string
	Tags          []string
	SupersededBy  *string
	CompiledTruth string
	TimelineEntry string
}

type PageUseCase interface {
	List(context.Context, entity.SourceID, string) ([]entity.Page, error)
	Get(context.Context, entity.SourceID, string, string) (entity.PageDetail, error)
	ListTypes(context.Context, entity.SourceID, string) ([]entity.PageType, error)
	Create(context.Context, entity.SourceID, string, CreatePageInput) (entity.PageDetail, error)
	Update(context.Context, entity.SourceID, string, string, UpdatePageInput) (entity.PageDetail, error)
}
