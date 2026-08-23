package interactor_test

import (
	"context"
	"errors"
	"testing"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/interactor"
)

func TestPageUpdateAppendsTimelineAndPreservesFrontmatter(t *testing.T) {
	repositories := &brainRepositoriesMock{
		brains: []entity.Brain{{
			ID: "brain-id", SourceID: "brainhub", Visibility: entconst.VisibilityPrivate, State: entconst.BrainStateReady,
		}},
		memberships: []entity.Membership{{BrainID: "brain-id", UserID: "editor", Role: entity.RoleEditor}},
	}
	pages := &pagesMock{details: []entity.PageDetail{{
		Page:          entity.Page{Slug: "decisions/core", Title: "Old", Type: "decision"},
		CompiledTruth: "old truth", Timeline: "2026-08-16 initial",
		Frontmatter: map[string]any{"owner": "brainhub"},
	}}}
	useCase, err := interactor.NewPageUseCase(pages, repositories, repositories, interactor.DefaultPublicPageTypes)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := useCase.Update(context.Background(), "brainhub", "decisions/core", "editor", input_port.UpdatePageInput{
		Title: "Core", Type: "decision", Tags: []string{"product", "product"},
		CompiledTruth: "new truth", TimelineEntry: "2026-08-17 changed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages.puts) != 1 {
		t.Fatalf("puts = %d; want 1", len(pages.puts))
	}
	written := pages.puts[0]
	if written.Timeline != "2026-08-16 initial\n2026-08-17 changed" {
		t.Fatalf("timeline = %q", written.Timeline)
	}
	if written.CompiledTruth != "new truth" || written.Frontmatter["owner"] != "brainhub" {
		t.Fatalf("written page = %#v", written)
	}
	if len(written.Tags) != 1 || updated.Timeline != written.Timeline {
		t.Fatalf("tags/detail = %#v %#v", written.Tags, updated)
	}
}

func TestPageWriteAuthorizationBeforeGBrain(t *testing.T) {
	repositories := &brainRepositoriesMock{
		brains: []entity.Brain{{
			ID: "brain-id", SourceID: "brainhub", Visibility: entconst.VisibilityPublic, State: entconst.BrainStateReady,
		}},
		memberships: []entity.Membership{{BrainID: "brain-id", UserID: "reader", Role: entity.RoleReader}},
	}
	pages := &pagesMock{}
	useCase, err := interactor.NewPageUseCase(pages, repositories, repositories, interactor.DefaultPublicPageTypes)
	if err != nil {
		t.Fatal(err)
	}
	input := input_port.UpdatePageInput{Title: "Core", Type: "decision", CompiledTruth: "truth"}
	if _, err := useCase.Update(context.Background(), "brainhub", "decisions/core", "reader", input); !errors.Is(err, input_port.ErrForbidden) {
		t.Fatalf("reader error = %v; want forbidden", err)
	}
	if _, err := useCase.Update(context.Background(), "brainhub", "decisions/core", "stranger", input); !errors.Is(err, input_port.ErrBrainNotFound) {
		t.Fatalf("stranger error = %v; want not found", err)
	}
	if pages.called || len(pages.puts) != 0 {
		t.Fatal("GBrain was called before write authorization")
	}
}

func TestPageCreateUsesSchemaTypesAndRejectsDuplicate(t *testing.T) {
	repositories := &brainRepositoriesMock{
		brains:      []entity.Brain{{ID: "brain-id", SourceID: "empty", Visibility: entconst.VisibilityPrivate, State: entconst.BrainStateReady}},
		memberships: []entity.Membership{{BrainID: "brain-id", UserID: "owner", Role: entity.RoleOwner}},
	}
	pages := &pagesMock{}
	useCase, err := interactor.NewPageUseCase(pages, repositories, repositories, interactor.DefaultPublicPageTypes)
	if err != nil {
		t.Fatal(err)
	}
	input := input_port.CreatePageInput{
		Slug: "notes/first", Title: "First", Type: "decision",
		CompiledTruth: "truth", TimelineEntry: "2026-08-17 initial",
	}
	if _, err := useCase.Create(context.Background(), "empty", "owner", input); err != nil {
		t.Fatal(err)
	}
	if len(pages.puts) != 1 {
		t.Fatalf("puts = %d", len(pages.puts))
	}
	if _, err := useCase.Create(context.Background(), "empty", "owner", input); !errors.Is(err, input_port.ErrPageAlreadyExists) {
		t.Fatalf("duplicate error = %v", err)
	}
	input.Slug = "notes/second"
	input.Type = "invented"
	if _, err := useCase.Create(context.Background(), "empty", "owner", input); !errors.Is(err, input_port.ErrInvalidInput) {
		t.Fatalf("invalid type error = %v", err)
	}
}
