package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// AlertService implements port.AlertService
type AlertService struct {
	alertRepo    port.AlertRepository
	auditService port.AuditService
	tenantSetter port.TenantContextSetter
}

// NewAlertService creates a new alert service
func NewAlertService(
	alertRepo port.AlertRepository,
	auditService port.AuditService,
	tenantSetter port.TenantContextSetter,
) *AlertService {
	return &AlertService{
		alertRepo:    alertRepo,
		auditService: auditService,
		tenantSetter: tenantSetter,
	}
}

// List returns paginated alerts for a tenant
func (s *AlertService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.AlertListResult, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	alerts, err := s.alertRepo.FindByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.alertRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &port.AlertListResult{
		Alerts: alerts,
		Total:  total,
		Page:   page,
		Limit:  limit,
	}, nil
}

// GetByID returns an alert by ID
func (s *AlertService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	return s.alertRepo.FindByID(ctx, id)
}

// Create creates a new alert
func (s *AlertService) Create(ctx context.Context, input port.CreateAlertInput) (*domain.Alert, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
		return nil, err
	}

	alert := &domain.Alert{
		ID:                uuid.New(),
		TenantID:          input.TenantID,
		WorkflowID:        input.WorkflowID,
		ExecutionID:       input.ExecutionID,
		Severity:          input.Severity,
		Title:             input.Title,
		Message:           input.Message,
		Status:            domain.AlertStatusTriggered,
		CreatedAt:         time.Now(),
		TriggeredByRuleID: input.TriggeredByRuleID,
		Source:            input.Source,
	}

	if err := s.alertRepo.Save(ctx, alert); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, input.TenantID, nil, domain.AuditEventAlertCreated, alert.ID, nil, alert)

	return alert, nil
}

// Acknowledge acknowledges an alert
func (s *AlertService) Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Alert, error) {
	alert, err := s.alertRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := alert.Acknowledge(userID); err != nil {
		return nil, err
	}

	if err := s.alertRepo.Update(ctx, alert); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, alert.TenantID, &userID, domain.AuditEventAlertAcknowledged, alert.ID, nil, alert)

	return alert, nil
}

// Resolve resolves an alert
func (s *AlertService) Resolve(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Alert, error) {
	alert, err := s.alertRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := alert.Resolve(userID); err != nil {
		return nil, err
	}

	if err := s.alertRepo.Update(ctx, alert); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, alert.TenantID, &userID, domain.AuditEventAlertResolved, alert.ID, nil, alert)

	return alert, nil
}

func (s *AlertService) logAudit(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, eventType string, resourceID uuid.UUID, oldValue, newValue interface{}) {
	if s.auditService == nil {
		return
	}

	action := domain.ActionCreate
	switch eventType {
	case domain.AuditEventAlertAcknowledged:
		action = domain.ActionAcknowledge
	case domain.AuditEventAlertResolved:
		action = domain.ActionResolve
	}

	log := domain.NewAuditLog(tenantID, userID, eventType, domain.ResourceTypeAlert, &resourceID, action).
		WithOldValue(oldValue).
		WithNewValue(newValue)

	s.auditService.Log(ctx, log)
}
