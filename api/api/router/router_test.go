package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"brainhub/api/router"
	"brainhub/domain/entconst"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type pageUseCase struct {
	sourceID entity.SourceID
	pages    []entity.Page
}

func (r *pageUseCase) List(_ context.Context, sourceID entity.SourceID, _ string) ([]entity.Page, error) {
	r.sourceID = sourceID
	if sourceID != "brainhub" {
		return nil, input_port.ErrBrainNotFound
	}
	return r.pages, nil
}

func (r *pageUseCase) Get(_ context.Context, sourceID entity.SourceID, slug, _ string) (entity.PageDetail, error) {
	r.sourceID = sourceID
	if sourceID != "brainhub" {
		return entity.PageDetail{}, input_port.ErrBrainNotFound
	}
	for _, page := range r.pages {
		if page.Slug == slug {
			return entity.PageDetail{Page: page, CompiledTruth: "# Public\n\nBody", Timeline: "- 2026-08-15: Created"}, nil
		}
	}
	return entity.PageDetail{}, input_port.ErrPageNotFound
}

type brainUseCase struct {
	brains []entity.Brain
}

func (u *brainUseCase) Create(_ context.Context, ownerID string, input input_port.CreateBrainInput) (entity.Brain, error) {
	brain := entity.Brain{
		ID: "new-brain-id", SourceID: entity.SourceID(input.SourceID), Name: input.Name,
		Description: input.Description, Visibility: input.Visibility, OwnerID: ownerID,
		State: entconst.BrainStateReady,
	}
	u.brains = append(u.brains, brain)
	return brain, nil
}

func (u *brainUseCase) Adopt(_ context.Context, ownerID string, sourceID entity.SourceID, input input_port.AdoptBrainInput) (entity.Brain, error) {
	for _, brain := range u.brains {
		if brain.SourceID == sourceID {
			return entity.Brain{}, input_port.ErrBrainAlreadyExists
		}
	}
	if sourceID == "missing" {
		return entity.Brain{}, input_port.ErrSourceNotFound
	}
	if input.Visibility == "" {
		input.Visibility = entconst.VisibilityPrivate
	}
	brain := entity.Brain{
		ID: "adopted-brain-id", SourceID: sourceID, Name: input.Name, Description: input.Description,
		Visibility: input.Visibility, OwnerID: ownerID, State: entconst.BrainStateReady,
	}
	u.brains = append(u.brains, brain)
	return brain, nil
}

func (u *brainUseCase) List(context.Context, string) ([]entity.Brain, error) {
	return u.brains, nil
}

func (u *brainUseCase) Get(_ context.Context, sourceID entity.SourceID, _ string) (entity.Brain, error) {
	for _, brain := range u.brains {
		if brain.SourceID == sourceID {
			return brain, nil
		}
	}
	return entity.Brain{}, input_port.ErrBrainNotFound
}

func (u *brainUseCase) ReissueWriter(_ context.Context, sourceID entity.SourceID, _ string) error {
	for _, brain := range u.brains {
		if brain.SourceID == sourceID {
			return nil
		}
	}
	return input_port.ErrBrainNotFound
}

type authUseCase struct {
	active bool
	user   entity.User
}

type accessUseCase struct {
	now      time.Time
	archived entity.SourceID
}

type mcpUseCase struct{ revokedCLIToken string }

func (u *mcpUseCase) Connection(context.Context, string) (input_port.MCPConnection, error) {
	return input_port.MCPConnection{}, nil
}

func (u *mcpUseCase) IssueClient(_ context.Context, userID, name string) (entity.MCPClient, error) {
	return entity.MCPClient{ID: "mcp-client", UserID: userID, Name: "claude-web"}, nil
}

func (u *mcpUseCase) IssueCLIToken(_ context.Context, userID, label string) (input_port.IssuedCLIToken, error) {
	return input_port.IssuedCLIToken{
		Token: entity.MCPToken{
			ID: "cli-token-id", Type: entity.MCPTokenCLI, UserID: userID, Label: &label,
			CreatedAt: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
			ExpiresAt: time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		},
		RawToken: "raw-cli-token",
	}, nil
}

func (u *mcpUseCase) RevokeCLIToken(_ context.Context, id, _ string) error {
	u.revokedCLIToken = id
	return nil
}

func (u *mcpUseCase) ValidateAuthorization(_ context.Context, input input_port.OAuthAuthorizationRequest, userID string) (input_port.OAuthAuthorization, error) {
	if input.ClientID != "mcp-client" {
		return input_port.OAuthAuthorization{}, input_port.ErrOAuthInvalidClient
	}
	return input_port.OAuthAuthorization{Client: entity.MCPClient{ID: "mcp-client", UserID: userID, Name: "claude-web"}, Input: input}, nil
}

func (u *mcpUseCase) ApproveAuthorization(context.Context, input_port.OAuthAuthorizationRequest, string) (string, error) {
	return "authorization-code", nil
}

func (u *mcpUseCase) ExchangeAuthorizationCode(context.Context, string, string, string, string) (input_port.OAuthTokenPair, error) {
	return input_port.OAuthTokenPair{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresIn: time.Hour}, nil
}

func (u *mcpUseCase) RefreshAccessToken(context.Context, string, string) (input_port.OAuthTokenPair, error) {
	return input_port.OAuthTokenPair{AccessToken: "access-token", RefreshToken: "refresh-token", ExpiresIn: time.Hour}, nil
}

func (u *mcpUseCase) RevokeToken(context.Context, string, string) error { return nil }

func (u *mcpUseCase) AuthorizeCall(_ context.Context, token, _ string, _ *string) (input_port.MCPCallAuthorization, error) {
	if token != "access-token" {
		return input_port.MCPCallAuthorization{}, input_port.ErrMCPUnauthorized
	}
	return input_port.MCPCallAuthorization{UserID: "user-1"}, nil
}

func (u *mcpUseCase) ReissueReader(context.Context, string) error { return nil }
func (u *mcpUseCase) ReconcileReaders(context.Context) error      { return nil }

func (u *accessUseCase) CreateInvitation(_ context.Context, _ entity.SourceID, actorID string, input input_port.CreateInvitationInput) (entity.Invitation, string, error) {
	return entity.Invitation{ID: "invitation-id", BrainID: "brain-id", Email: nil, Role: input.Role, InvitedBy: actorID, State: entity.InvitationStatePending, ExpiresAt: input.ExpiresAt, CreatedAt: u.now}, "raw-invitation-token", nil
}

func (u *accessUseCase) ListInvitations(context.Context, entity.SourceID, string) ([]entity.Invitation, error) {
	return []entity.Invitation{{ID: "invitation-id", State: entity.InvitationStatePending, Role: entity.RoleReader, ExpiresAt: u.now.Add(time.Hour), CreatedAt: u.now}}, nil
}

func (u *accessUseCase) RevokeInvitation(context.Context, string, string) error { return nil }

func (u *accessUseCase) PreviewInvitation(_ context.Context, token string) (input_port.InvitationPreview, error) {
	if token != "raw-invitation-token" {
		return input_port.InvitationPreview{}, input_port.ErrInvitationNotFound
	}
	return input_port.InvitationPreview{BrainName: "Brainhub", InvitedByName: "Alice"}, nil
}

func (u *accessUseCase) AcceptInvitation(context.Context, string, entity.User) (input_port.AcceptedInvitation, error) {
	return input_port.AcceptedInvitation{Membership: entity.Membership{Role: entity.RoleReader}, SourceID: "brainhub", BrainName: "Brainhub"}, nil
}

func (u *accessUseCase) IssueClient(context.Context, entity.SourceID, string, string) (entity.IssuedClient, error) {
	clientID := "oauth-client-id"
	return entity.IssuedClient{ID: "issued-id", GBrainClientID: &clientID, Label: "claude-web", State: entity.ClientStateActive, IssuedAt: u.now}, nil
}

func (u *accessUseCase) ListClients(context.Context, entity.SourceID, string) ([]entity.IssuedClient, error) {
	client, _ := u.IssueClient(context.Background(), "brainhub", "user-id", "claude-web")
	return []entity.IssuedClient{client}, nil
}

func (u *accessUseCase) RevokeClient(context.Context, string, string) (entity.IssuedClient, error) {
	return entity.IssuedClient{ID: "issued-id", State: entity.ClientStateRevoked}, nil
}

func (u *accessUseCase) RevokeMembership(context.Context, entity.SourceID, string, string) error {
	return nil
}

func (u *accessUseCase) ArchiveBrain(_ context.Context, sourceID entity.SourceID, _ string) error {
	u.archived = sourceID
	return nil
}

func (u *authUseCase) Register(context.Context, input_port.RegisterInput) (entity.User, entity.Session, string, error) {
	u.active = true
	return u.user, entity.Session{ID: "session-id", UserID: u.user.ID, ExpiresAt: time.Now().Add(30 * 24 * time.Hour)}, "register-token", nil
}

func (u *authUseCase) Login(_ context.Context, input input_port.LoginInput) (entity.User, entity.Session, string, error) {
	if input.Email != "alice@example.com" || input.Password != "correct-password" {
		return entity.User{}, entity.Session{}, "", input_port.ErrInvalidCredentials
	}
	u.active = true
	return u.user, entity.Session{ID: "session-id", UserID: u.user.ID, ExpiresAt: time.Now().Add(30 * 24 * time.Hour)}, "login-token", nil
}

func (u *authUseCase) Authenticate(_ context.Context, rawToken string) (entity.User, entity.Session, error) {
	if !u.active || (rawToken != "register-token" && rawToken != "login-token") {
		return entity.User{}, entity.Session{}, input_port.ErrUnauthorized
	}
	return u.user, entity.Session{ID: "session-id", UserID: u.user.ID}, nil
}

func (u *authUseCase) Logout(context.Context, string) error {
	if !u.active {
		return errors.New("session inactive")
	}
	u.active = false
	return nil
}

func TestRoutes(t *testing.T) {
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	pages := &pageUseCase{
		pages: []entity.Page{
			{Slug: "decisions/public", Title: "Public", Type: "decision", UpdatedAt: now},
			{Slug: "tasks/public", Title: "Tasks", Type: "task-list", UpdatedAt: now},
		},
	}
	brains := &brainUseCase{brains: []entity.Brain{
		{
			ID: "brain-id", SourceID: "brainhub", Name: "Brainhub", Description: "Public brain",
			Visibility: entconst.VisibilityPublic, OwnerID: "user-id", State: entconst.BrainStateReady,
			CreatedAt: now, UpdatedAt: now,
		},
	}}
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proxied-Path", r.URL.Path)
		w.Header().Set("X-Proxied-Query", r.URL.RawQuery)
		if r.URL.Path == "/authorize" {
			w.Header().Set("Location", "/admin/consent?request=test")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	auth := &authUseCase{user: entity.User{
		ID: "user-id", Email: "alice@example.com", Name: "Alice", State: entconst.UserStateActive,
		CreatedAt: now, UpdatedAt: now,
	}}
	access := &accessUseCase{now: now}
	mcp := &mcpUseCase{}
	routes, err := router.New(brains, pages, auth, access, mcp, "https://mcp.example.com/mcp", "https://brainhub.example.com", proxy, false)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(routes)
	defer server.Close()

	t.Run("healthz", func(t *testing.T) {
		response, err := http.Get(server.URL + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body map[string]string
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || body["status"] != "ok" {
			t.Fatalf("status/body = %d %v", response.StatusCode, body)
		}
	})

	t.Run("config", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/config")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body map[string]string
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || body["mcp_url"] != "https://mcp.example.com/mcp" || body["web_url"] != "https://brainhub.example.com" {
			t.Fatalf("status/body = %d %v", response.StatusCode, body)
		}
	})

	t.Run("brains", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/brains")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body []map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || len(body) != 1 {
			t.Fatalf("status/brains = %d %v", response.StatusCode, body)
		}
		if body[0]["source_id"] != "brainhub" || body[0]["description"] != "Public brain" || body[0]["state"] != "ready" {
			t.Fatalf("brain response = %v", body[0])
		}
	})

	t.Run("pages", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/brains/brainhub/pages")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body []map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if pages.sourceID.String() != "brainhub" {
			t.Errorf("source ID = %q", pages.sourceID)
		}
		if len(body) != 2 {
			t.Fatalf("pages = %d; want 2 public types", len(body))
		}
		for _, page := range body {
			if len(page) != 4 {
				t.Errorf("page fields = %v; want exactly four", page)
			}
		}
	})

	t.Run("page detail", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/brains/brainhub/pages/decisions/public")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || body["slug"] != "decisions/public" || body["compiled_truth"] != "# Public\n\nBody" {
			t.Fatalf("status/page = %d %v", response.StatusCode, body)
		}
	})

	t.Run("missing page", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/brains/brainhub/pages/missing")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d; want 404", response.StatusCode)
		}
	})

	t.Run("missing brain", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/brains/missing/pages")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d; want 404", response.StatusCode)
		}
	})

	t.Run("invalid source ID", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/brains/default/pages")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d; want 400", response.StatusCode)
		}
	})

	t.Run("old pages route removed", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/pages")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d; want 404", response.StatusCode)
		}
	})

	t.Run("create brain requires session", func(t *testing.T) {
		response, err := http.Post(server.URL+"/api/brains", "application/json", bytes.NewBufferString(`{"source_id":"private-brain","name":"Private"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", response.StatusCode)
		}
	})

	t.Run("adopt brain requires session", func(t *testing.T) {
		response, err := http.Post(server.URL+"/api/brains/existing/adopt", "application/json", bytes.NewBufferString(`{"name":"Existing"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", response.StatusCode)
		}
	})

	var sessionCookie *http.Cookie
	t.Run("register", func(t *testing.T) {
		response, err := http.Post(server.URL+"/api/auth/register", "application/json", bytes.NewBufferString(`{"email":"alice@example.com","password":"correct-password","name":"Alice"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d; want 201", response.StatusCode)
		}
		cookies := response.Cookies()
		if len(cookies) != 1 {
			t.Fatalf("cookies = %v", cookies)
		}
		sessionCookie = cookies[0]
		if sessionCookie.Name != "brainhub_session" || sessionCookie.Value == "" || !sessionCookie.HttpOnly || sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteLaxMode || sessionCookie.Path != "/" {
			t.Fatalf("cookie attributes = %+v", sessionCookie)
		}
	})

	t.Run("me", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, server.URL+"/api/me", nil)
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var user map[string]any
		if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || user["email"] != "alice@example.com" {
			t.Fatalf("status/user = %d %v", response.StatusCode, user)
		}
	})

	t.Run("CLI token issue and revoke require session and reveal raw token once", func(t *testing.T) {
		unauthorized, err := http.Post(server.URL+"/api/mcp/tokens", "application/json", bytes.NewBufferString(`{"label":"codex"}`))
		if err != nil {
			t.Fatal(err)
		}
		unauthorized.Body.Close()
		if unauthorized.StatusCode != http.StatusUnauthorized {
			t.Fatalf("unauthorized status = %d", unauthorized.StatusCode)
		}

		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/mcp/tokens", bytes.NewBufferString(`{"label":"codex"}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var issued map[string]any
		if err := json.NewDecoder(response.Body).Decode(&issued); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusCreated || response.Header.Get("Cache-Control") != "no-store" || issued["token"] != "raw-cli-token" || issued["label"] != "codex" {
			t.Fatalf("issued = %d %q %v", response.StatusCode, response.Header.Get("Cache-Control"), issued)
		}

		revoke, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/mcp/tokens/cli-token-id", nil)
		revoke.AddCookie(sessionCookie)
		revoked, err := http.DefaultClient.Do(revoke)
		if err != nil {
			t.Fatal(err)
		}
		revoked.Body.Close()
		if revoked.StatusCode != http.StatusNoContent || mcp.revokedCLIToken != "cli-token-id" {
			t.Fatalf("revoke = %d %q", revoked.StatusCode, mcp.revokedCLIToken)
		}
	})

	t.Run("retired Web page editing endpoints are unavailable", func(t *testing.T) {
		for _, tc := range []struct {
			method, path string
			status       int
		}{
			{http.MethodGet, "/api/brains/brainhub/page-types", http.StatusNotFound},
			{http.MethodPost, "/api/brains/brainhub/pages", http.StatusMethodNotAllowed},
			{http.MethodPut, "/api/brains/brainhub/pages/notes/example", http.StatusMethodNotAllowed},
		} {
			request, _ := http.NewRequest(tc.method, server.URL+tc.path, strings.NewReader(`{}`))
			request.AddCookie(sessionCookie)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != tc.status {
				t.Errorf("%s %s = %d, want %d", tc.method, tc.path, response.StatusCode, tc.status)
			}
		}
	})

	t.Run("create brain", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/brains", bytes.NewBufferString(`{"source_id":"private-brain","name":"Private","description":"Owned","visibility":"private"}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusCreated || body["source_id"] != "private-brain" || body["owner_id"] != "user-id" {
			t.Fatalf("status/body = %d %v", response.StatusCode, body)
		}
	})

	t.Run("adopt brain", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/brains/adopted/adopt", bytes.NewBufferString(`{"name":"Adopted","description":"Existing source"}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var body map[string]any
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusCreated || body["source_id"] != "adopted" || body["visibility"] != "private" || body["state"] != "ready" {
			t.Fatalf("status/body = %d %v", response.StatusCode, body)
		}
	})

	t.Run("adopt missing source", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/brains/missing/adopt", bytes.NewBufferString(`{"name":"Missing"}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d; want 404", response.StatusCode)
		}
	})

	t.Run("adopt registered brain", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/brains/adopted/adopt", bytes.NewBufferString(`{"name":"Adopted"}`))
		request.Header.Set("Content-Type", "application/json")
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d; want 409", response.StatusCode)
		}
	})

	t.Run("invitation and client routes", func(t *testing.T) {
		preview, err := http.Get(server.URL + "/api/invitations/raw-invitation-token")
		if err != nil {
			t.Fatal(err)
		}
		defer preview.Body.Close()
		var previewBody map[string]string
		_ = json.NewDecoder(preview.Body).Decode(&previewBody)
		if preview.StatusCode != http.StatusOK || len(previewBody) != 2 || previewBody["brain_name"] != "Brainhub" {
			t.Fatalf("preview status/body = %d %v", preview.StatusCode, previewBody)
		}

		request := func(method, path, body string) *http.Response {
			req, _ := http.NewRequest(method, server.URL+path, bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(sessionCookie)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			return response
		}
		created := request(http.MethodPost, "/api/brains/brainhub/invitations", `{"role":"reader","expires_at":"2026-08-16T01:00:00Z"}`)
		var createdBody map[string]any
		_ = json.NewDecoder(created.Body).Decode(&createdBody)
		created.Body.Close()
		if created.StatusCode != http.StatusCreated || createdBody["token"] != "raw-invitation-token" {
			t.Fatalf("create invitation = %d %v", created.StatusCode, createdBody)
		}
		for _, test := range []struct {
			method string
			path   string
			body   string
			status int
		}{
			{http.MethodGet, "/api/brains/brainhub/invitations", "", http.StatusOK},
			{http.MethodPost, "/api/invitations/raw-invitation-token/accept", "", http.StatusCreated},
			{http.MethodPost, "/api/brains/brainhub/clients", `{"label":"claude-web"}`, http.StatusCreated},
			{http.MethodGet, "/api/brains/brainhub/clients", "", http.StatusOK},
			{http.MethodDelete, "/api/clients/issued-id", "", http.StatusNoContent},
			{http.MethodDelete, "/api/invitations/invitation-id", "", http.StatusNoContent},
			{http.MethodDelete, "/api/brains/brainhub/members/other-user", "", http.StatusNoContent},
		} {
			response := request(test.method, test.path, test.body)
			response.Body.Close()
			if response.StatusCode != test.status {
				t.Errorf("%s %s = %d; want %d", test.method, test.path, response.StatusCode, test.status)
			}
		}
	})

	t.Run("archive brain", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/brains/brainhub", nil)
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent || access.archived != "brainhub" {
			t.Fatalf("status/source = %d %q", response.StatusCode, access.archived)
		}
	})

	t.Run("me without cookie", func(t *testing.T) {
		response, err := http.Get(server.URL + "/api/me")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d; want 401", response.StatusCode)
		}
	})

	t.Run("login failure response is identical", func(t *testing.T) {
		request := func(body string) (int, string) {
			response, err := http.Post(server.URL+"/api/auth/login", "application/json", bytes.NewBufferString(body))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			responseBody, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			return response.StatusCode, string(responseBody)
		}
		wrongStatus, wrongBody := request(`{"email":"alice@example.com","password":"wrong-password"}`)
		missingStatus, missingBody := request(`{"email":"missing@example.com","password":"wrong-password"}`)
		if wrongStatus != http.StatusUnauthorized || wrongStatus != missingStatus || wrongBody != missingBody {
			t.Fatalf("wrong=(%d,%q) missing=(%d,%q)", wrongStatus, wrongBody, missingStatus, missingBody)
		}
	})

	t.Run("logout revokes session", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/auth/logout", nil)
		request.AddCookie(sessionCookie)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			t.Fatalf("logout status = %d; want 204", response.StatusCode)
		}

		request, _ = http.NewRequest(http.MethodGet, server.URL+"/api/me", nil)
		request.AddCookie(sessionCookie)
		response, err = http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("post-logout me status = %d; want 401", response.StatusCode)
		}
	})

	t.Run("production cookie is secure", func(t *testing.T) {
		productionAuth := &authUseCase{user: auth.user}
		productionRoutes, err := router.New(brains, pages, productionAuth, access, mcp, "https://mcp.example.com/mcp", "https://brainhub.example.com", proxy, true)
		if err != nil {
			t.Fatal(err)
		}
		productionServer := httptest.NewServer(productionRoutes)
		defer productionServer.Close()
		response, err := http.Post(productionServer.URL+"/api/auth/register", "application/json", bytes.NewBufferString(`{"email":"alice@example.com","password":"correct-password","name":"Alice"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if cookies := response.Cookies(); len(cookies) != 1 || !cookies[0].Secure {
			t.Fatalf("production cookies = %+v; want Secure", cookies)
		}
	})

	t.Run("mcp requires brainhub token and replaces it upstream", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, server.URL+"/mcp", nil)
		request.Header.Set("Authorization", "Bearer access-token")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusNoContent || response.Header.Get("X-Proxied-Path") != "/mcp" {
			t.Fatalf("proxy status/path = %d %q", response.StatusCode, response.Header.Get("X-Proxied-Path"))
		}
	})

	t.Run("oauth metadata is served by brainhub", func(t *testing.T) {
		response, err := http.Get(server.URL + "/.well-known/oauth-authorization-server")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var metadata map[string]any
		if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK ||
			metadata["issuer"] != "https://mcp.example.com/" ||
			metadata["authorization_endpoint"] != "https://mcp.example.com/authorize" ||
			metadata["token_endpoint"] != "https://mcp.example.com/token" ||
			metadata["revocation_endpoint"] != "https://mcp.example.com/revoke" {
			t.Fatalf("metadata = %d %v", response.StatusCode, metadata)
		}
	})

	t.Run("dynamic registration is rejected", func(t *testing.T) {
		response, err := http.Post(server.URL+"/register", "application/json", bytes.NewBufferString(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("register status = %d; want 400", response.StatusCode)
		}
	})

	t.Run("proxy authorize without following redirect", func(t *testing.T) {
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		response, err := client.Get(server.URL + "/authorize?response_type=code&client_id=mcp-client&redirect_uri=https%3A%2F%2Fclaude.ai%2Fapi%2Fmcp%2Fauth_callback&code_challenge=abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQ&code_challenge_method=S256&resource=https%3A%2F%2Fmcp.example.com%2Fmcp")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusFound {
			t.Fatalf("status = %d; want 302", response.StatusCode)
		}
		if got := response.Header.Get("Location"); !strings.HasPrefix(got, "https://brainhub.example.com/login?next=") {
			t.Fatalf("Location = %q", got)
		}
	})

	t.Run("healthz method", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, server.URL+"/healthz", nil)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d; want 405", response.StatusCode)
		}
	})
}
