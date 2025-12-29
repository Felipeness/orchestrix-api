package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// AuditRepository implements port.AuditRepository
type AuditRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// FindByID finds an audit log by ID
// Note: The sqlc queries don't have a GetAuditLog method, so we implement it manually
func (r *AuditRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	// We need to get by tenant, so this is a simplified implementation
	// In practice, we'd need to add a GetAuditLog query to sqlc
	return nil, errors.New("GetAuditLog not implemented in sqlc - use ListAuditLogs instead")
}

// FindByTenant finds audit logs by tenant with pagination
func (r *AuditRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	rows, err := r.queries.ListAuditLogs(ctx, db.ListAuditLogsParams{
		TenantID: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []*domain.AuditLog{}, nil
		}
		return nil, err
	}

	logs := make([]*domain.AuditLog, len(rows))
	for i, row := range rows {
		logs[i] = r.toDomain(row)
	}
	return logs, nil
}

// CountByTenant counts audit logs for a tenant
func (r *AuditRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.queries.CountAuditLogsByTenant(ctx, tenantID)
}

// Save saves a new audit log
func (r *AuditRepository) Save(ctx context.Context, log *domain.AuditLog) error {
	_, err := r.queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		TenantID:     log.TenantID,
		UserID:       uuidToPgtype(log.UserID),
		EventType:    log.EventType,
		ResourceType: log.ResourceType,
		ResourceID:   uuidToPgtype(log.ResourceID),
		Action:       log.Action,
		OldValue:     log.OldValue,
		NewValue:     log.NewValue,
		IpAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
	})
	return err
}

// toDomain converts a db.AuditLog to domain.AuditLog
func (r *AuditRepository) toDomain(row db.AuditLog) *domain.AuditLog {
	var userID, resourceID *uuid.UUID

	if row.UserID.Valid {
		id := uuid.UUID(row.UserID.Bytes)
		userID = &id
	}
	if row.ResourceID.Valid {
		id := uuid.UUID(row.ResourceID.Bytes)
		resourceID = &id
	}

	return &domain.AuditLog{
		ID:           row.ID,
		TenantID:     row.TenantID,
		UserID:       userID,
		EventType:    row.EventType,
		ResourceType: row.ResourceType,
		ResourceID:   resourceID,
		Action:       row.Action,
		OldValue:     row.OldValue,
		NewValue:     row.NewValue,
		IPAddress:    row.IpAddress,
		UserAgent:    row.UserAgent,
		CreatedAt:    row.CreatedAt,
	}
}
