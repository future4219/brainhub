package interactor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"brainhub/domain/constructor"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/output_port"
)

type brainUseCase struct {
	brains       output_port.BrainRepository
	memberships  output_port.MembershipRepository
	transactions output_port.BrainTransactionManager
	provisioner  output_port.SourceProvisioner
	sources      output_port.SourceCatalog
	clock        output_port.Clock
	ids          output_port.IDGenerator
}

func NewBrainUseCase(
	brains output_port.BrainRepository,
	memberships output_port.MembershipRepository,
	transactions output_port.BrainTransactionManager,
	provisioner output_port.SourceProvisioner,
	sources output_port.SourceCatalog,
	clock output_port.Clock,
	ids output_port.IDGenerator,
) (input_port.BrainUseCase, error) {
	if brains == nil || memberships == nil || transactions == nil || provisioner == nil || sources == nil || clock == nil || ids == nil {
		return nil, errors.New("all brain dependencies are required")
	}
	return &brainUseCase{
		brains:       brains,
		memberships:  memberships,
		transactions: transactions,
		provisioner:  provisioner,
		sources:      sources,
		clock:        clock,
		ids:          ids,
	}, nil
}

func (u *brainUseCase) Create(ctx context.Context, ownerID string, input input_port.CreateBrainInput) (entity.Brain, error) {
	sourceID, err := constructor.NewSourceID(input.SourceID)
	if err != nil {
		return entity.Brain{}, fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
	}
	now := u.clock.Now()
	brain, err := constructor.NewBrainCreate(u.ids.New(), sourceID, input.Name, input.Description, input.Visibility, ownerID, now)
	if err != nil {
		return entity.Brain{}, fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
	}
	err = u.createRecords(ctx, brain, now)
	if errors.Is(err, output_port.ErrConflict) {
		return entity.Brain{}, input_port.ErrBrainAlreadyExists
	}
	if err != nil {
		return entity.Brain{}, fmt.Errorf("create brain records: %w", err)
	}

	if err := u.provisioner.Provision(ctx, sourceID); err != nil {
		failed, transitionErr := u.brains.TransitionBrain(ctx, brain.ID, entconst.BrainStateFailed, err.Error(), u.clock.Now())
		if transitionErr != nil {
			return brain, fmt.Errorf("mark failed brain after provisioning error: %w", transitionErr)
		}
		if errors.Is(err, output_port.ErrConflict) {
			return failed, input_port.ErrBrainAlreadyExists
		}
		return failed, fmt.Errorf("%w: %v", input_port.ErrProvisioningFailed, err)
	}

	ready, err := u.brains.TransitionBrain(ctx, brain.ID, entconst.BrainStateReady, "", u.clock.Now())
	if err != nil {
		return brain, fmt.Errorf("mark brain ready: %w", err)
	}
	return ready, nil
}

func (u *brainUseCase) Adopt(ctx context.Context, ownerID string, sourceID entity.SourceID, input input_port.AdoptBrainInput) (entity.Brain, error) {
	if _, err := u.brains.FindBrainBySourceID(ctx, sourceID); err == nil {
		return entity.Brain{}, input_port.ErrBrainAlreadyExists
	} else if !errors.Is(err, output_port.ErrNotFound) {
		return entity.Brain{}, fmt.Errorf("find brain before adopt: %w", err)
	}

	exists, err := u.sources.Exists(ctx, sourceID)
	if err != nil {
		return entity.Brain{}, fmt.Errorf("%w: %v", input_port.ErrSourceLookupFailed, err)
	}
	if !exists {
		return entity.Brain{}, input_port.ErrSourceNotFound
	}

	now := u.clock.Now()
	brain, err := constructor.NewBrainAdopt(u.ids.New(), sourceID, input.Name, input.Description, input.Visibility, ownerID, now)
	if err != nil {
		return entity.Brain{}, fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
	}
	if err := u.createRecords(ctx, brain, now); errors.Is(err, output_port.ErrConflict) {
		return entity.Brain{}, input_port.ErrBrainAlreadyExists
	} else if err != nil {
		return entity.Brain{}, fmt.Errorf("create adopted brain records: %w", err)
	}
	return brain, nil
}

func (u *brainUseCase) createRecords(ctx context.Context, brain entity.Brain, now time.Time) error {
	membership := constructor.NewOwnerMembership(u.ids.New(), brain.ID, brain.OwnerID, now)
	return u.transactions.WithinBrainTransaction(ctx, func(repositories output_port.BrainRepositories) error {
		if err := repositories.CreateBrain(ctx, brain); err != nil {
			return err
		}
		return repositories.CreateMembership(ctx, membership)
	})
}

func (u *brainUseCase) List(ctx context.Context, viewerID string) ([]entity.Brain, error) {
	brains, err := u.brains.ListBrains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list brains: %w", err)
	}
	roles, err := u.rolesForViewer(ctx, viewerID)
	if err != nil {
		return nil, err
	}

	visible := make([]entity.Brain, 0, len(brains))
	for _, brain := range brains {
		if canReadBrain(brain, roles[brain.ID]) {
			visible = append(visible, brain)
		}
	}
	return visible, nil
}

func (u *brainUseCase) Get(ctx context.Context, sourceID entity.SourceID, viewerID string) (entity.Brain, error) {
	brain, err := u.brains.FindBrainBySourceID(ctx, sourceID)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.Brain{}, input_port.ErrBrainNotFound
	}
	if err != nil {
		return entity.Brain{}, fmt.Errorf("find brain: %w", err)
	}
	roles, err := u.rolesForViewer(ctx, viewerID)
	if err != nil {
		return entity.Brain{}, err
	}
	if !canReadBrain(brain, roles[brain.ID]) {
		return entity.Brain{}, input_port.ErrBrainNotFound
	}
	return brain, nil
}

func (u *brainUseCase) rolesForViewer(ctx context.Context, viewerID string) (map[string]entity.Role, error) {
	roles := make(map[string]entity.Role)
	if viewerID == "" {
		return roles, nil
	}
	memberships, err := u.memberships.ListActiveMembershipsByUser(ctx, viewerID)
	if err != nil {
		return nil, fmt.Errorf("list viewer memberships: %w", err)
	}
	for _, membership := range memberships {
		roles[membership.BrainID] = membership.Role
	}
	return roles, nil
}

func canReadBrain(brain entity.Brain, role entity.Role) bool {
	if brain.State != entconst.BrainStateReady {
		return role == entity.RoleOwner
	}
	return brain.Visibility == entconst.VisibilityPublic || role != ""
}
