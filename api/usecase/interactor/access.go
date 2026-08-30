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

const (
	claudeWebLabel       = "claude-web"
	claudeWebRedirectURI = "https://claude.ai/api/mcp/auth_callback"
	maxStateReasonBytes  = 2000
)

type accessUseCase struct {
	brains       output_port.AccessBrainRepository
	users        output_port.UserRepository
	invitations  output_port.InvitationRepository
	memberships  output_port.AccessMembershipRepository
	clients      output_port.IssuedClientRepository
	transactions output_port.AccessTransactionManager
	admin        output_port.GBrainAdmin
	clock        output_port.Clock
	ids          output_port.IDGenerator
}

func NewAccessUseCase(
	brains output_port.AccessBrainRepository,
	users output_port.UserRepository,
	invitations output_port.InvitationRepository,
	memberships output_port.AccessMembershipRepository,
	clients output_port.IssuedClientRepository,
	transactions output_port.AccessTransactionManager,
	admin output_port.GBrainAdmin,
	clock output_port.Clock,
	ids output_port.IDGenerator,
) (input_port.AccessUseCase, error) {
	if brains == nil || users == nil || invitations == nil || memberships == nil || clients == nil || transactions == nil || admin == nil || clock == nil || ids == nil {
		return nil, errors.New("all access dependencies are required")
	}
	return &accessUseCase{
		brains: brains, users: users, invitations: invitations, memberships: memberships,
		clients: clients, transactions: transactions, admin: admin, clock: clock, ids: ids,
	}, nil
}

func (u *accessUseCase) CreateInvitation(ctx context.Context, sourceID entity.SourceID, actorID string, input input_port.CreateInvitationInput) (entity.Invitation, string, error) {
	brain, membership, err := u.brainAndMembership(ctx, sourceID, actorID)
	if err != nil {
		return entity.Invitation{}, "", err
	}
	if membership.Role != entity.RoleOwner {
		return entity.Invitation{}, "", input_port.ErrForbidden
	}
	if !input.Role.IsValid() || !input.ExpiresAt.After(u.clock.Now()) {
		return entity.Invitation{}, "", input_port.ErrInvalidInput
	}
	var email *string
	if strings.TrimSpace(input.Email) != "" {
		normalized, err := validation.NormalizeEmail(input.Email)
		if err != nil {
			return entity.Invitation{}, "", fmt.Errorf("%w: %v", input_port.ErrInvalidInput, err)
		}
		email = &normalized
	}
	invitation := entity.Invitation{
		ID: u.ids.New(), BrainID: brain.ID, Email: email, Role: input.Role,
		InvitedBy: actorID, State: entity.InvitationStatePending,
		ExpiresAt: input.ExpiresAt, CreatedAt: u.clock.Now(),
	}
	created, rawToken, err := u.invitations.CreateInvitation(ctx, invitation)
	if err != nil {
		return entity.Invitation{}, "", fmt.Errorf("create invitation: %w", err)
	}
	return created, rawToken, nil
}

func (u *accessUseCase) ListInvitations(ctx context.Context, sourceID entity.SourceID, actorID string) ([]entity.Invitation, error) {
	brain, membership, err := u.brainAndMembership(ctx, sourceID, actorID)
	if err != nil {
		return nil, err
	}
	if membership.Role != entity.RoleOwner {
		return nil, input_port.ErrForbidden
	}
	invitations, err := u.invitations.ListInvitationsByBrain(ctx, brain.ID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	now := u.clock.Now()
	for i := range invitations {
		if invitations[i].State == entity.InvitationStatePending && !now.Before(invitations[i].ExpiresAt) {
			invitations[i].State = entity.InvitationStateExpired
		}
	}
	return invitations, nil
}

func (u *accessUseCase) RevokeInvitation(ctx context.Context, invitationID, actorID string) error {
	invitation, err := u.invitations.FindInvitationByID(ctx, invitationID)
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.ErrInvitationNotFound
	}
	if err != nil {
		return fmt.Errorf("find invitation: %w", err)
	}
	brain, err := u.brains.FindBrainByID(ctx, invitation.BrainID)
	if err != nil {
		return fmt.Errorf("find invitation brain: %w", err)
	}
	membership, err := u.memberships.FindActiveMembership(ctx, brain.ID, actorID)
	if errors.Is(err, output_port.ErrNotFound) || (err == nil && membership.Role != entity.RoleOwner) {
		return input_port.ErrForbidden
	}
	if err != nil {
		return fmt.Errorf("find owner membership: %w", err)
	}
	if invitation.State == entity.InvitationStateAccepted {
		return input_port.ErrInvitationAccepted
	}
	if !invitation.IsAcceptable(u.clock.Now()) {
		return input_port.ErrInvitationNotFound
	}
	if err := u.invitations.RevokeInvitation(ctx, invitation.ID); err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	return nil
}

func (u *accessUseCase) PreviewInvitation(ctx context.Context, rawToken string) (input_port.InvitationPreview, error) {
	invitation, err := u.invitations.FindInvitationByToken(ctx, rawToken)
	if errors.Is(err, output_port.ErrNotFound) || (err == nil && !invitation.IsAcceptable(u.clock.Now())) {
		return input_port.InvitationPreview{}, input_port.ErrInvitationNotFound
	}
	if err != nil {
		return input_port.InvitationPreview{}, fmt.Errorf("find invitation: %w", err)
	}
	brain, err := u.brains.FindBrainByID(ctx, invitation.BrainID)
	if err != nil {
		return input_port.InvitationPreview{}, fmt.Errorf("find invitation brain: %w", err)
	}
	inviter, err := u.users.FindUserByID(ctx, invitation.InvitedBy)
	if err != nil {
		return input_port.InvitationPreview{}, fmt.Errorf("find inviter: %w", err)
	}
	return input_port.InvitationPreview{BrainName: brain.Name, InvitedByName: inviter.Name}, nil
}

func (u *accessUseCase) AcceptInvitation(ctx context.Context, rawToken string, user entity.User) (input_port.AcceptedInvitation, error) {
	var accepted entity.Invitation
	now := u.clock.Now()
	err := u.transactions.WithinAccessTransaction(ctx, func(repositories output_port.AccessRepositories) error {
		invitation, err := repositories.FindInvitationByToken(ctx, rawToken)
		if errors.Is(err, output_port.ErrNotFound) {
			return input_port.ErrInvitationNotFound
		}
		if err != nil {
			return err
		}
		if invitation.State == entity.InvitationStateAccepted {
			return input_port.ErrInvitationAccepted
		}
		if !invitation.IsAcceptable(now) {
			return input_port.ErrInvitationNotFound
		}
		if invitation.Email != nil && *invitation.Email != user.Email {
			return input_port.ErrInvitationEmail
		}
		if _, err := repositories.FindMembership(ctx, invitation.BrainID, user.ID); err == nil {
			return input_port.ErrMembershipExists
		} else if !errors.Is(err, output_port.ErrNotFound) {
			return err
		}
		invitedBy := invitation.InvitedBy
		membership := entity.Membership{
			ID: u.ids.New(), BrainID: invitation.BrainID, UserID: user.ID, Role: invitation.Role,
			InvitedBy: &invitedBy, CreatedAt: now, UpdatedAt: now,
		}
		if err := repositories.CreateMembership(ctx, membership); errors.Is(err, output_port.ErrConflict) {
			return input_port.ErrMembershipExists
		} else if err != nil {
			return err
		}
		if err := repositories.AcceptInvitation(ctx, invitation.ID, user.ID, now); errors.Is(err, output_port.ErrConflict) {
			return input_port.ErrInvitationAccepted
		} else if err != nil {
			return err
		}
		accepted = invitation
		return nil
	})
	if err != nil {
		return input_port.AcceptedInvitation{}, err
	}
	brain, err := u.brains.FindBrainByID(ctx, accepted.BrainID)
	if err != nil {
		return input_port.AcceptedInvitation{}, fmt.Errorf("find accepted brain: %w", err)
	}
	membership, err := u.memberships.FindActiveMembership(ctx, accepted.BrainID, user.ID)
	if err != nil {
		return input_port.AcceptedInvitation{}, fmt.Errorf("find accepted membership: %w", err)
	}
	return input_port.AcceptedInvitation{Membership: membership, SourceID: brain.SourceID, BrainName: brain.Name}, nil
}

func (u *accessUseCase) IssueClient(ctx context.Context, sourceID entity.SourceID, userID, label string) (entity.IssuedClient, error) {
	if label != claudeWebLabel {
		return entity.IssuedClient{}, input_port.ErrUnsupportedClient
	}
	brain, membership, err := u.brainAndMembership(ctx, sourceID, userID)
	if err != nil {
		return entity.IssuedClient{}, err
	}
	client := entity.IssuedClient{
		ID: u.ids.New(), UserID: userID, BrainID: brain.ID, Label: label,
		ReadSourceIDs: []entity.SourceID{sourceID}, Scopes: membership.Role.GBrainScopes(),
		State: entity.ClientStateIssuing, IssuedAt: u.clock.Now(),
	}
	if membership.Role != entity.RoleReader {
		writeSourceID := sourceID
		client.WriteSourceID = &writeSourceID
	}
	if err := u.clients.CreateIssuedClient(ctx, client); err != nil {
		return entity.IssuedClient{}, fmt.Errorf("create issuing client: %w", err)
	}
	name := userID + "-" + label + "-" + client.ID
	var source *string
	if client.WriteSourceID != nil {
		value := client.WriteSourceID.String()
		source = &value
	}
	registered, err := u.admin.RegisterClient(ctx, output_port.RegisterGBrainClientInput{
		Name: name, Scopes: client.Scopes, Source: source,
		FederatedRead: []string{sourceID.String()},
		GrantTypes:    []string{"authorization_code", "refresh_token"},
		RedirectURIs:  []string{claudeWebRedirectURI}, TokenEndpointAuthMethod: "none",
	})
	if err != nil {
		orphan, markErr := u.clients.MarkIssuedClientOrphan(ctx, client.ID, nil, failureReason(err))
		if markErr != nil {
			return client, fmt.Errorf("%w: register: %v; mark orphan: %v", input_port.ErrGBrainAdmin, err, markErr)
		}
		return orphan, fmt.Errorf("%w: %v", input_port.ErrGBrainAdmin, err)
	}
	gbrainClientID := registered.ID
	if _, err := u.memberships.FindActiveMembership(ctx, brain.ID, userID); err != nil {
		return u.revokeNewClientAfterMembershipLoss(ctx, client, gbrainClientID, err)
	}
	active, err := u.clients.ActivateIssuedClient(ctx, client.ID, gbrainClientID, u.clock.Now())
	if err == nil {
		return active, nil
	}
	orphan, markErr := u.clients.MarkIssuedClientOrphan(ctx, client.ID, &gbrainClientID, failureReason(err))
	if markErr != nil {
		return client, fmt.Errorf("save issued client: %v; mark orphan: %w", err, markErr)
	}
	return orphan, fmt.Errorf("save issued client: %w", err)
}

func (u *accessUseCase) ListClients(ctx context.Context, sourceID entity.SourceID, userID string) ([]entity.IssuedClient, error) {
	brain, _, err := u.brainAndMembership(ctx, sourceID, userID)
	if err != nil {
		return nil, err
	}
	clients, err := u.clients.ListIssuedClientsByUserBrain(ctx, userID, brain.ID)
	if err != nil {
		return nil, fmt.Errorf("list issued clients: %w", err)
	}
	return clients, nil
}

func (u *accessUseCase) RevokeClient(ctx context.Context, clientID, userID string) (entity.IssuedClient, error) {
	client, err := u.clients.FindIssuedClientByID(ctx, clientID)
	if errors.Is(err, output_port.ErrNotFound) || (err == nil && client.UserID != userID) {
		return entity.IssuedClient{}, input_port.ErrIssuedClientNotFound
	}
	if err != nil {
		return entity.IssuedClient{}, fmt.Errorf("find issued client: %w", err)
	}
	if client.State == entity.ClientStateRevoked {
		return client, nil
	}
	if client.GBrainClientID == nil {
		return client, fmt.Errorf("%w: orphan client has no GBrain client ID", input_port.ErrGBrainAdmin)
	}
	if err := u.admin.RevokeClient(ctx, *client.GBrainClientID); err != nil {
		orphan, markErr := u.clients.MarkIssuedClientOrphan(ctx, client.ID, client.GBrainClientID, failureReason(err))
		if markErr != nil {
			return client, fmt.Errorf("%w: revoke: %v; mark orphan: %v", input_port.ErrGBrainAdmin, err, markErr)
		}
		return orphan, fmt.Errorf("%w: %v", input_port.ErrGBrainAdmin, err)
	}
	revoked, err := u.clients.MarkIssuedClientRevoked(ctx, client.ID, u.clock.Now())
	if err != nil {
		return client, fmt.Errorf("mark client revoked: %w", err)
	}
	return revoked, nil
}

func (u *accessUseCase) RevokeMembership(ctx context.Context, sourceID entity.SourceID, actorID, userID string) error {
	brain, actorMembership, err := u.brainAndMembership(ctx, sourceID, actorID)
	if err != nil {
		return err
	}
	if actorMembership.Role != entity.RoleOwner {
		return input_port.ErrForbidden
	}
	if actorID == userID {
		return input_port.ErrCannotRevokeSelf
	}
	target, err := u.memberships.FindActiveMembership(ctx, brain.ID, userID)
	if errors.Is(err, output_port.ErrNotFound) {
		return input_port.ErrMembershipNotFound
	}
	if err != nil {
		return fmt.Errorf("find target membership: %w", err)
	}
	if err := u.memberships.RevokeMembership(ctx, target.ID, u.clock.Now()); err != nil {
		return fmt.Errorf("revoke membership: %w", err)
	}
	return u.revokeAllBrainClients(ctx, userID, brain.ID)
}

func (u *accessUseCase) ArchiveBrain(ctx context.Context, sourceID entity.SourceID, actorID string) error {
	brain, membership, err := u.brainAndMembership(ctx, sourceID, actorID)
	if err != nil {
		return err
	}
	if membership.Role != entity.RoleOwner {
		return input_port.ErrForbidden
	}
	clients, err := u.clients.ListRevocableIssuedClientsByBrain(ctx, brain.ID)
	if err != nil {
		return fmt.Errorf("list clients for brain archive: %w", err)
	}
	if err := u.revokeClients(ctx, clients); err != nil {
		return err
	}
	if err := u.transactions.WithinAccessTransaction(ctx, func(repositories output_port.AccessRepositories) error {
		if err := repositories.RevokePendingInvitationsByBrain(ctx, brain.ID); err != nil {
			return err
		}
		return repositories.ArchiveBrain(ctx, brain.ID, u.clock.Now())
	}); err != nil {
		return fmt.Errorf("archive brain: %w", err)
	}
	return nil
}

func (u *accessUseCase) brainAndMembership(ctx context.Context, sourceID entity.SourceID, userID string) (entity.Brain, entity.Membership, error) {
	brain, err := u.brains.FindBrainBySourceID(ctx, sourceID)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.Brain{}, entity.Membership{}, input_port.ErrBrainNotFound
	}
	if err != nil {
		return entity.Brain{}, entity.Membership{}, fmt.Errorf("find brain: %w", err)
	}
	membership, err := u.memberships.FindActiveMembership(ctx, brain.ID, userID)
	if errors.Is(err, output_port.ErrNotFound) {
		return entity.Brain{}, entity.Membership{}, input_port.ErrForbidden
	}
	if err != nil {
		return entity.Brain{}, entity.Membership{}, fmt.Errorf("find membership: %w", err)
	}
	return brain, membership, nil
}

func (u *accessUseCase) revokeAllBrainClients(ctx context.Context, userID, brainID string) error {
	clients, err := u.clients.ListRevocableIssuedClientsByUserBrain(ctx, userID, brainID)
	if err != nil {
		return fmt.Errorf("list clients for membership revocation: %w", err)
	}
	return u.revokeClients(ctx, clients)
}

func (u *accessUseCase) revokeClients(ctx context.Context, clients []entity.IssuedClient) error {
	var failures []error
	for _, client := range clients {
		if err := u.admin.RevokeClient(ctx, *client.GBrainClientID); err != nil {
			failures = append(failures, err)
			if _, markErr := u.clients.MarkIssuedClientOrphan(ctx, client.ID, client.GBrainClientID, failureReason(err)); markErr != nil {
				failures = append(failures, markErr)
			}
			continue
		}
		if _, err := u.clients.MarkIssuedClientRevoked(ctx, client.ID, u.clock.Now()); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%w: %v", input_port.ErrGBrainAdmin, errors.Join(failures...))
	}
	return nil
}

func (u *accessUseCase) revokeNewClientAfterMembershipLoss(ctx context.Context, client entity.IssuedClient, gbrainClientID string, membershipErr error) (entity.IssuedClient, error) {
	if err := u.admin.RevokeClient(ctx, gbrainClientID); err != nil {
		reason := failureReason(fmt.Errorf("membership no longer active: %v; revoke: %w", membershipErr, err))
		orphan, markErr := u.clients.MarkIssuedClientOrphan(ctx, client.ID, &gbrainClientID, reason)
		if markErr != nil {
			return client, fmt.Errorf("%w: %v", input_port.ErrGBrainAdmin, markErr)
		}
		return orphan, fmt.Errorf("%w: %v", input_port.ErrGBrainAdmin, err)
	}
	revoked, err := u.clients.MarkIssuedClientRevoked(ctx, client.ID, u.clock.Now())
	if err != nil {
		return client, fmt.Errorf("mark client revoked after membership loss: %w", err)
	}
	return revoked, input_port.ErrForbidden
}

func failureReason(err error) string {
	reason := err.Error()
	if len(reason) > maxStateReasonBytes {
		return reason[len(reason)-maxStateReasonBytes:]
	}
	return reason
}
