package interactor_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"brainhub/api/router"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
	"brainhub/usecase/interactor"
)

type codexSessionAuth struct{ input_port.AuthUseCase }

func (codexSessionAuth) Authenticate(_ context.Context, cookie string) (entity.User, entity.Session, error) {
	if cookie != "owner" && cookie != "other" {
		return entity.User{}, entity.Session{}, input_port.ErrUnauthorized
	}
	return entity.User{ID: cookie}, entity.Session{}, nil
}

func TestCodexBrowserOAuthFlow(t *testing.T) {
	const origin = "https://brainhub.example"
	for _, callback := range []string{"http://127.0.0.1:45678/callback", "http://127.0.0.1:45678/callback/BsVxUT8GHgJK"} {
		t.Run(callback, func(t *testing.T) {
			now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
			repo := newMCPRepositoryMock()
			repo.visible = []entity.MCPVisibleBrain{{SourceID: "my-brain", Name: "My brain", Role: "owner", State: "ready"}}
			reader := &brainReaderMock{}
			uc, err := interactor.NewMCPUseCase(repo, &readerRepositoryMock{}, reader, &mcpWriterMock{}, fixedClock{now}, &sequenceIDs{}, origin+"/mcp")
			if err != nil {
				t.Fatal(err)
			}
			upstreamCalled := false
			routes, err := router.New(nil, nil, codexSessionAuth{}, nil, uc, origin+"/mcp", origin,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					upstreamCalled = r.Header.Get("Authorization") == "Bearer gbrain-reader-token"
					w.WriteHeader(http.StatusOK)
				}), false)
			if err != nil {
				t.Fatal(err)
			}
			request := func(method, path, body, cookie string) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, origin+path, strings.NewReader(body))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if cookie != "" {
					r.AddCookie(&http.Cookie{Name: "brainhub_session", Value: cookie})
				}
				w := httptest.NewRecorder()
				routes.ServeHTTP(w, r)
				return w
			}
			decode := func(w *httptest.ResponseRecorder, target any) {
				t.Helper()
				if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
					t.Fatalf("response %d: %s: %v", w.Code, w.Body.String(), err)
				}
			}
			if w := request("POST", "/api/mcp/clients/codex", "", ""); w.Code != 401 {
				t.Fatalf("anonymous issuance = %d", w.Code)
			}
			w := request("POST", "/api/mcp/clients/codex", "", "owner")
			var client struct{ ID string }
			decode(w, &client)
			if w.Code != 201 || client.ID == "" {
				t.Fatalf("issue = %d %s", w.Code, w.Body.String())
			}
			w = request("POST", "/api/mcp/clients/codex", "", "owner")
			var repeated struct{ ID string }
			decode(w, &repeated)
			if repeated.ID != client.ID {
				t.Fatal("preparing again must reuse the client")
			}
			if w := request("POST", "/api/mcp/clients/unknown", "", "owner"); w.Code != 400 {
				t.Fatalf("unknown client = %d", w.Code)
			}
			var connection struct {
				CodexClient struct{ ID string } `json:"codex_client"`
			}
			decode(request("GET", "/api/mcp/connection", "", "owner"), &connection)
			if connection.CodexClient.ID != client.ID {
				t.Fatal("connection must expose the prepared Codex client")
			}
			var metadata map[string]any
			decode(request("GET", "/.well-known/oauth-authorization-server", "", ""), &metadata)
			if metadata["authorization_response_iss_parameter_supported"] != true || metadata["issuer"] != origin+"/" {
				t.Fatalf("metadata = %v", metadata)
			}

			verifier := strings.Repeat("a", 43)
			digest := sha256.Sum256([]byte(verifier))
			params := url.Values{
				"client_id": {client.ID}, "redirect_uri": {callback}, "response_type": {"code"},
				"code_challenge": {base64.RawURLEncoding.EncodeToString(digest[:])}, "code_challenge_method": {"S256"},
				"resource": {origin + "/mcp"}, "state": {"codex-state"},
			}
			w = request("GET", "/authorize?"+params.Encode(), "", "")
			if w.Code != 302 || !strings.HasPrefix(w.Header().Get("Location"), origin+"/login?next=") {
				t.Fatalf("login redirect = %d %s", w.Code, w.Header().Get("Location"))
			}
			w = request("GET", "/authorize?"+params.Encode(), "", "owner")
			if w.Code != 302 || !strings.HasPrefix(w.Header().Get("Location"), origin+"/oauth/authorize?") {
				t.Fatal("logged-in user must see consent")
			}
			if w := request("GET", "/api/oauth/authorization?"+params.Encode(), "", "other"); w.Code != 403 {
				t.Fatalf("foreign user consent = %d", w.Code)
			}
			var consent struct {
				ClientName string `json:"client_name"`
				Visible    []struct {
					SourceID string `json:"source_id"`
				} `json:"visible_brains"`
			}
			decode(request("GET", "/api/oauth/authorization?"+params.Encode(), "", "owner"), &consent)
			if consent.ClientName != "codex" || len(consent.Visible) != 1 || consent.Visible[0].SourceID != "my-brain" {
				t.Fatalf("consent = %+v", consent)
			}

			for _, bad := range []string{
				"https://evil.test/callback", "http://127.0.0.1.evil.test:45678/callback", "http://localhost:45678/callback",
				"http://127.0.0.1:45678/other", "http://127.0.0.1:45678/callback/extra", "http://127.0.0.1:45678/callback?next=evil",
				"http://user@127.0.0.1:45678/callback", "http://127.0.0.1:0/callback", "http://127.0.0.1:65536/callback",
				"http://127.0.0.1:45678/%63allback", "http://127.0.0.1:45678/callback#fragment",
			} {
				params.Set("redirect_uri", bad)
				if w := request("GET", "/authorize?"+params.Encode(), "", "owner"); w.Code != 400 {
					t.Errorf("accepted callback %q: %d", bad, w.Code)
				}
			}
			params.Set("redirect_uri", callback)
			params.Set("decision", "deny")
			var decided struct {
				Redirect string `json:"redirect_uri"`
			}
			decode(request("POST", "/api/oauth/authorization", params.Encode(), "owner"), &decided)
			redirect, _ := url.Parse(decided.Redirect)
			if redirect.Query().Get("error") != "access_denied" || redirect.Query().Get("iss") != origin+"/" || repo.code.ID != "" {
				t.Fatal("denial must return issuer without issuing a code")
			}
			params.Set("decision", "approve")
			decode(request("POST", "/api/oauth/authorization", params.Encode(), "owner"), &decided)
			redirect, _ = url.Parse(decided.Redirect)
			if redirect.Scheme+"://"+redirect.Host+redirect.Path != callback || redirect.Query().Get("state") != "codex-state" || redirect.Query().Get("iss") != origin+"/" {
				t.Fatalf("callback = %s", decided.Redirect)
			}
			exchange := url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {redirect.Query().Get("code")}, "redirect_uri": {callback}, "code_verifier": {verifier}}
			exchange.Set("redirect_uri", "http://127.0.0.1:45679/callback")
			if w := request("POST", "/token", exchange.Encode(), ""); w.Code != 400 {
				t.Fatal("token exchange must use the original listener port")
			}
			exchange.Set("redirect_uri", callback)
			exchange.Set("code_verifier", strings.Repeat("b", 43))
			if w := request("POST", "/token", exchange.Encode(), ""); w.Code != 400 {
				t.Fatal("wrong verifier must fail")
			}
			exchange.Set("code_verifier", verifier)
			w = request("POST", "/token", exchange.Encode(), "")
			var tokens struct {
				Access  string `json:"access_token"`
				Refresh string `json:"refresh_token"`
				Scope   string `json:"scope"`
			}
			decode(w, &tokens)
			if w.Code != 200 || tokens.Access == "" || tokens.Refresh == "" || tokens.Scope != "read" {
				t.Fatalf("tokens = %d %+v", w.Code, tokens)
			}
			if w := request("POST", "/token", exchange.Encode(), ""); w.Code != 400 {
				t.Fatal("code replay must fail")
			}
			r := httptest.NewRequest("POST", origin+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
			r.Header.Set("Authorization", "Bearer "+tokens.Access)
			w = httptest.NewRecorder()
			routes.ServeHTTP(w, r)
			if w.Code != 200 || !upstreamCalled || len(reader.sources) != 1 || reader.sources[0] != "my-brain" {
				t.Fatal("authorized Codex call must reach the current user's reader")
			}

		})
	}
}
