package router

import (
	"net/http"

	"brainhub/api/api/handler"
	"brainhub/api/api/middleware"
	"brainhub/usecase/input_port"
)

func New(brainUseCase input_port.BrainUseCase, pageUseCase input_port.PageUseCase, authUseCase input_port.AuthUseCase, accessUseCase input_port.AccessUseCase, publicMCPURL string, gbrainProxy http.Handler, secureCookie bool) http.Handler {
	mux := http.NewServeMux()
	brainHandler := handler.NewBrainHandler(brainUseCase)
	pageHandler := handler.NewPageHandler(pageUseCase)
	authHandler := handler.NewAuthHandler(authUseCase, secureCookie)
	accessHandler := handler.NewAccessHandler(accessUseCase)

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /api/config", handler.Config(publicMCPURL))
	mux.Handle("GET /api/brains", middleware.OptionalSession(authUseCase, http.HandlerFunc(brainHandler.List)))
	mux.Handle("POST /api/brains", middleware.RequireSession(authUseCase, http.HandlerFunc(brainHandler.Create)))
	mux.Handle("POST /api/brains/{sourceID}/adopt", middleware.RequireSession(authUseCase, http.HandlerFunc(brainHandler.Adopt)))
	mux.Handle("GET /api/brains/{sourceID}", middleware.OptionalSession(authUseCase, http.HandlerFunc(brainHandler.Get)))
	mux.Handle("GET /api/brains/{sourceID}/pages", middleware.OptionalSession(authUseCase, http.HandlerFunc(pageHandler.List)))
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
	mux.Handle("/mcp", gbrainProxy)
	mux.Handle("/mcp/", gbrainProxy)
	mux.Handle("/.well-known/", gbrainProxy)
	mux.Handle("/authorize", gbrainProxy)
	mux.Handle("/token", gbrainProxy)
	mux.Handle("/revoke", gbrainProxy)
	mux.Handle("/register", gbrainProxy)

	return mux
}
