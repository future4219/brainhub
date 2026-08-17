package interactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/interactor"
	"brainhub/usecase/output_port"
)

type brainRepositoriesMock struct {
	brains      []entity.Brain
	memberships []entity.Membership
	writers     []entity.BrainWriterClient
}

func (r *brainRepositoriesMock) CreateBrain(_ context.Context, brain entity.Brain) error {
	for _, existing := range r.brains {
		if existing.SourceID == brain.SourceID {
			return output_port.ErrConflict
		}
	}
	r.brains = append(r.brains, brain)
	return nil
}

func (r *brainRepositoriesMock) ListBrains(context.Context) ([]entity.Brain, error) {
	return append([]entity.Brain(nil), r.brains...), nil
}

func (r *brainRepositoriesMock) FindBrainBySourceID(_ context.Context, sourceID entity.SourceID) (entity.Brain, error) {
	for _, brain := range r.brains {
		if brain.SourceID == sourceID {
			return brain, nil
		}
	}
	return entity.Brain{}, output_port.ErrNotFound
}

func (r *brainRepositoriesMock) TransitionBrain(_ context.Context, id string, state entconst.BrainState, reason string, now time.Time) (entity.Brain, error) {
	for i, brain := range r.brains {
		if brain.ID != id {
			continue
		}
		if brain.State != entconst.BrainStateProvisioning && brain.State != entconst.BrainStateDegraded {
			return entity.Brain{}, output_port.ErrConflict
		}
		brain.State = state
		brain.StateReason = reason
		brain.UpdatedAt = now
		r.brains[i] = brain
		return brain, nil
	}
	return entity.Brain{}, output_port.ErrNotFound
}

func (r *brainRepositoriesMock) CreateBrainWriterClient(_ context.Context, client entity.BrainWriterClient) error {
	r.writers = append(r.writers, client)
	return nil
}

func (r *brainRepositoriesMock) FindBrainWriterClient(_ context.Context, brainID string) (entity.BrainWriterClient, error) {
	for _, client := range r.writers {
		if client.BrainID == brainID {
			return client, nil
		}
	}
	return entity.BrainWriterClient{}, output_port.ErrNotFound
}

func (r *brainRepositoriesMock) ListIssuingBrainWriterClients(context.Context) ([]entity.BrainWriterClient, error) {
	return nil, nil
}

func (r *brainRepositoriesMock) ActivateBrainWriterClient(context.Context, string, string, []byte, time.Time) (entity.BrainWriterClient, error) {
	return entity.BrainWriterClient{}, nil
}

func (r *brainRepositoriesMock) MarkBrainWriterClientOrphan(context.Context, string, *string, string) (entity.BrainWriterClient, error) {
	return entity.BrainWriterClient{}, nil
}

func (r *brainRepositoriesMock) ResetBrainWriterClient(context.Context, string) error { return nil }

func (r *brainRepositoriesMock) CreateMembership(_ context.Context, membership entity.Membership) error {
	r.memberships = append(r.memberships, membership)
	return nil
}

func (r *brainRepositoriesMock) ListActiveMembershipsByUser(_ context.Context, userID string) ([]entity.Membership, error) {
	var memberships []entity.Membership
	for _, membership := range r.memberships {
		if membership.UserID == userID && membership.RevokedAt == nil {
			memberships = append(memberships, membership)
		}
	}
	return memberships, nil
}

func (r *brainRepositoriesMock) WithinBrainTransaction(_ context.Context, fn func(output_port.BrainRepositories) error) error {
	return fn(r)
}

type provisionerMock struct {
	err   error
	calls []entity.SourceID
}

type writerMock struct {
	err      error
	calls    []entity.SourceID
	reissues []entity.SourceID
}

func (w *writerMock) Provision(_ context.Context, _ string, sourceID entity.SourceID) error {
	w.calls = append(w.calls, sourceID)
	return w.err
}

func (w *writerMock) Reissue(_ context.Context, _ string, sourceID entity.SourceID) error {
	w.reissues = append(w.reissues, sourceID)
	return w.err
}

func (p *provisionerMock) Provision(_ context.Context, sourceID entity.SourceID) error {
	p.calls = append(p.calls, sourceID)
	return p.err
}

type sourceCatalogMock struct {
	exists bool
	err    error
	calls  []entity.SourceID
}

func (s *sourceCatalogMock) Exists(_ context.Context, sourceID entity.SourceID) (bool, error) {
	s.calls = append(s.calls, sourceID)
	return s.exists, s.err
}

type pagesMock struct {
	called  bool
	pages   []entity.Page
	details []entity.PageDetail
	puts    []entity.PageWrite
}

func (p *pagesMock) List(context.Context, entity.SourceID) ([]entity.Page, error) {
	p.called = true
	return p.pages, nil
}

func (p *pagesMock) Get(_ context.Context, _ entity.SourceID, slug string) (entity.PageDetail, error) {
	p.called = true
	for _, page := range p.details {
		if page.Slug == slug {
			return page, nil
		}
	}
	return entity.PageDetail{}, output_port.ErrNotFound
}

func (p *pagesMock) GetEditable(ctx context.Context, _ string, sourceID entity.SourceID, slug string) (entity.PageDetail, error) {
	return p.Get(ctx, sourceID, slug)
}

func (p *pagesMock) ListTypes(context.Context, string, entity.SourceID) ([]entity.PageType, error) {
	return []entity.PageType{{Name: "decision", Primitive: "concept"}}, nil
}

func (p *pagesMock) Put(_ context.Context, _ string, _ entity.SourceID, page entity.PageWrite) error {
	p.puts = append(p.puts, page)
	detail := entity.PageDetail{
		Page:          entity.Page{Slug: page.Slug, Title: page.Title, Type: page.Type},
		CompiledTruth: page.CompiledTruth, Timeline: page.Timeline, Tags: page.Tags,
		SupersededBy: page.SupersededBy, Frontmatter: page.Frontmatter,
	}
	for i := range p.details {
		if p.details[i].Slug == page.Slug {
			p.details[i] = detail
			return nil
		}
	}
	p.details = append(p.details, detail)
	return nil
}

func TestBrainCreateStateMachine(t *testing.T) {
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	t.Run("ready", func(t *testing.T) {
		repositories := &brainRepositoriesMock{}
		provisioner := &provisionerMock{}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, provisioner, &sourceCatalogMock{}, &writerMock{}, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		brain, err := useCase.Create(context.Background(), "owner", input_port.CreateBrainInput{SourceID: "accounting", Name: "Accounting"})
		if err != nil {
			t.Fatal(err)
		}
		if brain.State != entconst.BrainStateReady || len(repositories.memberships) != 1 || repositories.memberships[0].Role != entity.RoleOwner {
			t.Fatalf("brain/membership = %+v %+v", brain, repositories.memberships)
		}
		if len(provisioner.calls) != 1 || provisioner.calls[0] != "accounting" {
			t.Fatalf("provision calls = %v", provisioner.calls)
		}
	})

	t.Run("failed is durable", func(t *testing.T) {
		repositories := &brainRepositoriesMock{}
		provisioner := &provisionerMock{err: errors.New("shim unavailable")}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, provisioner, &sourceCatalogMock{}, &writerMock{}, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		brain, err := useCase.Create(context.Background(), "owner", input_port.CreateBrainInput{SourceID: "failed-brain", Name: "Failed"})
		if !errors.Is(err, input_port.ErrProvisioningFailed) {
			t.Fatalf("error = %v", err)
		}
		if brain.State != entconst.BrainStateFailed || brain.StateReason != "shim unavailable" || repositories.brains[0].State != entconst.BrainStateFailed {
			t.Fatalf("failed brain = %+v stored=%+v", brain, repositories.brains[0])
		}
		if _, err := repositories.TransitionBrain(context.Background(), brain.ID, entconst.BrainStateReady, "", now); !errors.Is(err, output_port.ErrConflict) {
			t.Fatalf("failed to ready transition error = %v", err)
		}
	})

	t.Run("writer failure is degraded and recoverable", func(t *testing.T) {
		repositories := &brainRepositoriesMock{}
		writer := &writerMock{err: errors.New("writer unavailable")}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, &provisionerMock{}, &sourceCatalogMock{}, writer, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		brain, err := useCase.Create(context.Background(), "owner", input_port.CreateBrainInput{SourceID: "degraded-brain", Name: "Degraded"})
		if !errors.Is(err, input_port.ErrProvisioningFailed) || brain.State != entconst.BrainStateDegraded {
			t.Fatalf("brain/error = %+v %v", brain, err)
		}
		if len(repositories.writers) != 1 || repositories.writers[0].State != entity.WriterClientStateIssuing {
			t.Fatalf("writer row = %+v", repositories.writers)
		}
	})
}

func TestOnlyOwnerCanReissueWriter(t *testing.T) {
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	repositories := &brainRepositoriesMock{
		brains: []entity.Brain{{ID: "brain-id", SourceID: "brainhub", State: entconst.BrainStateDegraded}},
		memberships: []entity.Membership{
			{BrainID: "brain-id", UserID: "owner", Role: entity.RoleOwner},
			{BrainID: "brain-id", UserID: "editor", Role: entity.RoleEditor},
		},
	}
	writer := &writerMock{}
	useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, &provisionerMock{}, &sourceCatalogMock{}, writer, fixedClock{now}, &sequenceIDs{})
	if err != nil {
		t.Fatal(err)
	}
	if err := useCase.ReissueWriter(context.Background(), "brainhub", "editor"); !errors.Is(err, input_port.ErrForbidden) {
		t.Fatalf("editor error = %v", err)
	}
	if len(writer.reissues) != 0 {
		t.Fatal("writer was reissued for editor")
	}
	if err := useCase.ReissueWriter(context.Background(), "brainhub", "owner"); err != nil {
		t.Fatal(err)
	}
	if len(writer.reissues) != 1 || repositories.brains[0].State != entconst.BrainStateReady {
		t.Fatalf("reissues/brain = %v %+v", writer.reissues, repositories.brains[0])
	}
}

func TestBrainAdopt(t *testing.T) {
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

	t.Run("existing source becomes ready and private", func(t *testing.T) {
		repositories := &brainRepositoriesMock{}
		provisioner := &provisionerMock{}
		sources := &sourceCatalogMock{exists: true}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, provisioner, sources, &writerMock{}, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		brain, err := useCase.Adopt(context.Background(), "owner", "existing-source", input_port.AdoptBrainInput{Name: "Existing"})
		if err != nil {
			t.Fatal(err)
		}
		if brain.State != entconst.BrainStateReady || brain.Visibility != entconst.VisibilityPrivate {
			t.Fatalf("adopted brain = %+v", brain)
		}
		if len(repositories.memberships) != 1 || repositories.memberships[0].Role != entity.RoleOwner {
			t.Fatalf("memberships = %+v", repositories.memberships)
		}
		if len(provisioner.calls) != 0 {
			t.Fatalf("adopt called provisioner: %v", provisioner.calls)
		}
		if len(sources.calls) != 1 || sources.calls[0] != "existing-source" {
			t.Fatalf("source lookups = %v", sources.calls)
		}
	})

	t.Run("missing source", func(t *testing.T) {
		repositories := &brainRepositoriesMock{}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, &provisionerMock{}, &sourceCatalogMock{}, &writerMock{}, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = useCase.Adopt(context.Background(), "owner", "missing", input_port.AdoptBrainInput{Name: "Missing"})
		if !errors.Is(err, input_port.ErrSourceNotFound) || len(repositories.brains) != 0 {
			t.Fatalf("error/brains = %v %+v", err, repositories.brains)
		}
	})

	t.Run("registered brain wins without GBrain lookup", func(t *testing.T) {
		repositories := &brainRepositoriesMock{brains: []entity.Brain{{SourceID: "registered"}}}
		sources := &sourceCatalogMock{exists: true}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, &provisionerMock{}, sources, &writerMock{}, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = useCase.Adopt(context.Background(), "owner", "registered", input_port.AdoptBrainInput{Name: "Registered"})
		if !errors.Is(err, input_port.ErrBrainAlreadyExists) || len(sources.calls) != 0 {
			t.Fatalf("error/lookups = %v %v", err, sources.calls)
		}
	})

	t.Run("lookup failure is not not-found", func(t *testing.T) {
		repositories := &brainRepositoriesMock{}
		sources := &sourceCatalogMock{err: errors.New("upstream unavailable")}
		useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, &provisionerMock{}, sources, &writerMock{}, fixedClock{now}, &sequenceIDs{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = useCase.Adopt(context.Background(), "owner", "existing", input_port.AdoptBrainInput{Name: "Existing"})
		if !errors.Is(err, input_port.ErrSourceLookupFailed) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestBrainVisibilityUsesMembership(t *testing.T) {
	readyPublic := entity.Brain{ID: "public", SourceID: "public", Visibility: entconst.VisibilityPublic, State: entconst.BrainStateReady}
	readyPrivate := entity.Brain{ID: "private", SourceID: "private", Visibility: entconst.VisibilityPrivate, State: entconst.BrainStateReady, OwnerID: "owner-column-only"}
	failed := entity.Brain{ID: "failed", SourceID: "failed", Visibility: entconst.VisibilityPublic, State: entconst.BrainStateFailed, OwnerID: "viewer"}
	repositories := &brainRepositoriesMock{
		brains: []entity.Brain{readyPublic, readyPrivate, failed},
		memberships: []entity.Membership{
			{BrainID: "private", UserID: "viewer", Role: entity.RoleReader},
			{BrainID: "failed", UserID: "owner-member", Role: entity.RoleOwner},
		},
	}
	useCase, err := interactor.NewBrainUseCase(repositories, repositories, repositories, &provisionerMock{}, &sourceCatalogMock{}, &writerMock{}, fixedClock{}, &sequenceIDs{})
	if err != nil {
		t.Fatal(err)
	}

	anonymous, err := useCase.List(context.Background(), "")
	if err != nil || len(anonymous) != 1 || anonymous[0].ID != "public" {
		t.Fatalf("anonymous = %+v %v", anonymous, err)
	}
	viewer, err := useCase.List(context.Background(), "viewer")
	if err != nil || len(viewer) != 2 {
		t.Fatalf("viewer = %+v %v", viewer, err)
	}
	owner, err := useCase.List(context.Background(), "owner-member")
	if err != nil || len(owner) != 2 || owner[1].ID != "failed" {
		t.Fatalf("owner = %+v %v", owner, err)
	}
	if _, err := useCase.Get(context.Background(), "failed", "viewer"); !errors.Is(err, input_port.ErrBrainNotFound) {
		t.Fatalf("owner_id granted access without membership: %v", err)
	}
}

func TestPageAccessIsCheckedBeforeGBrain(t *testing.T) {
	repositories := &brainRepositoriesMock{brains: []entity.Brain{{
		ID: "private", SourceID: "private", Visibility: entconst.VisibilityPrivate, State: entconst.BrainStateReady,
	}}}
	pages := &pagesMock{pages: []entity.Page{
		{Slug: "allowed", Type: "decision"},
		{Slug: "internal", Type: "extract_receipt"},
	}}
	useCase, err := interactor.NewPageUseCase(pages, pages, repositories, repositories, interactor.DefaultPublicPageTypes)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.List(context.Background(), "private", "stranger"); !errors.Is(err, input_port.ErrBrainNotFound) {
		t.Fatalf("private access error = %v", err)
	}
	if pages.called {
		t.Fatal("GBrain was called before authorization")
	}
	repositories.memberships = append(repositories.memberships, entity.Membership{BrainID: "private", UserID: "reader", Role: entity.RoleReader})
	visible, err := useCase.List(context.Background(), "private", "reader")
	if err != nil || len(visible) != 2 {
		t.Fatalf("visible pages = %+v %v", visible, err)
	}
	pages.details = []entity.PageDetail{
		{Page: entity.Page{Slug: "allowed", Type: "decision"}, CompiledTruth: "body"},
		{Page: entity.Page{Slug: "internal", Type: "extract_receipt"}, CompiledTruth: "secret"},
	}
	detail, err := useCase.Get(context.Background(), "private", "allowed", "reader")
	if err != nil || detail.CompiledTruth != "body" {
		t.Fatalf("page detail = %+v %v", detail, err)
	}
	if internal, err := useCase.Get(context.Background(), "private", "internal", "reader"); err != nil || internal.CompiledTruth != "secret" {
		t.Fatalf("internal page = %+v %v", internal, err)
	}
}
