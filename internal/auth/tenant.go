package auth

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantMiddleware sets the tenant context for RLS in PostgreSQL
type TenantMiddleware struct {
	pool *pgxpool.Pool
}

// NewTenantMiddleware creates a new tenant middleware
func NewTenantMiddleware(pool *pgxpool.Pool) *TenantMiddleware {
	return &TenantMiddleware{pool: pool}
}

// Handler returns the HTTP middleware handler that sets tenant context for RLS
func (m *TenantMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := FromContext(r.Context())
		if user == nil {
			// No user in context, skip tenant setting
			next.ServeHTTP(w, r)
			return
		}

		tenantID := user.TenantID
		if tenantID.String() == "00000000-0000-0000-0000-000000000000" {
			// No tenant ID, skip
			next.ServeHTTP(w, r)
			return
		}

		// Set tenant context for RLS
		// Note: This sets the session variable that RLS policies use
		_, err := m.pool.Exec(r.Context(),
			"SELECT set_config('app.current_tenant_id', $1, true)",
			tenantID.String())
		if err != nil {
			http.Error(w, "failed to set tenant context", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
