package middleware

import (
	"context"
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

func Current(r *http.Request) (AuthContext, bool) {
	value, ok := r.Context().Value(authContextKey{}).(AuthContext)
	return value, ok
}
