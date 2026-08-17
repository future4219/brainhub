package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"brainhub/api/middleware"
	"brainhub/api/schema"
	"brainhub/usecase/input_port"
)

const maxAuthBodyBytes = 64 << 10

type AuthHandler struct {
	useCase      input_port.AuthUseCase
	secureCookie bool
}

func NewAuthHandler(useCase input_port.AuthUseCase, secureCookie bool) *AuthHandler {
	return &AuthHandler{useCase: useCase, secureCookie: secureCookie}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request schema.RegisterRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, session, rawToken, err := h.useCase.Register(r.Context(), input_port.RegisterInput{
		Email: request.Email, Password: request.Password, Name: request.Name,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.setSessionCookie(w, rawToken, session.ExpiresAt)
	writeJSON(w, http.StatusCreated, schema.UserResponseFromEntity(user))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request schema.LoginRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, session, rawToken, err := h.useCase.Login(r.Context(), input_port.LoginInput{
		Email: request.Email, Password: request.Password, RemoteAddr: remoteHost(r.RemoteAddr),
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.setSessionCookie(w, rawToken, session.ExpiresAt)
	writeJSON(w, http.StatusOK, schema.UserResponseFromEntity(user))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	current, ok := middleware.Current(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if err := h.useCase.Logout(r.Context(), current.Session.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	current, ok := middleware.Current(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, schema.UserResponseFromEntity(current.User))
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, rawToken string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return false
	}
	return true
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, input_port.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid input"})
	case errors.Is(err, input_port.ErrEmailAlreadyRegistered):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
	case errors.Is(err, input_port.ErrInvalidCredentials):
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	case errors.Is(err, input_port.ErrRateLimited):
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many login attempts"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
