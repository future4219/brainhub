package interactor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"brainhub/domain/entity"
	"brainhub/domain/validation"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

const DefaultPublicPageTypes = "decision,idea,research,note,concept,analysis,project,report,person,task-list"

type pageUseCase struct {
	pages       output_port.PageRepository
	editor      output_port.PageEditorRepository
	brains      output_port.BrainRepository
	memberships output_port.MembershipRepository
	publicTypes map[string]struct{}
}

func NewPageUseCase(
	pages output_port.PageRepository,
	editor output_port.PageEditorRepository,
	brains output_port.BrainRepository,
	memberships output_port.MembershipRepository,
	publicPageTypes string,
) (input_port.PageUseCase, error) {
	if pages == nil || editor == nil || brains == nil || memberships == nil {
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
	return &pageUseCase{pages: pages, editor: editor, brains: brains, memberships: memberships, publicTypes: allowed}, nil
}

func (u *pageUseCase) List(ctx context.Context, sourceID entity.SourceID, viewerID string) ([]entity.Page, error) {
	_, role, err := u.readAccess(ctx, sourceID, viewerID)
	if err != nil {
		return nil, err
	}
	pages, err := u.pages.List(ctx, sourceID)
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
	var page entity.PageDetail
	if role == entity.RoleOwner || role == entity.RoleEditor {
		page, err = u.editor.GetEditable(ctx, brain.ID, sourceID, slug)
	} else {
		page, err = u.pages.Get(ctx, sourceID, slug)
	}
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

func (u *pageUseCase) ListTypes(ctx context.Context, sourceID entity.SourceID, viewerID string) ([]entity.PageType, error) {
	brain, _, err := u.writeAccess(ctx, sourceID, viewerID)
	if err != nil {
		return nil, err
	}
	return u.editor.ListTypes(ctx, brain.ID, sourceID)
}

func (u *pageUseCase) Create(ctx context.Context, sourceID entity.SourceID, viewerID string, input input_port.CreatePageInput) (entity.PageDetail, error) {
	brain, _, err := u.writeAccess(ctx, sourceID, viewerID)
	if err != nil {
		return entity.PageDetail{}, err
	}
	if err := validatePageInput(input.Slug, input.Title, input.CompiledTruth, input.TimelineEntry, true); err != nil {
		return entity.PageDetail{}, err
	}
	if _, err := u.editor.GetEditable(ctx, brain.ID, sourceID, input.Slug); err == nil {
		return entity.PageDetail{}, input_port.ErrPageAlreadyExists
	} else if !errors.Is(err, output_port.ErrNotFound) {
		return entity.PageDetail{}, err
	}
	if err := u.validateType(ctx, brain.ID, sourceID, input.Type); err != nil {
		return entity.PageDetail{}, err
	}
	target, err := u.validateTarget(ctx, brain.ID, sourceID, input.Slug, input.SupersededBy)
	if err != nil {
		return entity.PageDetail{}, err
	}
	page := entity.PageWrite{
		Slug: input.Slug, Title: strings.TrimSpace(input.Title), Type: input.Type,
		Tags: normalizeTags(input.Tags), SupersededBy: target,
		CompiledTruth: input.CompiledTruth, Timeline: strings.TrimSpace(input.TimelineEntry),
		Frontmatter: map[string]any{},
	}
	if err := u.editor.Put(ctx, brain.ID, sourceID, page); err != nil {
		return entity.PageDetail{}, err
	}
	return u.editor.GetEditable(ctx, brain.ID, sourceID, input.Slug)
}

func (u *pageUseCase) Update(ctx context.Context, sourceID entity.SourceID, slug, viewerID string, input input_port.UpdatePageInput) (entity.PageDetail, error) {
	brain, _, err := u.writeAccess(ctx, sourceID, viewerID)
	if err != nil {
		return entity.PageDetail{}, err
	}
	if err := validatePageInput(slug, input.Title, input.CompiledTruth, input.TimelineEntry, false); err != nil {
		return entity.PageDetail{}, err
	}
	current, err := u.editor.GetEditable(ctx, brain.ID, sourceID, slug)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.PageDetail{}, input_port.ErrPageNotFound
	}
	if err != nil {
		return entity.PageDetail{}, err
	}
	if err := u.validateType(ctx, brain.ID, sourceID, input.Type); err != nil {
		return entity.PageDetail{}, err
	}
	target, err := u.validateTarget(ctx, brain.ID, sourceID, slug, input.SupersededBy)
	if err != nil {
		return entity.PageDetail{}, err
	}
	timeline := current.Timeline
	if entry := strings.TrimSpace(input.TimelineEntry); entry != "" {
		if timeline != "" && !strings.HasSuffix(timeline, "\n") {
			timeline += "\n"
		}
		timeline += entry
	}
	page := entity.PageWrite{
		Slug: slug, Title: strings.TrimSpace(input.Title), Type: input.Type,
		Tags: normalizeTags(input.Tags), SupersededBy: target,
		CompiledTruth: input.CompiledTruth, Timeline: timeline, Frontmatter: current.Frontmatter,
	}
	if err := u.editor.Put(ctx, brain.ID, sourceID, page); err != nil {
		return entity.PageDetail{}, err
	}
	return u.editor.GetEditable(ctx, brain.ID, sourceID, slug)
}

func validatePageInput(slug, title, compiledTruth, timelineEntry string, timelineRequired bool) error {
	for _, err := range []error{
		validation.ValidatePageSlug(slug), validation.ValidatePageTitle(title),
		validation.ValidateCompiledTruth(compiledTruth), validation.ValidateTimelineEntry(timelineEntry, timelineRequired),
	} {
		if err != nil {
			return fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
		}
	}
	return nil
}

func (u *pageUseCase) validateType(ctx context.Context, brainID string, sourceID entity.SourceID, requested string) error {
	types, err := u.editor.ListTypes(ctx, brainID, sourceID)
	if err != nil {
		return err
	}
	for _, pageType := range types {
		if pageType.Name == requested {
			return nil
		}
	}
	return fmt.Errorf("%w: type is not declared by the active GBrain schema pack", input_port.ErrInvalidInput)
}

func (u *pageUseCase) validateTarget(ctx context.Context, brainID string, sourceID entity.SourceID, slug string, requested *string) (*string, error) {
	if requested == nil || strings.TrimSpace(*requested) == "" {
		return nil, nil
	}
	target := strings.TrimSpace(*requested)
	if target == slug {
		return nil, fmt.Errorf("%w: superseded_by must point to another page", input_port.ErrInvalidInput)
	}
	if err := validation.ValidatePageSlug(target); err != nil {
		return nil, fmt.Errorf("%w: invalid superseded_by", input_port.ErrInvalidInput)
	}
	if _, err := u.editor.GetEditable(ctx, brainID, sourceID, target); errors.Is(err, output_port.ErrNotFound) {
		return nil, fmt.Errorf("%w: superseded_by page does not exist", input_port.ErrInvalidInput)
	} else if err != nil {
		return nil, err
	}
	return &target, nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized
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

func (u *pageUseCase) writeAccess(ctx context.Context, sourceID entity.SourceID, viewerID string) (entity.Brain, entity.Membership, error) {
	brain, err := u.brains.FindBrainBySourceID(ctx, sourceID)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.Brain{}, entity.Membership{}, input_port.ErrBrainNotFound
	}
	if err != nil {
		return entity.Brain{}, entity.Membership{}, fmt.Errorf("find brain: %w", err)
	}
	if viewerID == "" {
		return entity.Brain{}, entity.Membership{}, input_port.ErrBrainNotFound
	}
	memberships, err := u.memberships.ListActiveMembershipsByUser(ctx, viewerID)
	if err != nil {
		return entity.Brain{}, entity.Membership{}, fmt.Errorf("list viewer memberships: %w", err)
	}
	for _, membership := range memberships {
		if membership.BrainID != brain.ID {
			continue
		}
		if !membership.CanWrite() {
			return entity.Brain{}, entity.Membership{}, input_port.ErrForbidden
		}
		return brain, membership, nil
	}
	return entity.Brain{}, entity.Membership{}, input_port.ErrBrainNotFound
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
