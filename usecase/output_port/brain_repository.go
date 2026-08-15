package output_port

import (
	"context"
	"time"

	"brainhub/domain/entconst"
	"brainhub/domain/entity"
)

type BrainRepository interface {
	CreateBrain(context.Context, entity.Brain) error
	ListBrains(context.Context) ([]entity.Brain, error)
	FindBrainBySourceID(context.Context, entity.SourceID) (entity.Brain, error)
	TransitionBrain(context.Context, string, entconst.BrainState, string, time.Time) (entity.Brain, error)
}

type MembershipRepository interface {
	CreateMembership(context.Context, entity.Membership) error
	ListActiveMembershipsByUser(context.Context, string) ([]entity.Membership, error)
}

type BrainRepositories interface {
	BrainRepository
	MembershipRepository
}

type BrainTransactionManager interface {
	WithinBrainTransaction(context.Context, func(BrainRepositories) error) error
}
