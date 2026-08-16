package router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"brainhub/api/api/router"
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

func (r *pageUseCase) ListTypes(_ context.Context, sourceID entity.SourceID, _ string) ([]entity.PageType, error) {
	if sourceID != "brainhub" {
		return nil, input_port.ErrBrainNotFound
	}
	return []entity.PageType{{Name: "decision", Primitive: "concept"}}, nil
}

func (r *pageUseCase) Create(_ context.Context, sourceID entity.SourceID, _ string, input input_port.CreatePageInput) (entity.PageDetail, error) {
	if sourceID != "brainhub" {
		return entity.PageDetail{}, input_port.ErrBrainNotFound
	}
	for _, page := range r.pages {
		if page.Slug == input.Slug {
			return entity.PageDetail{}, input_port.ErrPageAlreadyExists
		}
	}
	page := entity.Page{Slug: input.Slug, Title: input.Title, Type: input.Type}
	r.pages = append(r.pages, page)
	return entity.PageDetail{Page: page, CompiledTruth: input.CompiledTruth, Timeline: input.TimelineEntry, Tags: input.Tags, SupersededBy: input.SupersededBy}, nil
}

func (r *pageUseCase) Update(_ context.Context, sourceID entity.SourceID, slug, _ string, input input_port.UpdatePageInput) (entity.PageDetail, error) {
	if sourceID != "brainhub" {
		return entity.PageDetail{}, input_port.ErrBrainNotFound
	}
	for i, page := range r.pages {
		if page.Slug == slug {
			page.Title, page.Type = input.Title, input.Type
			r.pages[i] = page
			return entity.PageDetail{Page: page, CompiledTruth: input.CompiledTruth, Timeline: input.TimelineEntry, Tags: input.Tags, SupersededBy: input.SupersededBy}, nil
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
	now time.Time
}

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
	server := httptest.NewServer(router.New(brains, pages, auth, access, "https://mcp.example.com/mcp", proxy, false))
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
		if response.StatusCode != http.StatusOK || body["mcp_url"] != "https://mcp.example.com/mcp" {
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

	t.Run("page write routes", func(t *testing.T) {
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
		for _, test := range []struct {
			method string
			path   string
			body   string
			status int
		}{
			{http.MethodGet, "/api/brains/brainhub/page-types", "", http.StatusOK},
			{http.MethodPost, "/api/brains/brainhub/pages", `{"slug":"notes/new","title":"New","type":"decision","tags":[],"superseded_by":null,"compiled_truth":"truth","timeline_entry":"2026-08-17 initial"}`, http.StatusCreated},
			{http.MethodPost, "/api/brains/brainhub/pages", `{"slug":"notes/new","title":"New","type":"decision","tags":[],"superseded_by":null,"compiled_truth":"truth","timeline_entry":"2026-08-17 initial"}`, http.StatusConflict},
			{http.MethodPut, "/api/brains/brainhub/pages/notes/new", `{"title":"Updated","type":"decision","tags":[],"superseded_by":null,"compiled_truth":"updated","timeline_entry":"2026-08-17 changed"}`, http.StatusOK},
			{http.MethodPut, "/api/brains/brainhub/pages/missing", `{"title":"Missing","type":"decision","tags":[],"superseded_by":null,"compiled_truth":"updated","timeline_entry":""}`, http.StatusNotFound},
			{http.MethodPut, "/api/brains/brainhub/pages/notes/new", `{"title":"Bad","type":"decision","tags":[],"superseded_by":null,"compiled_truth":"updated","timeline":"replace all","timeline_entry":""}`, http.StatusBadRequest},
			{http.MethodPost, "/api/brains/brainhub/writer/reissue", "", http.StatusNoContent},
		} {
			response := request(test.method, test.path, test.body)
			response.Body.Close()
			if response.StatusCode != test.status {
				t.Errorf("%s %s = %d; want %d", test.method, test.path, response.StatusCode, test.status)
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
		productionServer := httptest.NewServer(router.New(brains, pages, productionAuth, access, "https://mcp.example.com/mcp", proxy, true))
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

	for _, path := range []string{
		"/mcp",
		"/mcp/tools",
		"/.well-known/oauth-authorization-server",
		"/token",
		"/revoke",
		"/register",
	} {
		t.Run("proxy "+path, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodDelete, server.URL+path, nil)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusNoContent || response.Header.Get("X-Proxied-Path") != path {
				t.Fatalf("proxy status/path = %d %q", response.StatusCode, response.Header.Get("X-Proxied-Path"))
			}
		})
	}

	t.Run("proxy authorize without following redirect", func(t *testing.T) {
		client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		response, err := client.Get(server.URL + "/authorize?response_type=code&client_id=test-client")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusFound {
			t.Fatalf("status = %d; want 302", response.StatusCode)
		}
		if got := response.Header.Get("Location"); got != "/admin/consent?request=test" {
			t.Fatalf("Location = %q", got)
		}
		if got := response.Header.Get("X-Proxied-Path"); got != "/authorize" {
			t.Fatalf("proxied path = %q", got)
		}
		if got := response.Header.Get("X-Proxied-Query"); got != "response_type=code&client_id=test-client" {
			t.Fatalf("proxied query = %q", got)
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
