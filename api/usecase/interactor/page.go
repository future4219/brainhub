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
	pages       output_port.PageRepository
	brains      output_port.BrainRepository
	memberships output_port.MembershipRepository
	publicTypes map[string]struct{}
}

func NewPageUseCase(
	pages output_port.PageRepository,
	brains output_port.BrainRepository,
	memberships output_port.MembershipRepository,
	publicPageTypes string,
) (input_port.PageUseCase, error) {
	if pages == nil || brains == nil || memberships == nil {
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
	return &pageUseCase{pages: pages, brains: brains, memberships: memberships, publicTypes: allowed}, nil
}

func (u *pageUseCase) List(ctx context.Context, sourceID entity.SourceID, viewerID string) ([]entity.Page, error) {
	brain, role, err := u.readAccess(ctx, sourceID, viewerID)
	if err != nil {
		return nil, err
	}
	pages, err := u.pages.List(ctx, brain.ID, sourceID)
	if err != nil {
		return nil, err
	}
	if role != "" {
		return pages, nil
	}
	public := make([]entity.Page, 0, len(pages))
	for _, page := range pages {
		if _, ok := u.publicTypes[page.Type]; ok {
			public = append(public, page)
		}
	}
	return public, nil
}

func (u *pageUseCase) Get(ctx context.Context, sourceID entity.SourceID, slug, viewerID string) (entity.PageDetail, error) {
	brain, role, err := u.readAccess(ctx, sourceID, viewerID)
	if err != nil {
		return entity.PageDetail{}, err
	}
	page, err := u.pages.Get(ctx, brain.ID, sourceID, slug)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.PageDetail{}, input_port.ErrPageNotFound
	}
	if err != nil {
		return entity.PageDetail{}, err
	}
	if role == "" {
		if _, ok := u.publicTypes[page.Type]; !ok {
			return entity.PageDetail{}, input_port.ErrPageNotFound
		}
	}
	return page, nil
}

func (u *pageUseCase) readAccess(ctx context.Context, sourceID entity.SourceID, viewerID string) (entity.Brain, entity.Role, error) {
	brain, err := u.brains.FindBrainBySourceID(ctx, sourceID)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.Brain{}, "", input_port.ErrBrainNotFound
	}
	if err != nil {
		return entity.Brain{}, "", fmt.Errorf("find brain: %w", err)
	}
	role, err := u.viewerRole(ctx, brain.ID, viewerID)
	if err != nil {
		return entity.Brain{}, "", err
	}
	if !canReadBrain(brain, role) {
		return entity.Brain{}, "", input_port.ErrBrainNotFound
	}
	return brain, role, nil
}

func (u *pageUseCase) viewerRole(ctx context.Context, brainID, viewerID string) (entity.Role, error) {
	if viewerID == "" {
		return "", nil
	}
	memberships, err := u.memberships.ListActiveMembershipsByUser(ctx, viewerID)
	if err != nil {
		return "", fmt.Errorf("list viewer memberships: %w", err)
	}
	for _, membership := range memberships {
		if membership.BrainID == brainID {
			return membership.Role, nil
		}
	}
	return "", nil
}
