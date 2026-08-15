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
	"brainhub/usecase/interactor"
)

type pageRepository struct {
	sourceID entity.SourceID
	pages    []entity.Page
	sources  []entity.Source
}

func (r *pageRepository) List(_ context.Context, sourceID entity.SourceID) ([]entity.Page, error) {
	r.sourceID = sourceID
	return r.pages, nil
}

func (r *pageRepository) ListSources(context.Context) ([]entity.Source, error) {
	return r.sources, nil
}

type authUseCase struct {
	active bool
	user   entity.User
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
	repository := &pageRepository{
		pages: []entity.Page{
			{Slug: "decisions/public", Title: "Public", Type: "decision", UpdatedAt: now},
			{Slug: "tasks/public", Title: "Tasks", Type: "task-list", UpdatedAt: now},
			{Slug: "extracts/internal", Title: "Internal", Type: "extract_receipt", UpdatedAt: now},
		},
		sources: []entity.Source{
			{ID: "default", Name: "default"},
			{ID: "brainhub", Name: "brainhub", Federated: true, PageCount: 3, LastSyncAt: &now},
		},
	}
	brainUseCase, err := interactor.NewBrainUseCase(repository)
	if err != nil {
		t.Fatal(err)
	}
	pageUseCase, err := interactor.NewPageUseCase(repository, repository, interactor.DefaultPublicPageTypes)
	if err != nil {
		t.Fatal(err)
	}
	proxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proxied-Path", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	auth := &authUseCase{user: entity.User{
		ID: "user-id", Email: "alice@example.com", Name: "Alice", State: entconst.UserStateActive,
		CreatedAt: now, UpdatedAt: now,
	}}
	server := httptest.NewServer(router.New(brainUseCase, pageUseCase, auth, "https://mcp.example.com/mcp", proxy, false))
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
		var body struct {
			Sources []map[string]any `json:"sources"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || len(body.Sources) != 2 {
			t.Fatalf("status/sources = %d %v", response.StatusCode, body.Sources)
		}
		if len(body.Sources[1]) != 7 {
			t.Fatalf("source fields = %v; want GBrain's seven fields", body.Sources[1])
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
		if repository.sourceID.String() != "brainhub" {
			t.Errorf("source ID = %q", repository.sourceID)
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
		productionServer := httptest.NewServer(router.New(brainUseCase, pageUseCase, productionAuth, "https://mcp.example.com/mcp", proxy, true))
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

	for _, path := range []string{"/mcp", "/mcp/tools", "/.well-known/oauth-authorization-server"} {
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
