package router

import (
	"fmt"
	"net/http"

	"brainhub/api/handler"
	"brainhub/api/middleware"
	"brainhub/usecase/input_port"
)

func New(brainUseCase input_port.BrainUseCase, pageUseCase input_port.PageUseCase, authUseCase input_port.AuthUseCase, accessUseCase input_port.AccessUseCase, mcpUseCase input_port.MCPUseCase, publicMCPURL, publicWebURL string, gbrainProxy http.Handler, secureCookie bool) (http.Handler, error) {
	mux := http.NewServeMux()
	brainHandler := handler.NewBrainHandler(brainUseCase)
	pageHandler := handler.NewPageHandler(pageUseCase)
	authHandler := handler.NewAuthHandler(authUseCase, secureCookie)
	accessHandler := handler.NewAccessHandler(accessUseCase)
	oauthHandler, err := handler.NewOAuthHandler(mcpUseCase, publicMCPURL, publicWebURL)
	if err != nil {
		return nil, fmt.Errorf("initialize OAuth handler: %w", err)
	}
	mcpHandler := handler.NewMCPHandler(mcpUseCase, gbrainProxy, oauthHandler.ProtectedResourceMetadataURL())

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /api/config", handler.Config(publicMCPURL, publicWebURL))

	mux.Handle("GET /api/brains", middleware.OptionalSession(authUseCase, http.HandlerFunc(brainHandler.List)))
	mux.Handle("POST /api/brains", middleware.RequireSession(authUseCase, http.HandlerFunc(brainHandler.Create)))
	mux.Handle("POST /api/brains/{sourceID}/adopt", middleware.RequireSession(authUseCase, http.HandlerFunc(brainHandler.Adopt)))
	mux.Handle("POST /api/brains/{sourceID}/writer/reissue", middleware.RequireSession(authUseCase, http.HandlerFunc(brainHandler.ReissueWriter)))
	mux.Handle("DELETE /api/brains/{sourceID}", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.ArchiveBrain)))
	mux.Handle("GET /api/brains/{sourceID}", middleware.OptionalSession(authUseCase, http.HandlerFunc(brainHandler.Get)))
	mux.Handle("GET /api/brains/{sourceID}/pages", middleware.OptionalSession(authUseCase, http.HandlerFunc(pageHandler.List)))
	mux.Handle("POST /api/brains/{sourceID}/pages", middleware.RequireSession(authUseCase, http.HandlerFunc(pageHandler.Create)))
	mux.Handle("GET /api/brains/{sourceID}/page-types", middleware.RequireSession(authUseCase, http.HandlerFunc(pageHandler.ListTypes)))
	mux.Handle("PUT /api/brains/{sourceID}/pages/{slug...}", middleware.RequireSession(authUseCase, http.HandlerFunc(pageHandler.Update)))
	mux.Handle("GET /api/brains/{sourceID}/pages/{slug...}", middleware.OptionalSession(authUseCase, http.HandlerFunc(pageHandler.Get)))
	mux.Handle("POST /api/brains/{sourceID}/invitations", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.CreateInvitation)))
	mux.Handle("GET /api/brains/{sourceID}/invitations", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.ListInvitations)))
	mux.Handle("POST /api/brains/{sourceID}/clients", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.IssueClient)))
	mux.Handle("GET /api/brains/{sourceID}/clients", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.ListClients)))
	mux.Handle("DELETE /api/brains/{sourceID}/members/{userID}", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.RevokeMembership)))

	mux.Handle("DELETE /api/invitations/{id}", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.RevokeInvitation)))
	mux.HandleFunc("GET /api/invitations/{token}", accessHandler.PreviewInvitation)
	mux.Handle("POST /api/invitations/{token}/accept", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.AcceptInvitation)))

	mux.Handle("DELETE /api/clients/{id}", middleware.RequireSession(authUseCase, http.HandlerFunc(accessHandler.RevokeClient)))

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.Handle("POST /api/auth/logout", middleware.RequireSession(authUseCase, http.HandlerFunc(authHandler.Logout)))

	mux.Handle("GET /api/me", middleware.RequireSession(authUseCase, http.HandlerFunc(authHandler.Me)))
	mux.Handle("GET /api/mcp/connection", middleware.RequireSession(authUseCase, http.HandlerFunc(mcpHandler.Connection)))
	mux.Handle("POST /api/mcp/client", middleware.RequireSession(authUseCase, http.HandlerFunc(mcpHandler.IssueClient)))
	mux.Handle("POST /api/mcp/tokens", middleware.RequireSession(authUseCase, http.HandlerFunc(mcpHandler.IssueCLIToken)))
	mux.Handle("DELETE /api/mcp/tokens/{id}", middleware.RequireSession(authUseCase, http.HandlerFunc(mcpHandler.RevokeCLIToken)))
	mux.Handle("POST /api/mcp/reader/reissue", middleware.RequireSession(authUseCase, http.HandlerFunc(mcpHandler.ReissueReader)))
	mux.Handle("GET /api/oauth/authorization", middleware.RequireSession(authUseCase, http.HandlerFunc(oauthHandler.Consent)))
	mux.Handle("POST /api/oauth/authorization", middleware.RequireSession(authUseCase, http.HandlerFunc(oauthHandler.Decide)))

	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/mcp/", mcpHandler)

	mux.HandleFunc("GET /.well-known/oauth-authorization-server", oauthHandler.AuthorizationServerMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", oauthHandler.ProtectedResourceMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", oauthHandler.ProtectedResourceMetadata)
	mux.Handle("GET /authorize", middleware.OptionalSession(authUseCase, http.HandlerFunc(oauthHandler.Authorize)))
	mux.HandleFunc("POST /token", oauthHandler.Token)
	mux.HandleFunc("POST /revoke", oauthHandler.Revoke)
	mux.HandleFunc("POST /register", oauthHandler.Register)
	mux.HandleFunc("OPTIONS /token", oauthHandler.Options)
	mux.HandleFunc("OPTIONS /revoke", oauthHandler.Options)
	mux.HandleFunc("OPTIONS /register", oauthHandler.Options)

	return mux, nil
}
