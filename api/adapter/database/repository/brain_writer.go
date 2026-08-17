package repository

import (
	"context"
	"time"

	"brainhub/adapter/database/model"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

const brainWriterClientColumns = `
	brain_id, write_source_id, gbrain_client_id, client_secret_ciphertext,
	state, state_reason, issued_at, last_verified_at`

func (s *Store) CreateBrainWriterClient(ctx context.Context, client entity.BrainWriterClient) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO brain_writer_clients (
			brain_id, write_source_id, gbrain_client_id, client_secret_ciphertext,
			state, state_reason, issued_at, last_verified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		client.BrainID, client.WriteSourceID.String(), client.GBrainClientID,
		client.ClientSecretCiphertext, string(client.State), client.StateReason,
		client.IssuedAt, client.LastVerifiedAt,
	)
	return mapError(err)
}

func (s *Store) FindBrainWriterClient(ctx context.Context, brainID string) (entity.BrainWriterClient, error) {
	client, err := scanBrainWriterClient(s.queries.QueryRow(ctx,
		`SELECT `+brainWriterClientColumns+` FROM brain_writer_clients WHERE brain_id = $1`, brainID))
	if err != nil {
		return entity.BrainWriterClient{}, mapError(err)
	}
	return client, nil
}

func (s *Store) ListIssuingBrainWriterClients(ctx context.Context) ([]entity.BrainWriterClient, error) {
	rows, err := s.queries.Query(ctx, `SELECT `+brainWriterClientColumns+`
		FROM brain_writer_clients WHERE state = 'issuing' ORDER BY brain_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	clients := make([]entity.BrainWriterClient, 0)
	for rows.Next() {
		client, err := scanBrainWriterClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func (s *Store) ActivateBrainWriterClient(ctx context.Context, brainID, gbrainClientID string, ciphertext []byte, now time.Time) (entity.BrainWriterClient, error) {
	client, err := scanBrainWriterClient(s.queries.QueryRow(ctx, `
		UPDATE brain_writer_clients
		SET gbrain_client_id = $1, client_secret_ciphertext = $2, state = 'active',
			state_reason = '', issued_at = $3, last_verified_at = $3
		WHERE brain_id = $4 AND state = 'issuing'
		RETURNING `+brainWriterClientColumns, gbrainClientID, ciphertext, now, brainID))
	if err != nil {
		return entity.BrainWriterClient{}, mapError(err)
	}
	return client, nil
}

func (s *Store) MarkBrainWriterClientOrphan(ctx context.Context, brainID string, gbrainClientID *string, reason string) (entity.BrainWriterClient, error) {
	client, err := scanBrainWriterClient(s.queries.QueryRow(ctx, `
		UPDATE brain_writer_clients
		SET gbrain_client_id = $1,
			client_secret_ciphertext = NULL, state = 'orphan', state_reason = $2
		WHERE brain_id = $3
		RETURNING `+brainWriterClientColumns, gbrainClientID, reason, brainID))
	if err != nil {
		return entity.BrainWriterClient{}, mapError(err)
	}
	return client, nil
}

func (s *Store) ResetBrainWriterClient(ctx context.Context, brainID string) error {
	tag, err := s.queries.Exec(ctx, `
		UPDATE brain_writer_clients
		SET gbrain_client_id = NULL, client_secret_ciphertext = NULL,
			state = 'issuing', state_reason = '', issued_at = NULL, last_verified_at = NULL
		WHERE brain_id = $1`, brainID)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return output_port.ErrNotFound
	}
	return nil
}

type brainWriterScanner interface {
	Scan(...any) error
}

func scanBrainWriterClient(row brainWriterScanner) (entity.BrainWriterClient, error) {
	var client model.BrainWriterClient
	if err := row.Scan(
		&client.BrainID, &client.WriteSourceID, &client.GBrainClientID,
		&client.ClientSecretCiphertext, &client.State, &client.StateReason,
		&client.IssuedAt, &client.LastVerifiedAt,
	); err != nil {
		return entity.BrainWriterClient{}, err
	}
	return client.Entity(), nil
}
