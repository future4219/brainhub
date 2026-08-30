package input_port

import (
	"context"
	"errors"
	"time"

	"brainhub/domain/entity"
)

var (
	ErrInvitationNotFound   = errors.New("invitation not found")
	ErrInvitationAccepted   = errors.New("invitation already accepted")
	ErrInvitationEmail      = errors.New("invitation email does not match")
	ErrMembershipExists     = errors.New("membership already exists")
	ErrMembershipNotFound   = errors.New("membership not found")
	ErrForbidden            = errors.New("forbidden")
	ErrCannotRevokeSelf     = errors.New("owners cannot revoke themselves")
	ErrUnsupportedClient    = errors.New("unsupported client label")
	ErrIssuedClientNotFound = errors.New("issued client not found")
	ErrGBrainAdmin          = errors.New("GBrain admin operation failed")
)

type CreateInvitationInput struct {
	Email     string
	Role      entity.Role
	ExpiresAt time.Time
}

type InvitationPreview struct {
	BrainName     string
	InvitedByName string
}

type AcceptedInvitation struct {
	Membership entity.Membership
	SourceID   entity.SourceID
	BrainName  string
}

type AccessUseCase interface {
	CreateInvitation(context.Context, entity.SourceID, string, CreateInvitationInput) (entity.Invitation, string, error)
	ListInvitations(context.Context, entity.SourceID, string) ([]entity.Invitation, error)
	RevokeInvitation(context.Context, string, string) error
	PreviewInvitation(context.Context, string) (InvitationPreview, error)
	AcceptInvitation(context.Context, string, entity.User) (AcceptedInvitation, error)
	IssueClient(context.Context, entity.SourceID, string, string) (entity.IssuedClient, error)
	ListClients(context.Context, entity.SourceID, string) ([]entity.IssuedClient, error)
	RevokeClient(context.Context, string, string) (entity.IssuedClient, error)
	RevokeMembership(context.Context, entity.SourceID, string, string) error
	ArchiveBrain(context.Context, entity.SourceID, string) error
}
