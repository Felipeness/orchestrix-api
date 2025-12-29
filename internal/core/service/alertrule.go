package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// AlertRuleService implements port.AlertRuleService
type AlertRuleService struct {
	ruleRepo       port.AlertRuleRepository
	alertService   port.AlertService
	workflowService port.WorkflowService
	auditService   port.AuditService
	tenantSetter   port.TenantContextSetter
}

// NewAlertRuleService creates a new alert rule service
func NewAlertRuleService(
	ruleRepo port.AlertRuleRepository,
	alertService port.AlertService,
	workflowService port.WorkflowService,
	auditService port.AuditService,
	tenantSetter port.TenantContextSetter,
) *AlertRuleService {
	return &AlertRuleService{
		ruleRepo:       ruleRepo,
		alertService:   alertService,
		workflowService: workflowService,
		auditService:   auditService,
		tenantSetter:   tenantSetter,
	}
}

// List returns paginated alert rules for a tenant
func (s *AlertRuleService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.AlertRuleListResult, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	rules, err := s.ruleRepo.FindByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.ruleRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &port.AlertRuleListResult{
		Rules: rules,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetByID returns an alert rule by ID
func (s *AlertRuleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	return s.ruleRepo.FindByID(ctx, id)
}

// Create creates a new alert rule
func (s *AlertRuleService) Create(ctx context.Context, input port.CreateAlertRuleInput) (*domain.AlertRule, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
		return nil, err
	}

	rule := &domain.AlertRule{
		ID:                   uuid.New(),
		TenantID:             input.TenantID,
		Name:                 input.Name,
		Description:          input.Description,
		Enabled:              true,
		ConditionType:        input.ConditionType,
		ConditionConfig:      input.ConditionConfig,
		Severity:             input.Severity,
		AlertTitleTemplate:   input.AlertTitleTemplate,
		AlertMessageTemplate: input.AlertMessageTemplate,
		TriggerWorkflowID:    input.TriggerWorkflowID,
		TriggerInputTemplate: input.TriggerInputTemplate,
		CooldownSeconds:      input.CooldownSeconds,
		CreatedBy:            &input.CreatedBy,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.ruleRepo.Save(ctx, rule); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, input.TenantID, &input.CreatedBy, domain.AuditEventAlertRuleCreated, rule.ID, nil, rule)

	return rule, nil
}

// Update updates an existing alert rule
func (s *AlertRuleService) Update(ctx context.Context, id uuid.UUID, input port.UpdateAlertRuleInput) (*domain.AlertRule, error) {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldRule := *rule

	if input.Name != nil {
		rule.Name = *input.Name
	}
	if input.Description != nil {
		rule.Description = input.Description
	}
	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	}
	if input.ConditionType != nil {
		rule.ConditionType = *input.ConditionType
	}
	if input.ConditionConfig != nil {
		rule.ConditionConfig = input.ConditionConfig
	}
	if input.Severity != nil {
		rule.Severity = *input.Severity
	}
	if input.AlertTitleTemplate != nil {
		rule.AlertTitleTemplate = *input.AlertTitleTemplate
	}
	if input.AlertMessageTemplate != nil {
		rule.AlertMessageTemplate = input.AlertMessageTemplate
	}
	if input.TriggerWorkflowID != nil {
		rule.TriggerWorkflowID = input.TriggerWorkflowID
	}
	if input.TriggerInputTemplate != nil {
		rule.TriggerInputTemplate = input.TriggerInputTemplate
	}
	if input.CooldownSeconds != nil {
		rule.CooldownSeconds = *input.CooldownSeconds
	}
	rule.UpdatedAt = time.Now()

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, rule.TenantID, nil, domain.AuditEventAlertRuleUpdated, rule.ID, &oldRule, rule)

	return rule, nil
}

// Delete deletes an alert rule
func (s *AlertRuleService) Delete(ctx context.Context, id uuid.UUID) error {
	rule, err := s.ruleRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.ruleRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Log audit
	s.logAudit(ctx, rule.TenantID, nil, domain.AuditEventAlertRuleDeleted, rule.ID, rule, nil)

	return nil
}

// Evaluate evaluates all enabled rules for a metric value
func (s *AlertRuleService) Evaluate(ctx context.Context, tenantID uuid.UUID, metricName string, value float64) error {
	rules, err := s.ruleRepo.FindEnabledByTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !rule.CanTrigger() {
			continue
		}

		triggered, err := rule.EvaluateThreshold(value)
		if err != nil {
			continue
		}

		if triggered {
			// Create alert
			_, err := s.alertService.Create(ctx, port.CreateAlertInput{
				TenantID:          tenantID,
				Severity:          rule.Severity,
				Title:             rule.AlertTitleTemplate,
				Message:           rule.AlertMessageTemplate,
				TriggeredByRuleID: &rule.ID,
			})
			if err != nil {
				continue
			}

			// Update last triggered
			s.ruleRepo.UpdateLastTriggered(ctx, rule.ID)

			// Trigger workflow if configured
			if rule.TriggerWorkflowID != nil {
				s.workflowService.Execute(ctx, *rule.TriggerWorkflowID, "", nil)
			}
		}
	}

	return nil
}

func (s *AlertRuleService) logAudit(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, eventType string, resourceID uuid.UUID, oldValue, newValue interface{}) {
	if s.auditService == nil {
		return
	}

	action := domain.ActionCreate
	switch eventType {
	case domain.AuditEventAlertRuleUpdated:
		action = domain.ActionUpdate
	case domain.AuditEventAlertRuleDeleted:
		action = domain.ActionDelete
	}

	log := domain.NewAuditLog(tenantID, userID, eventType, domain.ResourceTypeAlertRule, &resourceID, action).
		WithOldValue(oldValue).
		WithNewValue(newValue)

	s.auditService.Log(ctx, log)
}
