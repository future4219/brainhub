package interactor_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"brainhub/adapter/gbrain"
	"brainhub/api/router"
	"brainhub/domain/entity"
	"brainhub/usecase/interactor"
)

// Exercise the real consent, PKCE, token, membership, proxy, and tool catalog
// boundaries together. The fake upstream never stores user data.
func TestMCPWriteConsentAndSourceBoundary(t *testing.T) {
	for _, sse := range []bool{false, true} {
		t.Run(fmt.Sprintf("SSE=%t", sse), func(t *testing.T) {
			const origin = "https://brainhub.example"
			now := time.Now()
			repo := newMCPRepositoryMock()
			repo.visible = []entity.MCPVisibleBrain{
				{ID: "own-id", SourceID: "own", Role: "owner", State: "ready"},
				{ID: "edit-id", SourceID: "editable", Role: "editor", State: "ready"},
				{ID: "read-id", SourceID: "reader", Role: "reader", State: "ready"},
				{ID: "public-id", SourceID: "public", Role: "public", State: "ready"},
				{ID: "broken-id", SourceID: "broken", Role: "owner", State: "degraded"},
			}
			writer := &mcpWriterMock{}
			uc, err := interactor.NewMCPUseCase(repo, &readerRepositoryMock{}, &brainReaderMock{}, writer, fixedClock{now}, &sequenceIDs{}, origin+"/mcp")
			if err != nil {
				t.Fatal(err)
			}
			writes := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				var result any
				if req["method"] == "tools/list" {
					if r.Header.Get("Authorization") != "Bearer gbrain-reader-token" {
						t.Error("catalog must use reader")
					}
					result = map[string]any{"tools": []any{map[string]any{"name": "get_page", "inputSchema": map[string]any{"type": "object"}}}}
				} else {
					params := req["params"].(map[string]any)
					if params["name"] == "put_page" {
						writes++
						if r.Header.Get("Authorization") != "Bearer gbrain-writer-token" {
							t.Error("write must use source-bound writer")
						}
						args := params["arguments"].(map[string]any)
						if _, ok := args["source_id"]; ok {
							t.Error("upstream put_page rejects source_id")
						}
						if args["content"] != "---\ntitle: Example\n---\nSaved" {
							t.Error("content changed")
						}
					} else if r.Header.Get("Authorization") != "Bearer gbrain-reader-token" {
						t.Error("other tools must never receive writer token")
					}
					result = map[string]any{"content": []any{map[string]string{"type": "text", "text": "ok"}}}
				}
				body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": result})
				if sse {
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", body)
				} else {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write(body)
				}
			}))
			defer upstream.Close()
			proxy, _ := gbrain.NewProxy(upstream.URL)
			routes, err := router.New(nil, nil, codexSessionAuth{}, nil, uc, origin+"/mcp", origin, proxy, false)
			if err != nil {
				t.Fatal(err)
			}
			request := func(method, path, body, token string, owner bool) *httptest.ResponseRecorder {
				r := httptest.NewRequest(method, origin+path, strings.NewReader(body))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if token != "" {
					r.Header.Set("Authorization", "Bearer "+token)
				}
				if owner {
					r.AddCookie(&http.Cookie{Name: "brainhub_session", Value: "owner"})
				}
				w := httptest.NewRecorder()
				routes.ServeHTTP(w, r)
				return w
			}
			client, _ := uc.IssueClient(context.Background(), "owner", "codex")
			verifier := strings.Repeat("a", 43)
			digest := sha256.Sum256([]byte(verifier))
			params := url.Values{"client_id": {client.ID}, "redirect_uri": {"http://127.0.0.1:45678/callback"}, "response_type": {"code"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(digest[:])}, "code_challenge_method": {"S256"}, "decision": {"approve"}}
			consent := request("GET", "/api/oauth/authorization?"+params.Encode(), "", "", true)
			if !strings.Contains(consent.Body.String(), `"scope":"read write"`) || !strings.Contains(consent.Body.String(), `"can_write":true`) {
				t.Fatalf("default consent: %s", consent.Body.String())
			}
			for _, tc := range []struct {
				requested, granted, want string
				status                   int
			}{
				{"", "", "read", 200}, // An old open consent tab cannot grant write.
				{"read write", "read", "read", 200},
				{"read write", "read write", "read write", 200},
				{"read", "read write", "", 400},
				{"read sources_admin", "read", "", 400},
			} {
				params.Set("scope", tc.requested)
				params.Set("granted_scope", tc.granted)
				w := request("POST", "/api/oauth/authorization", params.Encode(), "", true)
				if w.Code != tc.status {
					t.Fatalf("consent %+v: %d %s", tc, w.Code, w.Body.String())
				}
				if tc.status != 200 {
					continue
				}
				exchange := url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {repo.rawCode}, "redirect_uri": {params.Get("redirect_uri")}, "code_verifier": {verifier}}
				w = request("POST", "/token", exchange.Encode(), "", false)
				var pair map[string]any
				_ = json.Unmarshal(w.Body.Bytes(), &pair)
				if w.Code != 200 || pair["scope"] != tc.want {
					t.Fatalf("token scope: %s", w.Body.String())
				}
				refresh := url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ID}, "refresh_token": {"brainhub-refresh"}}
				w = request("POST", "/token", refresh.Encode(), "", false)
				_ = json.Unmarshal(w.Body.Bytes(), &pair)
				if pair["scope"] != tc.want {
					t.Fatal("refresh changed scope")
				}
				catalog := request("POST", "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "brainhub-access", false)
				if strings.Contains(catalog.Body.String(), `"name":"put_page"`) != (tc.want == "read write") {
					t.Fatalf("wrong catalog: %s", catalog.Body.String())
				}
				for _, source := range []any{"own", "editable", "reader", "public", "broken", "foreign", "__all__", "", nil, 42} {
					before := writes
					args := map[string]any{"slug": "notes/example", "content": "---\ntitle: Example\n---\nSaved"}
					if source != nil {
						args["source_id"] = source
					}
					body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "put_page", "arguments": args}})
					w := request("POST", "/mcp", string(body), "brainhub-access", false)
					wantWrite := tc.want == "read write" && (source == "own" || source == "editable")
					if (writes == before+1) != wantWrite {
						t.Fatalf("scope %s source %v: %d %s", tc.want, source, w.Code, w.Body.String())
					}
					if wantWrite && writer.sourceID.String() != source {
						t.Fatal("wrong source writer")
					}
					if !wantWrite && !strings.Contains(w.Body.String(), "permission_denied") {
						t.Fatalf("missing refusal: %s", w.Body.String())
					}
				}
			}
			// A previously consenting owner loses writes immediately after demotion.
			repo.token.WriteAllowed = true
			repo.visible[0].Role = "reader"
			before := writes
			w := request("POST", "/mcp", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"put_page","arguments":{"source_id":"own","slug":"notes/example","content":"replacement"}}}`, "brainhub-access", false)
			if writes != before || !strings.Contains(w.Body.String(), "permission_denied") {
				t.Fatal("demoted owner could write")
			}
			repo.visible[0].Role = "owner"
			writer.err = errors.New("writer unavailable")
			w = request("POST", "/mcp", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"put_page","arguments":{"source_id":"own","slug":"notes/example","content":"replacement"}}}`, "brainhub-access", false)
			if writes != before || w.Code != http.StatusServiceUnavailable {
				t.Fatal("unavailable writer must fail before reaching GBrain")
			}
			// Management operations continue through read credentials only.
			request("POST", "/mcp", `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"sources_add","arguments":{"id":"new"}}}`, "brainhub-access", false)
		})
	}
}
