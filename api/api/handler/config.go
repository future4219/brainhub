package handler

import (
	"net/http"

	"brainhub/api/schema"
)

func Config(publicMCPURL, publicWebURL string) http.HandlerFunc {
	response := schema.ConfigResponse{MCPURL: publicMCPURL, WebURL: publicWebURL}
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, response)
	}
}
