package gbrain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"brainhub/usecase/output_port"
)

func TestAdminClientRegistersAndRevokesWithOneRelogin(t *testing.T) {
	var logins int32
	var registrations int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/login":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["token"] != "bootstrap-secret" {
				t.Errorf("admin login token was not delivered")
			}
			n := atomic.AddInt32(&logins, 1)
			http.SetCookie(w, &http.Cookie{Name: adminCookieName, Value: "session-" + string(rune('0'+n)), Path: "/admin", Secure: true, HttpOnly: true, MaxAge: 86400})
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "authenticated"})
		case "/admin/api/register-client":
			if r.Header.Get("Cookie") == "gbrain_admin=session-1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Cookie") != "gbrain_admin=session-2" {
				t.Errorf("register cookie = %q", r.Header.Get("Cookie"))
			}
			var body struct {
				Name                    string   `json:"name"`
				Scopes                  []string `json:"scopes"`
				Source                  *string  `json:"source"`
				FederatedRead           []string `json:"federatedRead"`
				GrantTypes              []string `json:"grantTypes"`
				RedirectURIs            []string `json:"redirectUris"`
				TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Name != "user-claude-web-issued" || !reflect.DeepEqual(body.Scopes, []string{"read", "write"}) || body.Source == nil || *body.Source != "brainhub" {
				t.Errorf("register body = %+v", body)
			}
			if !reflect.DeepEqual(body.FederatedRead, []string{"brainhub"}) || !reflect.DeepEqual(body.GrantTypes, []string{"authorization_code", "refresh_token"}) || body.TokenEndpointAuthMethod != "none" {
				t.Errorf("register restrictions = %+v", body)
			}
			atomic.AddInt32(&registrations, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"clientId": "public-client-id", "tokenTtl": nil})
		case "/admin/api/revoke-client":
			if r.Header.Get("Cookie") != "gbrain_admin=session-2" {
				t.Errorf("revoke cookie = %q", r.Header.Get("Cookie"))
			}
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["clientId"] != "public-client-id" {
				t.Errorf("revoke body = %v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]bool{"revoked": true})
		case "/admin/api/rescope-client":
			var body struct {
				ClientID      string   `json:"clientId"`
				FederatedRead []string `json:"federatedRead"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ClientID != "public-client-id" || !reflect.DeepEqual(body.FederatedRead, []string{"brainhub", "public-brain"}) {
				t.Errorf("rescope body = %+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"clientId": body.ClientID, "federatedRead": body.FederatedRead})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewAdminClient(server.URL, "bootstrap-secret")
	if err != nil {
		t.Fatal(err)
	}
	source := "brainhub"
	registered, err := client.RegisterClient(context.Background(), output_port.RegisterGBrainClientInput{
		Name: "user-claude-web-issued", Scopes: []string{"read", "write"}, Source: &source,
		FederatedRead: []string{"brainhub"}, GrantTypes: []string{"authorization_code", "refresh_token"},
		RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"}, TokenEndpointAuthMethod: "none",
	})
	if err != nil || registered.ID != "public-client-id" || registered.Secret != "" {
		t.Fatalf("RegisterClient = %+v, %v", registered, err)
	}
	if err := client.RescopeClient(context.Background(), registered.ID, []string{"brainhub", "public-brain"}); err != nil {
		t.Fatal(err)
	}
	if err := client.RevokeClient(context.Background(), registered.ID); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&logins) != 2 || atomic.LoadInt32(&registrations) != 1 {
		t.Fatalf("logins/registrations = %d/%d", logins, registrations)
	}
}
