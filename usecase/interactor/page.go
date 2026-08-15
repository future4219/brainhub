package interactor

import (
	"context"
	"errors"
	"strings"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

const DefaultPublicPageTypes = "decision,idea,research,note,concept,analysis,project,report,person,task-list"

type pageUseCase struct {
	pageRepository   output_port.PageRepository
	sourceRepository output_port.SourceRepository
	allowed          map[string]struct{}
}

func NewPageUseCase(pageRepository output_port.PageRepository, sourceRepository output_port.SourceRepository, publicPageTypes string) (input_port.PageUseCase, error) {
	if pageRepository == nil {
		return nil, errors.New("page repository is required")
	}
	if sourceRepository == nil {
		return nil, errors.New("source repository is required")
	}

	allowed := make(map[string]struct{})
	for _, pageType := range strings.Split(publicPageTypes, ",") {
		if pageType = strings.TrimSpace(pageType); pageType != "" {
			allowed[pageType] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil, errors.New("at least one public page type is required")
	}

	return &pageUseCase{
		pageRepository:   pageRepository,
		sourceRepository: sourceRepository,
		allowed:          allowed,
	}, nil
}

func (u *pageUseCase) List(ctx context.Context, sourceID entity.SourceID) ([]entity.Page, error) {
	sources, err := u.sourceRepository.ListSources(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, source := range sources {
		if source.ID == sourceID.String() {
			found = true
			break
		}
	}
	if !found {
		return nil, input_port.ErrSourceNotFound
	}

	pages, err := u.pageRepository.List(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	public := make([]entity.Page, 0, len(pages))
	for _, page := range pages {
		if _, ok := u.allowed[page.Type]; ok {
			public = append(public, page)
		}
	}
	return public, nil
}
