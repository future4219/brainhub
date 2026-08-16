package repository

import (
	"context"
	"errors"
	"time"

	"brainhub/adapter/database/model"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"

	"github.com/jackc/pgx/v5"
)

const brainColumns = `
	id, source_id, name, description, visibility, owner_id, state,
	state_reason, created_at, updated_at, archived_at`

func (s *Store) CreateBrain(ctx context.Context, brain entity.Brain) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO brains (
			id, source_id, name, description, visibility, owner_id, state,
			state_reason, created_at, updated_at, archived_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		brain.ID, brain.SourceID.String(), brain.Name, brain.Description, string(brain.Visibility),
		brain.OwnerID, string(brain.State), brain.StateReason, brain.CreatedAt, brain.UpdatedAt, brain.ArchivedAt,
	)
	return mapError(err)
}

func (s *Store) ListBrains(ctx context.Context) ([]entity.Brain, error) {
	rows, err := s.queries.Query(ctx, `SELECT `+brainColumns+` FROM brains ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var brains []entity.Brain
	for rows.Next() {
		brain, err := scanBrain(rows)
		if err != nil {
			return nil, err
		}
		brains = append(brains, brain)
	}
	return brains, rows.Err()
}

func (s *Store) FindBrainBySourceID(ctx context.Context, sourceID entity.SourceID) (entity.Brain, error) {
	brain, err := scanBrain(s.queries.QueryRow(ctx, `SELECT `+brainColumns+` FROM brains WHERE source_id = $1`, sourceID.String()))
	if err != nil {
		return entity.Brain{}, mapError(err)
	}
	return brain, nil
}

func (s *Store) FindBrainByID(ctx context.Context, id string) (entity.Brain, error) {
	brain, err := scanBrain(s.queries.QueryRow(ctx, `SELECT `+brainColumns+` FROM brains WHERE id = $1`, id))
	if err != nil {
		return entity.Brain{}, mapError(err)
	}
	return brain, nil
}

func (s *Store) TransitionBrain(ctx context.Context, id string, state entconst.BrainState, reason string, now time.Time) (entity.Brain, error) {
	brain, err := scanBrain(s.queries.QueryRow(ctx, `
		UPDATE brains
		SET state = $1, state_reason = $2, updated_at = $3
		WHERE id = $4 AND state = 'provisioning'
		RETURNING `+brainColumns,
		string(state), reason, now, id,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.Brain{}, output_port.ErrConflict
	}
	if err != nil {
		return entity.Brain{}, err
	}
	return brain, nil
}

func (s *Store) CreateMembership(ctx context.Context, membership entity.Membership) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO memberships (
			id, brain_id, user_id, role, invited_by, created_at, updated_at, revoked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		membership.ID, membership.BrainID, membership.UserID, string(membership.Role), membership.InvitedBy,
		membership.CreatedAt, membership.UpdatedAt, membership.RevokedAt,
	)
	return mapError(err)
}

func (s *Store) ListActiveMembershipsByUser(ctx context.Context, userID string) ([]entity.Membership, error) {
	rows, err := s.queries.Query(ctx, `
		SELECT id, brain_id, user_id, role, invited_by, created_at, updated_at, revoked_at
		FROM memberships
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberships []entity.Membership
	for rows.Next() {
		var membership model.Membership
		if err := rows.Scan(
			&membership.ID, &membership.BrainID, &membership.UserID, &membership.Role,
			&membership.InvitedBy, &membership.CreatedAt, &membership.UpdatedAt, &membership.RevokedAt,
		); err != nil {
			return nil, err
		}
		memberships = append(memberships, membership.Entity())
	}
	return memberships, rows.Err()
}

func (s *Store) WithinBrainTransaction(ctx context.Context, fn func(output_port.BrainRepositories) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	transactionStore := &Store{
		pool:    s.pool,
		queries: tx,
		clock:   s.clock,
		ids:     s.ids,
		random:  s.random,
	}
	if err := fn(transactionStore); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type brainScanner interface {
	Scan(...any) error
}

func scanBrain(row brainScanner) (entity.Brain, error) {
	var brain model.Brain
	if err := row.Scan(
		&brain.ID, &brain.SourceID, &brain.Name, &brain.Description, &brain.Visibility,
		&brain.OwnerID, &brain.State, &brain.StateReason, &brain.CreatedAt, &brain.UpdatedAt, &brain.ArchivedAt,
	); err != nil {
		return entity.Brain{}, err
	}
	return brain.Entity(), nil
}
