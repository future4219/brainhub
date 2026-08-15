package router

import (
	"net/http"

	"brainhub/api/api/handler"
	"brainhub/api/api/middleware"
	"brainhub/usecase/input_port"
)

func New(brainUseCase input_port.BrainUseCase, pageUseCase input_port.PageUseCase, authUseCase input_port.AuthUseCase, publicMCPURL string, gbrainProxy http.Handler, secureCookie bool) http.Handler {
	mux := http.NewServeMux()
	brainHandler := handler.NewBrainHandler(brainUseCase)
	pageHandler := handler.NewPageHandler(pageUseCase)
	authHandler := handler.NewAuthHandler(authUseCase, secureCookie)

	mux.HandleFunc("GET /healthz", handler.Health)
	mux.HandleFunc("GET /api/config", handler.Config(publicMCPURL))
	mux.HandleFunc("GET /api/brains", brainHandler.List)
	mux.HandleFunc("GET /api/brains/{sourceID}/pages", pageHandler.List)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.Handle("POST /api/auth/logout", middleware.RequireSession(authUseCase, http.HandlerFunc(authHandler.Logout)))
	mux.Handle("GET /api/me", middleware.RequireSession(authUseCase, http.HandlerFunc(authHandler.Me)))
	mux.Handle("/mcp", gbrainProxy)
	mux.Handle("/mcp/", gbrainProxy)
	mux.Handle("/.well-known/", gbrainProxy)

	return mux
}
