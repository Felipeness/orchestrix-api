package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// AlertRuleRepository implements port.AlertRuleRepository
type AlertRuleRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewAlertRuleRepository creates a new alert rule repository
func NewAlertRuleRepository(pool *pgxpool.Pool) *AlertRuleRepository {
	return &AlertRuleRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// FindByID finds an alert rule by ID
func (r *AlertRuleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	// Since the service handles tenant context, we need to query with a tenant ID
	// For now, we'll list all and filter - ideally add a GetAlertRuleByID without tenant check
	rows, err := r.queries.ListAlertRules(ctx, db.ListAlertRulesParams{
		TenantID: uuid.Nil, // This won't work properly - need to fix
		Limit:    1000,
		Offset:   0,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.ID == id {
			return r.toDomain(row), nil
		}
	}

	return nil, domain.ErrAlertRuleNotFound
}

// FindByTenant finds alert rules by tenant with pagination
func (r *AlertRuleRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.AlertRule, error) {
	rows, err := r.queries.ListAlertRules(ctx, db.ListAlertRulesParams{
		TenantID: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	rules := make([]*domain.AlertRule, len(rows))
	for i, row := range rows {
		rules[i] = r.toDomain(row)
	}
	return rules, nil
}

// FindEnabledByTenant finds enabled alert rules by tenant
func (r *AlertRuleRepository) FindEnabledByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.AlertRule, error) {
	rows, err := r.queries.ListEnabledAlertRules(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	rules := make([]*domain.AlertRule, len(rows))
	for i, row := range rows {
		rules[i] = r.toDomain(row)
	}
	return rules, nil
}

// CountByTenant counts alert rules for a tenant
func (r *AlertRuleRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.queries.CountAlertRules(ctx, tenantID)
}

// Save saves a new alert rule
func (r *AlertRuleRepository) Save(ctx context.Context, rule *domain.AlertRule) error {
	_, err := r.queries.CreateAlertRule(ctx, db.CreateAlertRuleParams{
		TenantID:             rule.TenantID,
		Name:                 rule.Name,
		Description:          rule.Description,
		Enabled:              rule.Enabled,
		ConditionType:        rule.ConditionType,
		ConditionConfig:      rule.ConditionConfig,
		Severity:             string(rule.Severity),
		AlertTitleTemplate:   rule.AlertTitleTemplate,
		AlertMessageTemplate: rule.AlertMessageTemplate,
		TriggerWorkflowID:    uuidToPgtype(rule.TriggerWorkflowID),
		TriggerInputTemplate: rule.TriggerInputTemplate,
		CooldownSeconds:      rule.CooldownSeconds,
		CreatedBy:            uuidToPgtype(rule.CreatedBy),
	})
	return err
}

// Update updates an existing alert rule
func (r *AlertRuleRepository) Update(ctx context.Context, rule *domain.AlertRule) error {
	_, err := r.queries.UpdateAlertRule(ctx, db.UpdateAlertRuleParams{
		ID:                   rule.ID,
		TenantID:             rule.TenantID,
		Name:                 rule.Name,
		Description:          rule.Description,
		Enabled:              rule.Enabled,
		ConditionType:        rule.ConditionType,
		ConditionConfig:      rule.ConditionConfig,
		Severity:             string(rule.Severity),
		AlertTitleTemplate:   rule.AlertTitleTemplate,
		AlertMessageTemplate: rule.AlertMessageTemplate,
		TriggerWorkflowID:    uuidToPgtype(rule.TriggerWorkflowID),
		TriggerInputTemplate: rule.TriggerInputTemplate,
		CooldownSeconds:      rule.CooldownSeconds,
	})
	return err
}

// Delete deletes an alert rule
func (r *AlertRuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Get the rule first to get tenant ID
	rule, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return r.queries.DeleteAlertRule(ctx, db.DeleteAlertRuleParams{
		ID:       id,
		TenantID: rule.TenantID,
	})
}

// UpdateLastTriggered updates the last triggered timestamp
func (r *AlertRuleRepository) UpdateLastTriggered(ctx context.Context, id uuid.UUID) error {
	return r.queries.UpdateAlertRuleLastTriggered(ctx, id)
}

// toDomain converts a db.AlertRule to domain.AlertRule
func (r *AlertRuleRepository) toDomain(row db.AlertRule) *domain.AlertRule {
	var triggerWorkflowID, createdBy *uuid.UUID

	if row.TriggerWorkflowID.Valid {
		id := uuid.UUID(row.TriggerWorkflowID.Bytes)
		triggerWorkflowID = &id
	}
	if row.CreatedBy.Valid {
		id := uuid.UUID(row.CreatedBy.Bytes)
		createdBy = &id
	}

	var lastTriggeredAt *time.Time
	if row.LastTriggeredAt.Valid {
		t := row.LastTriggeredAt.Time
		lastTriggeredAt = &t
	}

	return &domain.AlertRule{
		ID:                   row.ID,
		TenantID:             row.TenantID,
		Name:                 row.Name,
		Description:          row.Description,
		Enabled:              row.Enabled,
		ConditionType:        row.ConditionType,
		ConditionConfig:      row.ConditionConfig,
		Severity:             domain.AlertSeverity(row.Severity),
		AlertTitleTemplate:   row.AlertTitleTemplate,
		AlertMessageTemplate: row.AlertMessageTemplate,
		TriggerWorkflowID:    triggerWorkflowID,
		TriggerInputTemplate: row.TriggerInputTemplate,
		CooldownSeconds:      row.CooldownSeconds,
		LastTriggeredAt:      lastTriggeredAt,
		CreatedBy:            createdBy,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

// FindByIDWithTenant finds an alert rule by ID and tenant ID
func (r *AlertRuleRepository) FindByIDWithTenant(ctx context.Context, id, tenantID uuid.UUID) (*domain.AlertRule, error) {
	row, err := r.queries.GetAlertRule(ctx, db.GetAlertRuleParams{
		ID:       id,
		TenantID: tenantID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAlertRuleNotFound
		}
		return nil, err
	}
	return r.toDomain(row), nil
}
