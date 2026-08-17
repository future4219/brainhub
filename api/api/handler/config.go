package handler

import (
	"encoding/json"
	"net/http"

	"brainhub/api/schema"
)

func Config(publicMCPURL string) http.HandlerFunc {
	response := schema.ConfigResponse{MCPURL: publicMCPURL}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
