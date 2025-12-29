package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantContextSetter implements port.TenantContextSetter
type TenantContextSetter struct {
	pool *pgxpool.Pool
}

// NewTenantContextSetter creates a new tenant context setter
func NewTenantContextSetter(pool *pgxpool.Pool) *TenantContextSetter {
	return &TenantContextSetter{pool: pool}
}

// SetTenantContext sets the tenant context for RLS
func (s *TenantContextSetter) SetTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)",
		tenantID.String())
	return err
}
