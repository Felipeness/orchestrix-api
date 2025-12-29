package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// AlertRepository implements port.AlertRepository
type AlertRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// FindByID finds an alert by ID
func (r *AlertRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	row, err := r.queries.GetAlert(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAlertNotFound
		}
		return nil, err
	}
	return r.toDomain(row), nil
}

// FindByTenant finds alerts by tenant with pagination
func (r *AlertRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Alert, error) {
	rows, err := r.queries.ListAlerts(ctx, db.ListAlertsParams{
		TenantID: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	alerts := make([]*domain.Alert, len(rows))
	for i, row := range rows {
		alerts[i] = r.toDomain(row)
	}
	return alerts, nil
}

// CountByTenant counts alerts for a tenant
func (r *AlertRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.queries.CountAlerts(ctx, tenantID)
}

// Save saves a new alert
func (r *AlertRepository) Save(ctx context.Context, alert *domain.Alert) error {
	_, err := r.queries.CreateAlert(ctx, db.CreateAlertParams{
		TenantID:          alert.TenantID,
		Title:             alert.Title,
		Message:           alert.Message,
		Severity:          string(alert.Severity),
		Source:            alert.Source,
		Metadata:          alert.Metadata,
		TriggeredByRuleID: uuidToPgtype(alert.TriggeredByRuleID),
	})
	return err
}

// Update updates an existing alert (acknowledge or resolve)
func (r *AlertRepository) Update(ctx context.Context, alert *domain.Alert) error {
	// Handle acknowledge
	if alert.Status == domain.AlertStatusAcknowledged && alert.AcknowledgedBy != nil {
		_, err := r.queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
			ID:             alert.ID,
			AcknowledgedBy: uuidToPgtype(alert.AcknowledgedBy),
		})
		return err
	}

	// Handle resolve
	if alert.Status == domain.AlertStatusResolved && alert.ResolvedBy != nil {
		_, err := r.queries.ResolveAlert(ctx, db.ResolveAlertParams{
			ID:         alert.ID,
			ResolvedBy: uuidToPgtype(alert.ResolvedBy),
		})
		return err
	}

	return nil
}

// toDomain converts a db.Alert to domain.Alert
func (r *AlertRepository) toDomain(row db.Alert) *domain.Alert {
	var workflowID, executionID, acknowledgedBy, resolvedBy, triggeredByRuleID, triggeredWorkflowExecutionID *uuid.UUID

	if row.WorkflowID.Valid {
		id := uuid.UUID(row.WorkflowID.Bytes)
		workflowID = &id
	}
	if row.ExecutionID.Valid {
		id := uuid.UUID(row.ExecutionID.Bytes)
		executionID = &id
	}
	if row.AcknowledgedBy.Valid {
		id := uuid.UUID(row.AcknowledgedBy.Bytes)
		acknowledgedBy = &id
	}
	if row.ResolvedBy.Valid {
		id := uuid.UUID(row.ResolvedBy.Bytes)
		resolvedBy = &id
	}
	if row.TriggeredByRuleID.Valid {
		id := uuid.UUID(row.TriggeredByRuleID.Bytes)
		triggeredByRuleID = &id
	}
	if row.TriggeredWorkflowExecutionID.Valid {
		id := uuid.UUID(row.TriggeredWorkflowExecutionID.Bytes)
		triggeredWorkflowExecutionID = &id
	}

	var acknowledgedAt, resolvedAt *time.Time
	if row.AcknowledgedAt.Valid {
		t := row.AcknowledgedAt.Time
		acknowledgedAt = &t
	}
	if row.ResolvedAt.Valid {
		t := row.ResolvedAt.Time
		resolvedAt = &t
	}

	return &domain.Alert{
		ID:                           row.ID,
		TenantID:                     row.TenantID,
		WorkflowID:                   workflowID,
		ExecutionID:                  executionID,
		Severity:                     domain.AlertSeverity(row.Severity),
		Title:                        row.Title,
		Message:                      row.Message,
		Status:                       domain.AlertStatus(row.Status),
		AcknowledgedAt:               acknowledgedAt,
		AcknowledgedBy:               acknowledgedBy,
		ResolvedAt:                   resolvedAt,
		ResolvedBy:                   resolvedBy,
		CreatedAt:                    row.CreatedAt,
		TriggeredByRuleID:            triggeredByRuleID,
		TriggeredWorkflowExecutionID: triggeredWorkflowExecutionID,
		Source:                       row.Source,
		Metadata:                     row.Metadata,
	}
}

// timeToPgtype converts *time.Time to pgtype.Timestamptz
func timeToPgtype(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
