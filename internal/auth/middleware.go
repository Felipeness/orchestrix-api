package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const userContextKey contextKey = "user"

// User holds authenticated user information
type User struct {
	ID       string
	TenantID uuid.UUID
	Email    string
	Name     string
	Roles    []string
}

// KeycloakClaims represents the JWT claims from Keycloak
type KeycloakClaims struct {
	jwt.RegisteredClaims
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Name          string                 `json:"name"`
	GivenName     string                 `json:"given_name"`
	FamilyName    string                 `json:"family_name"`
	TenantID      string                 `json:"tenant_id"`
	RealmAccess   RealmAccess            `json:"realm_access"`
	ResourceAccess map[string]RoleAccess `json:"resource_access"`
}

type RealmAccess struct {
	Roles []string `json:"roles"`
}

type RoleAccess struct {
	Roles []string `json:"roles"`
}

// Config for the auth middleware
type Config struct {
	KeycloakURL   string
	Realm         string
	ClientID      string
	SkipPaths     []string // paths that don't require auth (e.g., /health)
}

// Middleware creates a new JWT validation middleware
type Middleware struct {
	config  Config
	jwks    keyfunc.Keyfunc
	jwksErr error
	once    sync.Once
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(config Config) *Middleware {
	return &Middleware{
		config: config,
	}
}

// initJWKS initializes the JWKS (JSON Web Key Set) from Keycloak
func (m *Middleware) initJWKS() error {
	m.once.Do(func() {
		jwksURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs",
			m.config.KeycloakURL, m.config.Realm)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		m.jwks, m.jwksErr = keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
		if m.jwksErr != nil {
			slog.Error("failed to create JWKS", "error", m.jwksErr, "url", jwksURL)
		} else {
			slog.Info("JWKS initialized", "url", jwksURL)
		}
	})
	return m.jwksErr
}

// Handler returns the HTTP middleware handler
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for certain paths
		for _, path := range m.config.SkipPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Initialize JWKS if needed
		if err := m.initJWKS(); err != nil {
			slog.Error("JWKS not available", "error", err)
			http.Error(w, "authentication service unavailable", http.StatusServiceUnavailable)
			return
		}

		// Extract token from header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &KeycloakClaims{}, m.jwks.Keyfunc,
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(fmt.Sprintf("%s/realms/%s", m.config.KeycloakURL, m.config.Realm)),
			jwt.WithAudience(m.config.ClientID),
		)
		if err != nil {
			slog.Debug("token validation failed", "error", err)
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*KeycloakClaims)
		if !ok || !token.Valid {
			http.Error(w, "invalid token claims", http.StatusUnauthorized)
			return
		}

		// Parse tenant ID
		var tenantID uuid.UUID
		if claims.TenantID != "" {
			tenantID, err = uuid.Parse(claims.TenantID)
			if err != nil {
				slog.Warn("invalid tenant_id in token", "tenant_id", claims.TenantID)
			}
		}

		// Build user object
		user := &User{
			ID:       claims.Subject,
			TenantID: tenantID,
			Email:    claims.Email,
			Name:     claims.Name,
			Roles:    claims.RealmAccess.Roles,
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
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

// RequireRole middleware ensures user has a specific role
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasRole(r.Context(), role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) uuid.UUID {
	user := FromContext(ctx)
	if user == nil {
		return uuid.Nil
	}
	return user.TenantID
}

// UserInfo returns JSON-serializable user info
func (u *User) UserInfo() map[string]any {
	return map[string]any{
		"id":        u.ID,
		"tenant_id": u.TenantID.String(),
		"email":     u.Email,
		"name":      u.Name,
		"roles":     u.Roles,
	}
}

// MarshalJSON implements json.Marshaler
func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.UserInfo())
}
