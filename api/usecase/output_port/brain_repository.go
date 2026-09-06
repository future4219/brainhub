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

// BrainWriterClientRepository stores the source credential record in Brainhub DB.
type BrainWriterClientRepository interface {
	CreateBrainWriterClient(context.Context, entity.BrainWriterClient) error
	FindBrainWriterClient(context.Context, string) (entity.BrainWriterClient, error)
	ListIssuingBrainWriterClients(context.Context) ([]entity.BrainWriterClient, error)
	ActivateBrainWriterClient(context.Context, string, string, []byte, time.Time) (entity.BrainWriterClient, error)
	MarkBrainWriterClientOrphan(context.Context, string, *string, string) (entity.BrainWriterClient, error)
	ResetBrainWriterClient(context.Context, string) error
}

type BrainRepositories interface {
	BrainRepository
	MembershipRepository
	BrainWriterClientRepository
}

type BrainTransactionManager interface {
	WithinBrainTransaction(context.Context, func(BrainRepositories) error) error
}
