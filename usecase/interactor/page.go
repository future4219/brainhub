package interactor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

const DefaultPublicPageTypes = "decision,idea,research,note,concept,analysis,project,report,person,task-list"

type pageUseCase struct {
	pageRepository output_port.PageRepository
	brains         output_port.BrainRepository
	memberships    output_port.MembershipRepository
	allowed        map[string]struct{}
}

func NewPageUseCase(
	pageRepository output_port.PageRepository,
	brains output_port.BrainRepository,
	memberships output_port.MembershipRepository,
	publicPageTypes string,
) (input_port.PageUseCase, error) {
	if pageRepository == nil || brains == nil || memberships == nil {
		return nil, errors.New("all page dependencies are required")
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
		pageRepository: pageRepository,
		brains:         brains,
		memberships:    memberships,
		allowed:        allowed,
	}, nil
}

func (u *pageUseCase) List(ctx context.Context, sourceID entity.SourceID, viewerID string) ([]entity.Page, error) {
	if err := u.requireReadAccess(ctx, sourceID, viewerID); err != nil {
		return nil, err
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

func (u *pageUseCase) Get(ctx context.Context, sourceID entity.SourceID, slug, viewerID string) (entity.PageDetail, error) {
	if err := u.requireReadAccess(ctx, sourceID, viewerID); err != nil {
		return entity.PageDetail{}, err
	}
	page, err := u.pageRepository.Get(ctx, sourceID, slug)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.PageDetail{}, input_port.ErrPageNotFound
	}
	if err != nil {
		return entity.PageDetail{}, err
	}
	if _, ok := u.allowed[page.Type]; !ok {
		return entity.PageDetail{}, input_port.ErrPageNotFound
	}
	return page, nil
}

func (u *pageUseCase) requireReadAccess(ctx context.Context, sourceID entity.SourceID, viewerID string) error {
	brain, err := u.brains.FindBrainBySourceID(ctx, sourceID)
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.ErrBrainNotFound
	}
	if err != nil {
		return fmt.Errorf("find brain: %w", err)
	}

	role := entity.Role("")
	if viewerID != "" {
		memberships, err := u.memberships.ListActiveMembershipsByUser(ctx, viewerID)
		if err != nil {
			return fmt.Errorf("list viewer memberships: %w", err)
		}
		for _, membership := range memberships {
			if membership.BrainID == brain.ID {
				role = membership.Role
				break
			}
		}
	}
	if !canReadBrain(brain, role) {
		return input_port.ErrBrainNotFound
	}
	return nil
}
