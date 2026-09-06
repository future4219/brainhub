package handler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"brainhub/api/middleware"
	"brainhub/api/schema"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type OAuthHandler struct {
	useCase input_port.MCPUseCase
	mcpURL  string
	webURL  string
	issuer  string
}

func NewOAuthHandler(useCase input_port.MCPUseCase, mcpURL, webURL string) (*OAuthHandler, error) {
	mcp, err := url.Parse(mcpURL)
	if err != nil || mcp.Scheme == "" || mcp.Host == "" {
		return nil, errors.New("public MCP URL is invalid")
	}
	web, err := url.Parse(webURL)
	if err != nil || web.Scheme == "" || web.Host == "" {
		return nil, errors.New("public web URL is invalid")
	}
	mcp.Path, mcp.RawPath, mcp.RawQuery, mcp.Fragment = "/", "", "", ""
	return &OAuthHandler{useCase: useCase, mcpURL: mcpURL, webURL: strings.TrimRight(webURL, "/"), issuer: mcp.String()}, nil
}

func (h *OAuthHandler) AuthorizationServerMetadata(w http.ResponseWriter, _ *http.Request) {
	h.cors(w)
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                         h.issuer,
		"authorization_endpoint":                         h.issuer + "authorize",
		"token_endpoint":                                 h.issuer + "token",
		"revocation_endpoint":                            h.issuer + "revoke",
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"revocation_endpoint_auth_methods_supported":     []string{"none"},
		"scopes_supported":                               []string{"read", "write"},
		"authorization_response_iss_parameter_supported": true,
	})
}

func (h *OAuthHandler) ProtectedResourceMetadata(w http.ResponseWriter, _ *http.Request) {
	h.cors(w)
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":              h.mcpURL,
		"authorization_servers": []string{h.issuer},
		"scopes_supported":      []string{"read", "write"},
		"resource_name":         "brainhub MCP Server",
	})
}

func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	input := authorizationInput(r.FormValue)
	if _, err := h.useCase.ValidateAuthorization(r.Context(), input, ""); err != nil {
		h.authorizationError(w, err)
		return
	}
	consentPath := "/oauth/authorize?" + r.URL.Query().Encode()
	if current, ok := middleware.Current(r); ok {
		if _, err := h.useCase.ValidateAuthorization(r.Context(), input, current.User.ID); err != nil {
			h.authorizationError(w, err)
			return
		}
		http.Redirect(w, r, h.webURL+consentPath, http.StatusFound)
		return
	}
	login := h.webURL + "/login?next=" + url.QueryEscape(consentPath)
	http.Redirect(w, r, login, http.StatusFound)
}

func (h *OAuthHandler) Consent(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	authorization, err := h.useCase.ValidateAuthorization(r.Context(), authorizationInput(r.URL.Query().Get), current.User.ID)
	if err != nil {
		h.authorizationError(w, err)
		return
	}
	connection, err := h.useCase.Connection(r.Context(), current.User.ID)
	if err != nil {
		http.Error(w, "failed to load access scope", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"client_name":    authorization.Client.Name,
		"scope":          authorization.Input.Scope,
		"client_id":      authorization.Client.ID,
		"visible_brains": schema.MCPConnectionResponseFromInput(connection).VisibleBrains,
	})
}

func (h *OAuthHandler) Decide(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid authorization request.")
		return
	}
	current, _ := middleware.Current(r)
	input := authorizationInput(r.Form.Get)
	authorization, err := h.useCase.ValidateAuthorization(r.Context(), input, current.User.ID)
	if err != nil {
		h.authorizationError(w, err)
		return
	}
	redirect, _ := url.Parse(authorization.Input.RedirectURI)
	query := redirect.Query()
	query.Set("iss", h.issuer)
	if state := r.Form.Get("state"); state != "" {
		query.Set("state", state)
	}
	if r.Form.Get("decision") != "approve" {
		query.Set("error", "access_denied")
		query.Set("error_description", "The user denied the authorization request.")
	} else {
		code, err := h.useCase.ApproveAuthorization(r.Context(), input, current.User.ID)
		if err != nil {
			h.authorizationError(w, err)
			return
		}
		query.Set("code", code)
	}
	redirect.RawQuery = query.Encode()
	writeJSON(w, http.StatusOK, map[string]string{"redirect_uri": redirect.String()})
}

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	h.cors(w)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid token request.")
		return
	}
	if _, _, ok := r.BasicAuth(); ok || r.Form.Get("client_secret") != "" {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "This public client does not use a client secret.")
		return
	}
	clientID := r.Form.Get("client_id")
	if clientID == "" {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "client_id is required.")
		return
	}
	var pair input_port.OAuthTokenPair
	var err error
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		pair, err = h.useCase.ExchangeAuthorizationCode(
			r.Context(), clientID, r.Form.Get("code"), r.Form.Get("redirect_uri"), r.Form.Get("code_verifier"),
		)
	case "refresh_token":
		pair, err = h.useCase.RefreshAccessToken(r.Context(), clientID, r.Form.Get("refresh_token"))
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "Only authorization_code and refresh_token are supported.")
		return
	}
	if err != nil {
		h.tokenError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  pair.AccessToken,
		"token_type":    "Bearer",
		"expires_in":    int64(pair.ExpiresIn.Seconds()),
		"refresh_token": pair.RefreshToken,
		"scope":         entity.MCPScope(pair.WriteAllowed),
	})
}

func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	h.cors(w)
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid revocation request.")
		return
	}
	if _, _, ok := r.BasicAuth(); ok || r.Form.Get("client_secret") != "" {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "This public client does not use a client secret.")
		return
	}
	if err := h.useCase.RevokeToken(r.Context(), r.Form.Get("client_id"), r.Form.Get("token")); errors.Is(err, input_port.ErrOAuthInvalidClient) {
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "Unknown client.")
		return
	} else if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Token revocation failed.")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *OAuthHandler) Register(w http.ResponseWriter, _ *http.Request) {
	h.cors(w)
	writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "Dynamic client registration is not supported. Create a Claude Web Client ID in brainhub.")
}

func (h *OAuthHandler) Options(w http.ResponseWriter, _ *http.Request) {
	h.cors(w)
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.WriteHeader(http.StatusNoContent)
}

func (h *OAuthHandler) authorizationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, input_port.ErrOAuthInvalidClient):
		writeOAuthError(w, http.StatusBadRequest, "invalid_client", "Unknown or revoked client.")
	case errors.Is(err, input_port.ErrOAuthAccessDenied):
		writeOAuthError(w, http.StatusForbidden, "access_denied", "This Client ID belongs to another brainhub user.")
	case errors.Is(err, input_port.ErrOAuthInvalidScope):
		writeOAuthError(w, http.StatusBadRequest, "invalid_scope", "Only read and write scopes are supported; consent cannot exceed the requested scope.")
	case errors.Is(err, input_port.ErrOAuthInvalidRequest):
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "response_type, redirect_uri, resource, and PKCE S256 must match the registered client.")
	default:
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Authorization could not be completed.")
	}
}

func (h *OAuthHandler) tokenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, input_port.ErrOAuthInvalidClient):
		writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "Unknown or revoked client.")
	case errors.Is(err, input_port.ErrOAuthInvalidGrant):
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "The code or refresh token is invalid, expired, revoked, or already used.")
	default:
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "Token issuance failed.")
	}
}

func (h *OAuthHandler) cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (h *OAuthHandler) ProtectedResourceMetadataURL() string {
	return h.issuer + ".well-known/oauth-protected-resource/mcp"
}

func authorizationInput(value func(string) string) input_port.OAuthAuthorizationRequest {
	return input_port.OAuthAuthorizationRequest{
		Scope: value("scope"), GrantedScope: value("granted_scope"), ClientID: value("client_id"), RedirectURI: value("redirect_uri"), ResponseType: value("response_type"),
		CodeChallenge: value("code_challenge"), CodeChallengeMethod: value("code_challenge_method"), Resource: value("resource"),
	}
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}
