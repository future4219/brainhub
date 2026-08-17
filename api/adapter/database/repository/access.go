package repository

import (
	"context"
	"encoding/base64"
	"io"
	"time"

	"brainhub/adapter/database/model"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

const invitationColumns = `
	id, brain_id, email, role, token_hash, invited_by, state,
	expires_at, created_at, accepted_at, accepted_by`

const issuedClientColumns = `
	id, user_id, brain_id, gbrain_client_id, label, write_source_id,
	read_source_ids, scopes, state, state_reason, issued_at,
	last_verified_at, revoked_at`

func (s *Store) CreateInvitation(ctx context.Context, invitation entity.Invitation) (entity.Invitation, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(s.random, tokenBytes); err != nil {
		return entity.Invitation{}, "", err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	invitation.TokenHash = hashToken(rawToken)
	_, err := s.queries.Exec(ctx, `
		INSERT INTO invitations (
			id, brain_id, email, role, token_hash, invited_by, state,
			expires_at, created_at, accepted_at, accepted_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		invitation.ID, invitation.BrainID, invitation.Email, string(invitation.Role), invitation.TokenHash,
		invitation.InvitedBy, string(invitation.State), invitation.ExpiresAt, invitation.CreatedAt,
		invitation.AcceptedAt, invitation.AcceptedBy,
	)
	if err != nil {
		return entity.Invitation{}, "", mapError(err)
	}
	return invitation, rawToken, nil
}

func (s *Store) FindInvitationByID(ctx context.Context, id string) (entity.Invitation, error) {
	invitation, err := scanInvitation(s.queries.QueryRow(ctx, `SELECT `+invitationColumns+` FROM invitations WHERE id = $1`, id))
	if err != nil {
		return entity.Invitation{}, mapError(err)
	}
	return invitation, nil
}

func (s *Store) FindInvitationByToken(ctx context.Context, rawToken string) (entity.Invitation, error) {
	invitation, err := scanInvitation(s.queries.QueryRow(ctx, `SELECT `+invitationColumns+` FROM invitations WHERE token_hash = $1`, hashToken(rawToken)))
	if err != nil {
		return entity.Invitation{}, mapError(err)
	}
	return invitation, nil
}

func (s *Store) ListInvitationsByBrain(ctx context.Context, brainID string) ([]entity.Invitation, error) {
	rows, err := s.queries.Query(ctx, `SELECT `+invitationColumns+` FROM invitations WHERE brain_id = $1 ORDER BY created_at DESC, id`, brainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []entity.Invitation
	for rows.Next() {
		invitation, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, invitation)
	}
	return invitations, rows.Err()
}

func (s *Store) AcceptInvitation(ctx context.Context, id, userID string, now time.Time) error {
	tag, err := s.queries.Exec(ctx, `
		UPDATE invitations
		SET state = 'accepted', accepted_at = $1, accepted_by = $2
		WHERE id = $3 AND state = 'pending' AND expires_at > $1`, now, userID, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return output_port.ErrConflict
	}
	return nil
}

func (s *Store) RevokeInvitation(ctx context.Context, id string) error {
	tag, err := s.queries.Exec(ctx, `UPDATE invitations SET state = 'revoked' WHERE id = $1 AND state = 'pending'`, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return output_port.ErrConflict
	}
	return nil
}

func (s *Store) FindMembership(ctx context.Context, brainID, userID string) (entity.Membership, error) {
	return s.findMembership(ctx, brainID, userID, false)
}

func (s *Store) FindActiveMembership(ctx context.Context, brainID, userID string) (entity.Membership, error) {
	return s.findMembership(ctx, brainID, userID, true)
}

func (s *Store) findMembership(ctx context.Context, brainID, userID string, activeOnly bool) (entity.Membership, error) {
	query := `
		SELECT id, brain_id, user_id, role, invited_by, created_at, updated_at, revoked_at
		FROM memberships WHERE brain_id = $1 AND user_id = $2`
	if activeOnly {
		query += ` AND revoked_at IS NULL`
	}
	var membership model.Membership
	err := s.queries.QueryRow(ctx, query, brainID, userID).Scan(
		&membership.ID, &membership.BrainID, &membership.UserID, &membership.Role,
		&membership.InvitedBy, &membership.CreatedAt, &membership.UpdatedAt, &membership.RevokedAt,
	)
	if err != nil {
		return entity.Membership{}, mapError(err)
	}
	return membership.Entity(), nil
}

func (s *Store) RevokeMembership(ctx context.Context, id string, now time.Time) error {
	tag, err := s.queries.Exec(ctx, `
		UPDATE memberships SET revoked_at = $1, updated_at = $1
		WHERE id = $2 AND revoked_at IS NULL`, now, id)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return output_port.ErrConflict
	}
	return nil
}

func (s *Store) CreateIssuedClient(ctx context.Context, client entity.IssuedClient) error {
	var writeSourceID *string
	if client.WriteSourceID != nil {
		value := client.WriteSourceID.String()
		writeSourceID = &value
	}
	readSourceIDs := make([]string, len(client.ReadSourceIDs))
	for i, sourceID := range client.ReadSourceIDs {
		readSourceIDs[i] = sourceID.String()
	}
	_, err := s.queries.Exec(ctx, `
		INSERT INTO issued_clients (
			id, user_id, brain_id, gbrain_client_id, label, write_source_id,
			read_source_ids, scopes, state, state_reason, issued_at,
			last_verified_at, revoked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		client.ID, client.UserID, client.BrainID, client.GBrainClientID, client.Label, writeSourceID,
		readSourceIDs, client.Scopes, string(client.State), client.StateReason, client.IssuedAt,
		client.LastVerifiedAt, client.RevokedAt,
	)
	return mapError(err)
}

func (s *Store) FindIssuedClientByID(ctx context.Context, id string) (entity.IssuedClient, error) {
	client, err := scanIssuedClient(s.queries.QueryRow(ctx, `SELECT `+issuedClientColumns+` FROM issued_clients WHERE id = $1`, id))
	if err != nil {
		return entity.IssuedClient{}, mapError(err)
	}
	return client, nil
}

func (s *Store) ListIssuedClientsByUserBrain(ctx context.Context, userID, brainID string) ([]entity.IssuedClient, error) {
	return s.listIssuedClients(ctx, `user_id = $1 AND brain_id = $2`, userID, brainID)
}

func (s *Store) ListRevocableIssuedClientsByUserBrain(ctx context.Context, userID, brainID string) ([]entity.IssuedClient, error) {
	return s.listIssuedClients(ctx, `user_id = $1 AND brain_id = $2 AND gbrain_client_id IS NOT NULL AND state IN ('active', 'orphan')`, userID, brainID)
}

func (s *Store) listIssuedClients(ctx context.Context, where string, args ...any) ([]entity.IssuedClient, error) {
	rows, err := s.queries.Query(ctx, `SELECT `+issuedClientColumns+` FROM issued_clients WHERE `+where+` ORDER BY issued_at DESC, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var clients []entity.IssuedClient
	for rows.Next() {
		client, err := scanIssuedClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func (s *Store) ActivateIssuedClient(ctx context.Context, id, gbrainClientID string, now time.Time) (entity.IssuedClient, error) {
	return s.updateIssuedClient(ctx, `
		UPDATE issued_clients
		SET gbrain_client_id = $1, state = 'active', state_reason = '', last_verified_at = $2
		WHERE id = $3 AND state = 'issuing'
		RETURNING `+issuedClientColumns, gbrainClientID, now, id)
}

func (s *Store) MarkIssuedClientOrphan(ctx context.Context, id string, gbrainClientID *string, reason string) (entity.IssuedClient, error) {
	return s.updateIssuedClient(ctx, `
		UPDATE issued_clients
		SET gbrain_client_id = COALESCE($1, gbrain_client_id), state = 'orphan', state_reason = $2
		WHERE id = $3 AND state <> 'revoked'
		RETURNING `+issuedClientColumns, gbrainClientID, reason, id)
}

func (s *Store) MarkIssuedClientRevoked(ctx context.Context, id string, now time.Time) (entity.IssuedClient, error) {
	return s.updateIssuedClient(ctx, `
		UPDATE issued_clients SET state = 'revoked', state_reason = '', revoked_at = $1
		WHERE id = $2 AND state <> 'revoked'
		RETURNING `+issuedClientColumns, now, id)
}

func (s *Store) updateIssuedClient(ctx context.Context, query string, args ...any) (entity.IssuedClient, error) {
	client, err := scanIssuedClient(s.queries.QueryRow(ctx, query, args...))
	if err != nil {
		return entity.IssuedClient{}, mapError(err)
	}
	return client, nil
}

func (s *Store) WithinAccessTransaction(ctx context.Context, fn func(output_port.AccessRepositories) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	transactionStore := &Store{pool: s.pool, queries: tx, clock: s.clock, ids: s.ids, random: s.random}
	if err := fn(transactionStore); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type accessScanner interface {
	Scan(...any) error
}

func scanInvitation(row accessScanner) (entity.Invitation, error) {
	var invitation model.Invitation
	if err := row.Scan(
		&invitation.ID, &invitation.BrainID, &invitation.Email, &invitation.Role, &invitation.TokenHash,
		&invitation.InvitedBy, &invitation.State, &invitation.ExpiresAt, &invitation.CreatedAt,
		&invitation.AcceptedAt, &invitation.AcceptedBy,
	); err != nil {
		return entity.Invitation{}, err
	}
	return invitation.Entity(), nil
}

func scanIssuedClient(row accessScanner) (entity.IssuedClient, error) {
	var client model.IssuedClient
	if err := row.Scan(
		&client.ID, &client.UserID, &client.BrainID, &client.GBrainClientID, &client.Label,
		&client.WriteSourceID, &client.ReadSourceIDs, &client.Scopes, &client.State,
		&client.StateReason, &client.IssuedAt, &client.LastVerifiedAt, &client.RevokedAt,
	); err != nil {
		return entity.IssuedClient{}, err
	}
	return client.Entity(), nil
}
