package middleware

import (
	"context"
	"errors"
	"net/http"

	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

const SessionCookieName = "brainhub_session"

type authContextKey struct{}

type AuthContext struct {
	User    entity.User
	Session entity.Session
}

func RequireSession(useCase input_port.AuthUseCase, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		user, session, err := useCase.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, AuthContext{User: user, Session: session})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func OptionalSession(useCase input_port.AuthUseCase, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if errors.Is(err, http.ErrNoCookie) {
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			http.Error(w, "invalid session cookie", http.StatusBadRequest)
			return
		}
		user, session, err := useCase.Authenticate(r.Context(), cookie.Value)
		if errors.Is(err, input_port.ErrUnauthorized) {
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			http.Error(w, "failed to authenticate session", http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, AuthContext{User: user, Session: session})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Current(r *http.Request) (AuthContext, bool) {
	value, ok := r.Context().Value(authContextKey{}).(AuthContext)
	return value, ok
}
