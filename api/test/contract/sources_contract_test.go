//go:build contract

package contract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"brainhub/adapter/gbrain"
	"brainhub/usecase/output_port"
)

// TestSourcesAddWithOperationalScopesIsRejectedBeforeConfinement verifies the
// credentials Brainhub actually uses: read/write cannot invoke sources_add at all.
func TestSourcesAddWithOperationalScopesIsRejectedBeforeConfinement(t *testing.T) {
	assertSourcesAddRejected(t, []string{"read", "write"}, "insufficient_scope", "requires 'sources_admin' scope")
}

// TestSourcesAddWithSourcesAdminStillRejectsLocalPath verifies the security
// confinement contract itself. Brainhub does not normally hold sources_admin;
// this elevated test scope is used only to reach the path-specific guard.
func TestSourcesAddWithSourcesAdminStillRejectsLocalPath(t *testing.T) {
	assertSourcesAddRejected(t, []string{"read", "write", "sources_admin"}, "", "sources_add: path is not honored over MCP (security confinement).")
}

func assertSourcesAddRejected(t *testing.T, scopes []string, wantError, wantMessage string) {
	t.Helper()
	baseURL := os.Getenv("GBRAIN_CONTRACT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3131"
	}
	bootstrapToken := os.Getenv("GBRAIN_ADMIN_BOOTSTRAP_TOKEN")
	if bootstrapToken == "" {
		t.Fatal("GBRAIN_ADMIN_BOOTSTRAP_TOKEN is required for contract tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := gbrain.NewAdminClient(baseURL, bootstrapToken)
	if err != nil {
		t.Fatal(err)
	}
	source := "brainhub"
	registered, err := admin.RegisterClient(ctx, output_port.RegisterGBrainClientInput{
		Name:   "brainhub-contract-sources-add-" + fmt.Sprint(time.Now().UnixNano()),
		Scopes: scopes, Source: &source, FederatedRead: []string{source},
		GrantTypes: []string{"client_credentials"}, TokenEndpointAuthMethod: "client_secret_post",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if err := admin.RevokeClient(cleanupCtx, registered.ID); err != nil {
			t.Errorf("cleanup GBrain client: %v", err)
		}
	})

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {strings.Join(scopes, " ")},
		"client_id":     {registered.ID},
		"client_secret": {registered.Secret},
	}
	tokenRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResponse, err := http.DefaultClient.Do(tokenRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("token endpoint HTTP %d", tokenResponse.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResponse.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	if token.AccessToken == "" {
		t.Fatal("token endpoint returned no access_token")
	}

	sourceID := fmt.Sprintf("contract-path-%x", time.Now().UnixNano())
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{
			"name":      "sources_add",
			"arguments": map[string]any{"id": sourceID, "path": "/tmp/brainhub-contract-path"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/mcp", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("MCP endpoint HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "data: ") {
				body = []byte(strings.TrimPrefix(line, "data: "))
				break
			}
		}
	}
	var rpc struct {
		Result *struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &rpc); err != nil {
		t.Fatalf("decode MCP response: %v", err)
	}
	if rpc.Result == nil || !rpc.Result.IsError || len(rpc.Result.Content) == 0 {
		t.Fatalf("sources_add accepted a local path: %s", body)
	}
	var rejection struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(rpc.Result.Content[0].Text), &rejection); err != nil {
		t.Fatalf("decode sources_add rejection: %v", err)
	}
	if wantError != "" && rejection.Error != wantError {
		t.Fatalf("sources_add error = %q; want %q", rejection.Error, wantError)
	}
	if !strings.Contains(rejection.Message, wantMessage) {
		t.Fatalf("sources_add message = %q; want %q", rejection.Message, wantMessage)
	}
	t.Logf("%s: %s", rejection.Error, rejection.Message)
}
