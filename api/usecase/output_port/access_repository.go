package output_port

import (
	"context"
	"time"

	"brainhub/domain/entity"
)

type AccessBrainRepository interface {
	FindBrainByID(context.Context, string) (entity.Brain, error)
	FindBrainBySourceID(context.Context, entity.SourceID) (entity.Brain, error)
	ArchiveBrain(context.Context, string, time.Time) error
}

type InvitationRepository interface {
	CreateInvitation(context.Context, entity.Invitation) (entity.Invitation, string, error)
	FindInvitationByID(context.Context, string) (entity.Invitation, error)
	FindInvitationByToken(context.Context, string) (entity.Invitation, error)
	ListInvitationsByBrain(context.Context, string) ([]entity.Invitation, error)
	AcceptInvitation(context.Context, string, string, time.Time) error
	RevokeInvitation(context.Context, string) error
	RevokePendingInvitationsByBrain(context.Context, string) error
}

type AccessMembershipRepository interface {
	CreateMembership(context.Context, entity.Membership) error
	FindMembership(context.Context, string, string) (entity.Membership, error)
	FindActiveMembership(context.Context, string, string) (entity.Membership, error)
	RevokeMembership(context.Context, string, time.Time) error
}

type IssuedClientRepository interface {
	CreateIssuedClient(context.Context, entity.IssuedClient) error
	FindIssuedClientByID(context.Context, string) (entity.IssuedClient, error)
	ListIssuedClientsByUserBrain(context.Context, string, string) ([]entity.IssuedClient, error)
	ListRevocableIssuedClientsByUserBrain(context.Context, string, string) ([]entity.IssuedClient, error)
	ListRevocableIssuedClientsByBrain(context.Context, string) ([]entity.IssuedClient, error)
	ActivateIssuedClient(context.Context, string, string, time.Time) (entity.IssuedClient, error)
	MarkIssuedClientOrphan(context.Context, string, *string, string) (entity.IssuedClient, error)
	MarkIssuedClientRevoked(context.Context, string, time.Time) (entity.IssuedClient, error)
}

type AccessRepositories interface {
	AccessBrainRepository
	InvitationRepository
	AccessMembershipRepository
	DeleteFailedBrain(context.Context, string) error
}

type AccessTransactionManager interface {
	WithinAccessTransaction(context.Context, func(AccessRepositories) error) error
}
