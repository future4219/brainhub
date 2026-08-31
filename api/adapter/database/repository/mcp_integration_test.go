package repository_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"brainhub/adapter/clock"
	"brainhub/adapter/database"
	"brainhub/adapter/database/repository"
	"brainhub/adapter/ulid"
	"brainhub/config"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/output_port"
)

func TestMCPRepositoryAuthorizationBoundaryWithPostgres(t *testing.T) {
	databaseURL, err := config.DatabaseURL()
	if err != nil {
		t.Skip("BRAINHUB_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	ids, now := ulid.Generator{}, time.Now().UTC()
	store := repository.New(pool, clock.Clock{}, ids)
	user := entity.User{ID: ids.New(), Email: ids.New() + "@example.com", Name: "MCP User", State: entconst.UserStateActive, CreatedAt: now, UpdatedAt: now}
	other := entity.User{ID: ids.New(), Email: ids.New() + "@example.com", Name: "MCP Other", State: entconst.UserStateActive, CreatedAt: now, UpdatedAt: now}
	for _, value := range []entity.User{user, other} {
		if err := store.CreateUser(ctx, value); err != nil {
			t.Fatal(err)
		}
	}

	suffix := strings.ToLower(ids.New()[20:])
	privateOwn := entity.Brain{ID: ids.New(), SourceID: entity.SourceID("mcp-own-" + suffix), Name: "Own", Visibility: entconst.VisibilityPrivate, OwnerID: user.ID, State: entconst.BrainStateReady, CreatedAt: now, UpdatedAt: now}
	publicOther := entity.Brain{ID: ids.New(), SourceID: entity.SourceID("mcp-public-" + suffix), Name: "Public", Visibility: entconst.VisibilityPublic, OwnerID: other.ID, State: entconst.BrainStateReady, CreatedAt: now, UpdatedAt: now}
	privateOther := entity.Brain{ID: ids.New(), SourceID: entity.SourceID("mcp-other-" + suffix), Name: "Other", Visibility: entconst.VisibilityPrivate, OwnerID: other.ID, State: entconst.BrainStateReady, CreatedAt: now, UpdatedAt: now}
	newOwn := entity.Brain{ID: ids.New(), SourceID: entity.SourceID("mcp-new-" + suffix), Name: "New", Visibility: entconst.VisibilityPrivate, OwnerID: user.ID, State: entconst.BrainStateReady, CreatedAt: now, UpdatedAt: now}
	brainIDs := []string{privateOwn.ID, publicOther.ID, privateOther.ID, newOwn.ID}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM brainhub_reader_clients WHERE user_id = ANY($1)", []string{user.ID, other.ID})
		_, _ = pool.Exec(ctx, "DELETE FROM mcp_tokens WHERE user_id = ANY($1)", []string{user.ID, other.ID})
		_, _ = pool.Exec(ctx, "DELETE FROM mcp_authorization_codes WHERE user_id = ANY($1)", []string{user.ID, other.ID})
		_, _ = pool.Exec(ctx, "DELETE FROM mcp_clients WHERE user_id = ANY($1)", []string{user.ID, other.ID})
		_, _ = pool.Exec(ctx, "DELETE FROM memberships WHERE brain_id = ANY($1)", brainIDs)
		_, _ = pool.Exec(ctx, "DELETE FROM brains WHERE id = ANY($1)", brainIDs)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = ANY($1)", []string{user.ID, other.ID})
	})

	for _, brain := range []entity.Brain{privateOwn, publicOther, privateOther} {
		if err := store.CreateBrain(ctx, brain); err != nil {
			t.Fatal(err)
		}
	}
	ownMembership := entity.Membership{ID: ids.New(), BrainID: privateOwn.ID, UserID: user.ID, Role: entity.RoleOwner, CreatedAt: now, UpdatedAt: now}
	if err := store.CreateMembership(ctx, ownMembership); err != nil {
		t.Fatal(err)
	}

	client := entity.MCPClient{ID: ids.New(), UserID: user.ID, Name: "claude-web", RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"}, CreatedAt: now}
	if err := store.CreateMCPClient(ctx, client); err != nil {
		t.Fatal(err)
	}
	access, refresh, err := store.CreateMCPTokenPair(ctx, client.ID, user.ID, now.Add(time.Hour), now.Add(30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, "SELECT token_hash FROM mcp_tokens WHERE client_id = $1", client.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedTokens := 0
	for rows.Next() {
		storedTokens++
		var hash string
		if err := rows.Scan(&hash); err != nil {
			t.Fatal(err)
		}
		if len(hash) != 64 || hash == access || hash == refresh {
			t.Fatalf("unsafe stored MCP token: length=%d equals_raw=%t", len(hash), hash == access || hash == refresh)
		}
	}
	rows.Close()
	if storedTokens != 2 {
		t.Fatalf("stored token count = %d; want 2", storedTokens)
	}
	label := "codex"
	cliToken := entity.MCPToken{
		ID: ids.New(), Type: entity.MCPTokenCLI, UserID: user.ID, Label: &label,
		ExpiresAt: now.Add(90 * 24 * time.Hour), CreatedAt: now,
	}
	rawCLI, err := store.CreateMCPCLIToken(ctx, cliToken)
	if err != nil {
		t.Fatal(err)
	}
	var storedCLIHash, storedCLILabel string
	var storedCLIClientID *string
	if err := pool.QueryRow(ctx, `
		SELECT token_hash, label, client_id FROM mcp_tokens WHERE id = $1`, cliToken.ID,
	).Scan(&storedCLIHash, &storedCLILabel, &storedCLIClientID); err != nil {
		t.Fatal(err)
	}
	if len(storedCLIHash) != 64 || storedCLIHash == rawCLI || storedCLILabel != label || storedCLIClientID != nil {
		t.Fatalf("stored CLI token = hash_length:%d equals_raw:%t label:%q client:%v", len(storedCLIHash), storedCLIHash == rawCLI, storedCLILabel, storedCLIClientID)
	}
	listedCLI, err := store.ListMCPCLITokens(ctx, user.ID)
	if err != nil || len(listedCLI) != 1 || listedCLI[0].ID != cliToken.ID || listedCLI[0].Label == nil || *listedCLI[0].Label != label {
		t.Fatalf("listed CLI tokens = %+v %v", listedCLI, err)
	}
	otherCLI, err := store.ListMCPCLITokens(ctx, other.ID)
	if err != nil || len(otherCLI) != 0 {
		t.Fatalf("other user CLI tokens = %+v %v", otherCLI, err)
	}
	if _, err := store.VerifyMCPAccessToken(ctx, access, now); err != nil {
		t.Fatal(err)
	}
	verifiedCLI, err := store.VerifyMCPAccessToken(ctx, rawCLI, now)
	if err != nil || verifiedCLI.UserID != user.ID || verifiedCLI.Type != entity.MCPTokenCLI {
		t.Fatalf("verified CLI token = %+v %v", verifiedCLI, err)
	}
	if _, err := store.VerifyMCPAccessToken(ctx, rawCLI, cliToken.ExpiresAt); !errors.Is(err, output_port.ErrTokenExpired) {
		t.Fatalf("expired CLI token error = %v", err)
	}
	if _, err := store.VerifyMCPAccessToken(ctx, access, now.Add(2*time.Hour)); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("expired token error = %v", err)
	}

	visible, err := store.ListMCPVisibleBrains(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertVisibleSources(t, visible, map[entity.SourceID]string{privateOwn.SourceID: "owner", publicOther.SourceID: "public"}, privateOther.SourceID)
	if err := store.RevokeMembership(ctx, ownMembership.ID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	visible, err = store.ListMCPVisibleBrains(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertVisibleSources(t, visible, map[entity.SourceID]string{publicOther.SourceID: "public"}, privateOwn.SourceID, privateOther.SourceID)

	if err := store.CreateBrain(ctx, newOwn); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateMembership(ctx, entity.Membership{ID: ids.New(), BrainID: newOwn.ID, UserID: user.ID, Role: entity.RoleOwner, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	visible, err = store.ListMCPVisibleBrains(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertVisibleSources(t, visible, map[entity.SourceID]string{publicOther.SourceID: "public", newOwn.SourceID: "owner"}, privateOwn.SourceID, privateOther.SourceID)

	if _, err := pool.Exec(ctx, "UPDATE users SET state = 'suspended' WHERE id = $1", user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.VerifyMCPAccessToken(ctx, access, now); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("suspended user token error = %v", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE users SET state = 'active' WHERE id = $1", user.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeMCPCLIToken(ctx, cliToken.ID, other.ID, now.Add(time.Second)); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("foreign CLI revoke error = %v", err)
	}
	if err := store.RevokeMCPCLIToken(ctx, cliToken.ID, user.ID, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.VerifyMCPAccessToken(ctx, rawCLI, now); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("revoked CLI token error = %v", err)
	}
	if err := store.RevokeMCPToken(ctx, access, client.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.VerifyMCPAccessToken(ctx, access, now); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("revoked access token error = %v", err)
	}
	if _, _, _, err := store.RotateMCPRefreshToken(ctx, refresh, client.ID, now, now.Add(time.Hour), now.Add(30*24*time.Hour)); !errors.Is(err, output_port.ErrNotFound) {
		t.Fatalf("paired refresh token error = %v", err)
	}
}

func assertVisibleSources(t *testing.T, brains []entity.MCPVisibleBrain, want map[entity.SourceID]string, forbidden ...entity.SourceID) {
	t.Helper()
	got := make(map[entity.SourceID]string, len(brains))
	for _, brain := range brains {
		got[brain.SourceID] = brain.Role
	}
	for source, role := range want {
		if got[source] != role {
			t.Fatalf("visible brains = %+v; want %s as %q", brains, source, role)
		}
	}
	for _, source := range forbidden {
		if _, ok := got[source]; ok {
			t.Fatalf("forbidden source %s is visible: %+v", source, brains)
		}
	}
}
