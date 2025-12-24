package tenant

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const tenantContextKey contextKey = "tenant"

// Context holds tenant information
type Context struct {
	ID   uuid.UUID
	Name string
	Plan string
}

// Middleware extracts tenant from request and adds to context
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant from JWT claims or header
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			http.Error(w, "tenant ID required", http.StatusUnauthorized)
			return
		}

		id, err := uuid.Parse(tenantID)
		if err != nil {
			http.Error(w, "invalid tenant ID", http.StatusBadRequest)
			return
		}

		ctx := &Context{
			ID:   id,
			Name: "default", // Would be loaded from DB
			Plan: "pro",     // Would be loaded from DB
		}

		r = r.WithContext(context.WithValue(r.Context(), tenantContextKey, ctx))
		next.ServeHTTP(w, r)
	})
}

// FromContext extracts tenant context from request context
func FromContext(ctx context.Context) *Context {
	if tc, ok := ctx.Value(tenantContextKey).(*Context); ok {
		return tc
	}
	return nil
}
