package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

// User holds authenticated user information
type User struct {
	ID    string
	Email string
	Roles []string
}

// Middleware validates JWT and adds user to context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// TODO: Validate JWT with Keycloak
		// For now, just check token exists
		if token == "" {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		// TODO: Parse claims from JWT
		user := &User{
			ID:    "user-123",
			Email: "user@example.com",
			Roles: []string{"user"},
		}

		r = r.WithContext(context.WithValue(r.Context(), userContextKey, user))
		next.ServeHTTP(w, r)
	})
}

// FromContext extracts user from request context
func FromContext(ctx context.Context) *User {
	if u, ok := ctx.Value(userContextKey).(*User); ok {
		return u
	}
	return nil
}

// HasRole checks if user has a specific role
func HasRole(ctx context.Context, role string) bool {
	user := FromContext(ctx)
	if user == nil {
		return false
	}
	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
}
