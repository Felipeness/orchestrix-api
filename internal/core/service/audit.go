package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// AuditService implements port.AuditService
type AuditService struct {
	auditRepo    port.AuditRepository
	tenantSetter port.TenantContextSetter
}

// NewAuditService creates a new audit service
func NewAuditService(
	auditRepo port.AuditRepository,
	tenantSetter port.TenantContextSetter,
) *AuditService {
	return &AuditService{
		auditRepo:    auditRepo,
		tenantSetter: tenantSetter,
	}
}

// List returns paginated audit logs for a tenant
func (s *AuditService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.AuditListResult, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	logs, err := s.auditRepo.FindByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.auditRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &port.AuditListResult{
		Logs:  logs,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetByID returns an audit log by ID
func (s *AuditService) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	return s.auditRepo.FindByID(ctx, id)
}

// Log saves an audit log entry
func (s *AuditService) Log(ctx context.Context, log *domain.AuditLog) error {
	return s.auditRepo.Save(ctx, log)
}
