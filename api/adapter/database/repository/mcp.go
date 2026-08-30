package repository

import (
	"context"
	"time"

	"brainhub/adapter/database/model"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"

	"github.com/jackc/pgx/v5"
)

const readerClientColumns = `
	id, user_id, gbrain_client_id, client_secret_ciphertext, federated_read,
	state, state_reason, issued_at, last_verified_at`

func (s *Store) CreateMCPClient(ctx context.Context, client entity.MCPClient) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO mcp_clients (id, user_id, name, redirect_uris, created_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		client.ID, client.UserID, client.Name, client.RedirectURIs, client.CreatedAt, client.RevokedAt,
	)
	return mapError(err)
}

func (s *Store) FindMCPClientByID(ctx context.Context, id string) (entity.MCPClient, error) {
	return scanMCPClient(s.queries.QueryRow(ctx, `
		SELECT c.id, c.user_id, c.name, c.redirect_uris, c.created_at, c.revoked_at
		FROM mcp_clients c
		JOIN users u ON u.id = c.user_id AND u.state = 'active'
		WHERE c.id = $1`, id))
}

func (s *Store) FindActiveMCPClientByUserName(ctx context.Context, userID, name string) (entity.MCPClient, error) {
	return scanMCPClient(s.queries.QueryRow(ctx, `
		SELECT c.id, c.user_id, c.name, c.redirect_uris, c.created_at, c.revoked_at
		FROM mcp_clients c
		JOIN users u ON u.id = c.user_id AND u.state = 'active'
		WHERE c.user_id = $1 AND c.name = $2 AND c.revoked_at IS NULL`, userID, name))
}

func (s *Store) CreateMCPAuthorizationCode(ctx context.Context, code entity.MCPAuthorizationCode) (string, error) {
	raw, err := randomToken(s.random)
	if err != nil {
		return "", err
	}
	_, err = s.queries.Exec(ctx, `
		INSERT INTO mcp_authorization_codes (
			id, code_hash, client_id, user_id, redirect_uri, code_challenge,
			resource, expires_at, created_at, consumed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		code.ID, hashToken(raw), code.ClientID, code.UserID, code.RedirectURI,
		code.CodeChallenge, code.Resource, code.ExpiresAt, code.CreatedAt, code.ConsumedAt,
	)
	return raw, mapError(err)
}

func (s *Store) ConsumeMCPAuthorizationCode(ctx context.Context, raw, clientID, redirectURI, challenge string, now time.Time) (entity.MCPAuthorizationCode, error) {
	var code model.MCPAuthorizationCode
	err := s.queries.QueryRow(ctx, `
		UPDATE mcp_authorization_codes
		SET consumed_at = $1
		WHERE code_hash = $2 AND client_id = $3 AND redirect_uri = $4
		  AND code_challenge = $5 AND consumed_at IS NULL AND expires_at > $1
		RETURNING id, client_id, user_id, redirect_uri, code_challenge,
		          resource, expires_at, created_at, consumed_at`,
		now, hashToken(raw), clientID, redirectURI, challenge,
	).Scan(
		&code.ID, &code.ClientID, &code.UserID, &code.RedirectURI, &code.CodeChallenge,
		&code.Resource, &code.ExpiresAt, &code.CreatedAt, &code.ConsumedAt,
	)
	if err != nil {
		return entity.MCPAuthorizationCode{}, mapError(err)
	}
	return authorizationCodeEntity(code), nil
}

func (s *Store) CreateMCPTokenPair(ctx context.Context, clientID, userID string, accessExpiry, refreshExpiry time.Time) (string, string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	access, refresh, err := s.insertMCPTokenPair(ctx, tx, clientID, userID, accessExpiry, refreshExpiry)
	if err != nil {
		return "", "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *Store) RotateMCPRefreshToken(ctx context.Context, raw, clientID string, now, accessExpiry, refreshExpiry time.Time) (entity.MCPToken, string, string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entity.MCPToken{}, "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var token model.MCPToken
	err = tx.QueryRow(ctx, `
		UPDATE mcp_tokens SET revoked_at = $1
		WHERE token_hash = $2 AND client_id = $3 AND token_type = 'refresh'
		  AND revoked_at IS NULL AND expires_at > $1
		RETURNING id, token_type, client_id, user_id, expires_at, created_at, revoked_at`,
		now, hashToken(raw), clientID,
	).Scan(&token.ID, &token.Type, &token.ClientID, &token.UserID, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt)
	if err != nil {
		return entity.MCPToken{}, "", "", mapError(err)
	}
	access, refresh, err := s.insertMCPTokenPair(ctx, tx, clientID, token.UserID, accessExpiry, refreshExpiry)
	if err != nil {
		return entity.MCPToken{}, "", "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return entity.MCPToken{}, "", "", err
	}
	return tokenEntity(token), access, refresh, nil
}

func (s *Store) insertMCPTokenPair(ctx context.Context, tx pgx.Tx, clientID, userID string, accessExpiry, refreshExpiry time.Time) (string, string, error) {
	access, err := randomToken(s.random)
	if err != nil {
		return "", "", err
	}
	refresh, err := randomToken(s.random)
	if err != nil {
		return "", "", err
	}
	now := s.clock.Now()
	tag, err := tx.Exec(ctx, `
		WITH authorized AS (
			SELECT c.id AS client_id, c.user_id
			FROM mcp_clients c
			JOIN users u ON u.id = c.user_id AND u.state = 'active'
			WHERE c.id = $3 AND c.user_id = $4 AND c.revoked_at IS NULL
		)
		INSERT INTO mcp_tokens (id, token_hash, token_type, client_id, user_id, expires_at, created_at)
		SELECT $1, $2, 'access', client_id, user_id, $5::timestamptz, $6::timestamptz FROM authorized
		UNION ALL
		SELECT $7, $8, 'refresh', client_id, user_id, $9::timestamptz, $6::timestamptz FROM authorized`,
		s.ids.New(), hashToken(access), clientID, userID, accessExpiry, now,
		s.ids.New(), hashToken(refresh), refreshExpiry,
	)
	if err != nil {
		return "", "", mapError(err)
	}
	if tag.RowsAffected() != 2 {
		return "", "", output_port.ErrNotFound
	}
	return access, refresh, nil
}

func (s *Store) VerifyMCPAccessToken(ctx context.Context, raw string, now time.Time) (entity.MCPToken, error) {
	var token model.MCPToken
	err := s.queries.QueryRow(ctx, `
		SELECT t.id, t.token_type, t.client_id, t.user_id, t.expires_at, t.created_at, t.revoked_at
		FROM mcp_tokens t
		JOIN mcp_clients c ON c.id = t.client_id AND c.user_id = t.user_id AND c.revoked_at IS NULL
		JOIN users u ON u.id = t.user_id AND u.state = 'active'
		WHERE t.token_hash = $1 AND t.token_type = 'access'
		  AND t.revoked_at IS NULL AND t.expires_at > $2`, hashToken(raw), now,
	).Scan(&token.ID, &token.Type, &token.ClientID, &token.UserID, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt)
	if err != nil {
		return entity.MCPToken{}, mapError(err)
	}
	return tokenEntity(token), nil
}

func (s *Store) RevokeMCPToken(ctx context.Context, raw, clientID string, now time.Time) error {
	_, err := s.queries.Exec(ctx, `
		WITH target AS (
			SELECT user_id FROM mcp_tokens WHERE token_hash = $2 AND client_id = $3
		)
		UPDATE mcp_tokens SET revoked_at = $1
		WHERE client_id = $3 AND user_id IN (SELECT user_id FROM target)
		  AND revoked_at IS NULL`,
		now, hashToken(raw), clientID,
	)
	return mapError(err)
}

func (s *Store) ListMCPVisibleBrains(ctx context.Context, userID string) ([]entity.MCPVisibleBrain, error) {
	rows, err := s.queries.Query(ctx, `
		SELECT b.source_id, b.name, b.state, COALESCE(m.role, 'public')
		FROM brains b
		LEFT JOIN memberships m
		  ON m.brain_id = b.id AND m.user_id = $1 AND m.revoked_at IS NULL
		WHERE b.archived_at IS NULL AND (
		  (m.id IS NOT NULL AND b.state IN ('ready', 'degraded'))
		  OR (b.visibility = 'public' AND b.state = 'ready')
		)
		ORDER BY b.created_at, b.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	visible := make([]entity.MCPVisibleBrain, 0)
	for rows.Next() {
		var brain entity.MCPVisibleBrain
		var sourceID string
		if err := rows.Scan(&sourceID, &brain.Name, &brain.State, &brain.Role); err != nil {
			return nil, err
		}
		brain.SourceID = entity.SourceID(sourceID)
		visible = append(visible, brain)
	}
	return visible, rows.Err()
}

func (s *Store) CreateReaderClient(ctx context.Context, client entity.ReaderClient) error {
	_, err := s.queries.Exec(ctx, `
		INSERT INTO brainhub_reader_clients (
			id, user_id, gbrain_client_id, client_secret_ciphertext, federated_read,
			state, state_reason, issued_at, last_verified_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		client.ID, client.UserID, client.GBrainClientID, client.ClientSecretCiphertext,
		sourceIDStrings(client.FederatedRead), string(client.State), client.StateReason,
		client.IssuedAt, client.LastVerifiedAt,
	)
	return mapError(err)
}

func (s *Store) FindReaderClientByUser(ctx context.Context, userID string) (entity.ReaderClient, error) {
	return scanReaderClient(s.queries.QueryRow(ctx,
		`SELECT `+readerClientColumns+` FROM brainhub_reader_clients WHERE user_id = $1`, userID))
}

func (s *Store) ListReaderClients(ctx context.Context) ([]entity.ReaderClient, error) {
	rows, err := s.queries.Query(ctx, `SELECT `+readerClientColumns+` FROM brainhub_reader_clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	clients := make([]entity.ReaderClient, 0)
	for rows.Next() {
		client, err := scanReaderClient(rows)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func (s *Store) ActivateReaderClient(ctx context.Context, userID, gbrainClientID string, ciphertext []byte, sources []entity.SourceID, now time.Time) (entity.ReaderClient, error) {
	return scanReaderClient(s.queries.QueryRow(ctx, `
		UPDATE brainhub_reader_clients
		SET gbrain_client_id = $1, client_secret_ciphertext = $2, federated_read = $3,
		    state = 'active', state_reason = '', issued_at = $4, last_verified_at = $4
		WHERE user_id = $5 AND state = 'issuing'
		RETURNING `+readerClientColumns,
		gbrainClientID, ciphertext, sourceIDStrings(sources), now, userID,
	))
}

func (s *Store) UpdateReaderClientScope(ctx context.Context, userID string, sources []entity.SourceID, now time.Time) (entity.ReaderClient, error) {
	return scanReaderClient(s.queries.QueryRow(ctx, `
		UPDATE brainhub_reader_clients
		SET federated_read = $1, state_reason = '', last_verified_at = $2
		WHERE user_id = $3 AND state = 'active'
		RETURNING `+readerClientColumns, sourceIDStrings(sources), now, userID))
}

func (s *Store) MarkReaderClientOrphan(ctx context.Context, userID string, gbrainClientID *string, reason string) (entity.ReaderClient, error) {
	return scanReaderClient(s.queries.QueryRow(ctx, `
		UPDATE brainhub_reader_clients
		SET gbrain_client_id = COALESCE($1, gbrain_client_id),
		    client_secret_ciphertext = NULL, state = 'orphan', state_reason = $2
		WHERE user_id = $3
		RETURNING `+readerClientColumns, gbrainClientID, reason, userID))
}

func (s *Store) ResetReaderClient(ctx context.Context, userID string) error {
	tag, err := s.queries.Exec(ctx, `
		UPDATE brainhub_reader_clients
		SET gbrain_client_id = NULL, client_secret_ciphertext = NULL,
		    federated_read = '{}', state = 'issuing', state_reason = '',
		    issued_at = NULL, last_verified_at = NULL
		WHERE user_id = $1 AND state = 'orphan'`, userID)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return output_port.ErrConflict
	}
	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanMCPClient(row rowScanner) (entity.MCPClient, error) {
	var client model.MCPClient
	if err := row.Scan(&client.ID, &client.UserID, &client.Name, &client.RedirectURIs, &client.CreatedAt, &client.RevokedAt); err != nil {
		return entity.MCPClient{}, mapError(err)
	}
	return entity.MCPClient{
		ID: client.ID, UserID: client.UserID, Name: client.Name,
		RedirectURIs: append([]string(nil), client.RedirectURIs...),
		CreatedAt:    client.CreatedAt, RevokedAt: client.RevokedAt,
	}, nil
}

func scanReaderClient(row rowScanner) (entity.ReaderClient, error) {
	var client model.ReaderClient
	if err := row.Scan(
		&client.ID, &client.UserID, &client.GBrainClientID, &client.ClientSecretCiphertext,
		&client.FederatedRead, &client.State, &client.StateReason, &client.IssuedAt, &client.LastVerifiedAt,
	); err != nil {
		return entity.ReaderClient{}, mapError(err)
	}
	sources := make([]entity.SourceID, len(client.FederatedRead))
	for i, source := range client.FederatedRead {
		sources[i] = entity.SourceID(source)
	}
	return entity.ReaderClient{
		ID: client.ID, UserID: client.UserID, GBrainClientID: client.GBrainClientID,
		ClientSecretCiphertext: append([]byte(nil), client.ClientSecretCiphertext...),
		FederatedRead:          sources, State: entity.ReaderClientState(client.State), StateReason: client.StateReason,
		IssuedAt: client.IssuedAt, LastVerifiedAt: client.LastVerifiedAt,
	}, nil
}

func sourceIDStrings(sources []entity.SourceID) []string {
	values := make([]string, len(sources))
	for i, source := range sources {
		values[i] = source.String()
	}
	return values
}

func authorizationCodeEntity(code model.MCPAuthorizationCode) entity.MCPAuthorizationCode {
	return entity.MCPAuthorizationCode{
		ID: code.ID, ClientID: code.ClientID, UserID: code.UserID,
		RedirectURI: code.RedirectURI, CodeChallenge: code.CodeChallenge, Resource: code.Resource,
		ExpiresAt: code.ExpiresAt, CreatedAt: code.CreatedAt, ConsumedAt: code.ConsumedAt,
	}
}

func tokenEntity(token model.MCPToken) entity.MCPToken {
	return entity.MCPToken{
		ID: token.ID, Type: entity.MCPTokenType(token.Type), ClientID: token.ClientID,
		UserID: token.UserID, ExpiresAt: token.ExpiresAt, CreatedAt: token.CreatedAt, RevokedAt: token.RevokedAt,
	}
}
