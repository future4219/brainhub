package config

import "testing"

func TestLoadDerivesPublicURLs(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("BRAINHUB_PUBLIC_URL", "https://brainhub.example/")
	t.Setenv("BRAINHUB_PUBLIC_WEB_URL", "https://app.example/")

	configuration, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.PublicMCPURL != "https://brainhub.example/mcp" {
		t.Fatalf("PublicMCPURL = %q", configuration.PublicMCPURL)
	}
	if configuration.PublicWebURL != "https://app.example" {
		t.Fatalf("PublicWebURL = %q", configuration.PublicWebURL)
	}
}

func TestLoadUsesLocalPublicURLDefaults(t *testing.T) {
	setRequiredEnvironment(t)
	t.Setenv("BRAINHUB_PUBLIC_URL", "")
	t.Setenv("BRAINHUB_PUBLIC_WEB_URL", "")

	configuration, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.PublicMCPURL != "http://localhost:8080/mcp" || configuration.PublicWebURL != "http://localhost:3000" {
		t.Fatalf("public URLs = %q %q", configuration.PublicMCPURL, configuration.PublicWebURL)
	}
}

func setRequiredEnvironment(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{
		"BRAINHUB_DATABASE_URL":          "postgresql://brainhub:password@localhost:5432/brainhub",
		"BRAINHUB_GBRAIN_CLIENT_ID":      "client-id",
		"BRAINHUB_GBRAIN_CLIENT_SECRET":  "client-secret",
		"GBRAIN_ADMIN_BOOTSTRAP_TOKEN":   "admin-token",
		"BRAINHUB_WRITER_CREDENTIAL_KEY": "writer-key",
		"SHIM_TOKEN":                     "shim-token",
	} {
		t.Setenv(name, value)
	}
}
