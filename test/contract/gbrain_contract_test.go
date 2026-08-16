//go:build contract

package contract_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"brainhub/adapter/gbrain"
	"brainhub/usecase/output_port"
)

func TestGBrainAdminClientLifecycle(t *testing.T) {
	baseURL := os.Getenv("GBRAIN_CONTRACT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3131"
	}
	bootstrapToken := os.Getenv("GBRAIN_ADMIN_BOOTSTRAP_TOKEN")
	if bootstrapToken == "" {
		t.Fatal("GBRAIN_ADMIN_BOOTSTRAP_TOKEN is required for contract tests")
	}
	client, err := gbrain.NewAdminClient(baseURL, bootstrapToken)
	if err != nil {
		t.Fatal(err)
	}
	source := "brainhub"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	clientID, err := client.RegisterClient(ctx, output_port.RegisterGBrainClientInput{
		Name: fmt.Sprintf("brainhub-contract-%d", time.Now().UnixNano()), Scopes: []string{"read"}, Source: &source,
		FederatedRead: []string{source}, GrantTypes: []string{"authorization_code", "refresh_token"},
		RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"}, TokenEndpointAuthMethod: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if err := client.RevokeClient(cleanupCtx, clientID); err != nil {
			t.Errorf("cleanup GBrain client: %v", err)
		}
	})
	if err := client.RevokeClient(ctx, clientID); err != nil {
		t.Fatal(err)
	}
}
