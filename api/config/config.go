package config

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"brainhub/usecase/interactor"
)

const defaultPublicURL = "http://localhost:8080"
const defaultPublicWebURL = "http://localhost:3000"

type Config struct {
	DatabaseURL         string
	GBrainBaseURL       string
	GBrainClientID      string
	GBrainClientSecret  string
	GBrainAdminToken    string
	WriterCredentialKey string
	ShimURL             string
	ShimToken           string
	PublicMCPURL        string
	PublicWebURL        string
	PublicPageTypes     string
	Production          bool
}

func Load() (Config, error) {
	databaseURL, err := DatabaseURL()
	if err != nil {
		return Config{}, err
	}
	clientID := os.Getenv("BRAINHUB_GBRAIN_CLIENT_ID")
	clientSecret := os.Getenv("BRAINHUB_GBRAIN_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return Config{}, errors.New("BRAINHUB_GBRAIN_CLIENT_ID and BRAINHUB_GBRAIN_CLIENT_SECRET are required")
	}
	adminToken := os.Getenv("GBRAIN_ADMIN_BOOTSTRAP_TOKEN")
	if adminToken == "" {
		return Config{}, errors.New("GBRAIN_ADMIN_BOOTSTRAP_TOKEN is required")
	}
	writerCredentialKey := os.Getenv("BRAINHUB_WRITER_CREDENTIAL_KEY")
	if writerCredentialKey == "" {
		return Config{}, errors.New("BRAINHUB_WRITER_CREDENTIAL_KEY is required")
	}
	shimToken := os.Getenv("SHIM_TOKEN")
	if shimToken == "" {
		return Config{}, errors.New("SHIM_TOKEN is required")
	}
	publicURL := strings.TrimRight(envOrDefault("BRAINHUB_PUBLIC_URL", defaultPublicURL), "/")
	publicWebURL := strings.TrimRight(envOrDefault("BRAINHUB_PUBLIC_WEB_URL", defaultPublicWebURL), "/")

	return Config{
		DatabaseURL:         databaseURL,
		GBrainBaseURL:       envOrDefault("GBRAIN_BASE_URL", "http://localhost:3131"),
		GBrainClientID:      clientID,
		GBrainClientSecret:  clientSecret,
		GBrainAdminToken:    adminToken,
		WriterCredentialKey: writerCredentialKey,
		ShimURL:             envOrDefault("BRAINHUB_SHIM_URL", "http://127.0.0.1:8081"),
		ShimToken:           shimToken,
		PublicMCPURL:        publicURL + "/mcp",
		PublicWebURL:        publicWebURL,
		PublicPageTypes:     envOrDefault("GBRAIN_PUBLIC_PAGE_TYPES", interactor.DefaultPublicPageTypes),
		Production:          os.Getenv("BRAINHUB_ENV") == "production",
	}, nil
}

func DatabaseURL() (string, error) {
	value := os.Getenv("BRAINHUB_DATABASE_URL")
	if value == "" {
		return "", errors.New("BRAINHUB_DATABASE_URL is required")
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return "", errors.New("BRAINHUB_DATABASE_URL must be a PostgreSQL URL")
	}
	if parsed.Host == "" || strings.TrimPrefix(parsed.Path, "/") != "brainhub" {
		return "", errors.New("BRAINHUB_DATABASE_URL must target the brainhub database")
	}
	return value, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
