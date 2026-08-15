package config

import (
	"errors"
	"net/url"
	"os"
	"strings"

	"brainhub/usecase/interactor"
)

const defaultPublicMCPURL = "http://localhost:8080/mcp"

type Config struct {
	DatabaseURL        string
	GBrainBaseURL      string
	GBrainClientID     string
	GBrainClientSecret string
	PublicMCPURL       string
	PublicPageTypes    string
	Production         bool
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

	return Config{
		DatabaseURL:        databaseURL,
		GBrainBaseURL:      envOrDefault("GBRAIN_BASE_URL", "http://localhost:3131"),
		GBrainClientID:     clientID,
		GBrainClientSecret: clientSecret,
		PublicMCPURL:       envOrDefault("PUBLIC_MCP_URL", defaultPublicMCPURL),
		PublicPageTypes:    envOrDefault("GBRAIN_PUBLIC_PAGE_TYPES", interactor.DefaultPublicPageTypes),
		Production:         os.Getenv("BRAINHUB_ENV") == "production",
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
