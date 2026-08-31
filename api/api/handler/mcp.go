package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"brainhub/api/middleware"
	"brainhub/api/schema"
	"brainhub/usecase/input_port"
)

const maxMCPRequestBytes = 4 << 20

type MCPHandler struct {
	useCase             input_port.MCPUseCase
	proxy               http.Handler
	resourceMetadataURL string
}

func NewMCPHandler(useCase input_port.MCPUseCase, proxy http.Handler, resourceMetadataURL string) *MCPHandler {
	return &MCPHandler{useCase: useCase, proxy: proxy, resourceMetadataURL: resourceMetadataURL}
}

func (h *MCPHandler) Connection(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	connection, err := h.useCase.Connection(r.Context(), current.User.ID)
	if err != nil {
		http.Error(w, "failed to load MCP connection", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, schema.MCPConnectionResponseFromInput(connection))
}

func (h *MCPHandler) IssueClient(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	client, err := h.useCase.IssueClient(r.Context(), current.User.ID)
	if err != nil {
		http.Error(w, "failed to issue MCP client", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, schema.MCPClientResponseFromEntity(client))
}

func (h *MCPHandler) IssueCLIToken(w http.ResponseWriter, r *http.Request) {
	var request schema.CreateMCPCLITokenRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	current, _ := middleware.Current(r)
	issued, err := h.useCase.IssueCLIToken(r.Context(), current.User.ID, request.Label)
	if errors.Is(err, input_port.ErrMCPInvalidTokenLabel) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "label must be between 1 and 64 characters"})
		return
	}
	if err != nil {
		http.Error(w, "failed to issue MCP CLI token", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, schema.CreatedMCPCLITokenResponse{
		MCPCLITokenResponse: schema.MCPCLITokenResponseFromEntity(issued.Token),
		Token:               issued.RawToken,
	})
}

func (h *MCPHandler) RevokeCLIToken(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	if err := h.useCase.RevokeCLIToken(r.Context(), r.PathValue("id"), current.User.ID); errors.Is(err, input_port.ErrMCPTokenNotFound) {
		http.Error(w, "CLI token not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "failed to revoke MCP CLI token", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MCPHandler) ReissueReader(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	if err := h.useCase.ReissueReader(r.Context(), current.User.ID); err != nil {
		http.Error(w, "failed to reissue reader client", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		h.unauthorized(w)
		return
	}
	var request map[string]any
	var toolName string
	var requestedSource *string
	var body []byte
	if r.Method == http.MethodPost {
		var err error
		body, err = io.ReadAll(io.LimitReader(r.Body, maxMCPRequestBytes+1))
		if err != nil || len(body) > maxMCPRequestBytes {
			http.Error(w, "invalid MCP request body", http.StatusBadRequest)
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid MCP JSON-RPC request", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "invalid MCP JSON-RPC request", http.StatusBadRequest)
			return
		}
		toolName, requestedSource = mcpTool(request)
	}
	authorization, err := h.useCase.AuthorizeCall(r.Context(), rawToken, toolName, requestedSource)
	switch {
	case errors.Is(err, input_port.ErrMCPTokenExpired):
		h.expiredToken(w)
		return
	case errors.Is(err, input_port.ErrMCPUnauthorized):
		h.unauthorized(w)
		return
	case errors.Is(err, input_port.ErrMCPUnsupportedTool):
		writeMCPToolError(w, request["id"], "unsupported_tool", "search is not available. Use query instead — it supports the same intent and respects your access scope.")
		return
	case errors.Is(err, input_port.ErrMCPForbidden):
		writeMCPToolError(w, request["id"], "permission_denied", "The requested source is outside your current brainhub access scope.")
		return
	case err != nil:
		http.Error(w, "MCP reader is unavailable; open brainhub connection settings and follow the recovery message", http.StatusServiceUnavailable)
		return
	}
	if authorization.SourceID != nil {
		params, _ := request["params"].(map[string]any)
		arguments, _ := params["arguments"].(map[string]any)
		if arguments == nil {
			arguments = make(map[string]any)
			params["arguments"] = arguments
		}
		arguments["source_id"] = *authorization.SourceID
		var marshalErr error
		body, marshalErr = json.Marshal(request)
		if marshalErr != nil {
			http.Error(w, "failed to prepare MCP request", http.StatusInternalServerError)
			return
		}
	}
	if r.Method == http.MethodPost {
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	}
	// The caller's brainhub token is never valid at GBrain. Only the per-user
	// reader token produced after current Membership reconciliation is forwarded.
	r.Header.Set("Authorization", "Bearer "+authorization.GBrainToken)
	h.proxy.ServeHTTP(w, r)
}

func mcpTool(request map[string]any) (string, *string) {
	if request["method"] != "tools/call" {
		return "", nil
	}
	params, _ := request["params"].(map[string]any)
	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]any)
	source, ok := arguments["source_id"].(string)
	if !ok {
		return name, nil
	}
	return name, &source
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	returnToken := len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != ""
	if !returnToken {
		return "", false
	}
	return parts[1], true
}

func (h *MCPHandler) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+h.resourceMetadataURL+`"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (h *MCPHandler) expiredToken(w http.ResponseWriter) {
	const description = "CLI token expired. Issue a new CLI token in brainhub connection settings and replace the configured bearer token."
	w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+h.resourceMetadataURL+`", error="invalid_token", error_description="`+description+`"`)
	writeJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "token_expired", "error_description": description,
	})
}

func writeMCPToolError(w http.ResponseWriter, id any, code, message string) {
	payload, _ := json.Marshal(map[string]string{"error": code, "message": message})
	writeJSON(w, http.StatusOK, map[string]any{
		"jsonrpc": "2.0", "id": id,
		"result": map[string]any{
			"content": []map[string]string{{"type": "text", "text": string(payload)}},
			"isError": true,
		},
	})
}
