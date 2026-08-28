package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
	UserContextKey   contextKey = "user"
)

func AuthMiddleware(authService inbound.AuthService, userService inbound.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Verifica token no Cognito
			cognitoID, err := authService.VerifyToken(r.Context(), token)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Busca usuário
			user, err := userService.GetUserByCognitoID(r.Context(), cognitoID)
			if err != nil {
				http.Error(w, "user not found", http.StatusUnauthorized)
				return
			}

			// Adiciona userID ao contexto
			ctx := context.WithValue(r.Context(), UserIDContextKey, user.ID)
			ctx = context.WithValue(ctx, UserContextKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
