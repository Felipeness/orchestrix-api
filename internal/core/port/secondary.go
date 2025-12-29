package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
)

// ============================================================================
// SECONDARY PORTS (Driven)
// These interfaces define what the application NEEDS from the outside world.
// They are IMPLEMENTED by adapters (postgres, temporal, etc.)
// ============================================================================

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Workflow, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	Save(ctx context.Context, workflow *domain.Workflow) error
	Update(ctx context.Context, workflow *domain.Workflow) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Execution, error)
	FindByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*domain.Execution, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	Save(ctx context.Context, execution *domain.Execution) error
	Update(ctx context.Context, execution *domain.Execution) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error
	UpdateTemporalIDs(ctx context.Context, id uuid.UUID, temporalWorkflowID, temporalRunID string) error
}

// AlertRepository defines the interface for alert persistence
type AlertRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Alert, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	Save(ctx context.Context, alert *domain.Alert) error
	Update(ctx context.Context, alert *domain.Alert) error
}

// AlertRuleRepository defines the interface for alert rule persistence
type AlertRuleRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.AlertRule, error)
	FindEnabledByTenant(ctx context.Context, tenantID uuid.UUID) ([]*domain.AlertRule, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	Save(ctx context.Context, rule *domain.AlertRule) error
	Update(ctx context.Context, rule *domain.AlertRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastTriggered(ctx context.Context, id uuid.UUID) error
}

// AuditRepository defines the interface for audit log persistence
type AuditRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	Save(ctx context.Context, log *domain.AuditLog) error
}

// WorkflowExecutor defines the interface for executing workflows via Temporal
type WorkflowExecutor interface {
	Execute(ctx context.Context, workflow *domain.Workflow, input map[string]interface{}) (*ExecuteResult, error)
	Cancel(ctx context.Context, temporalWorkflowID string) error
	GetStatus(ctx context.Context, temporalWorkflowID string) (string, error)
}

// ExecuteResult represents the result of starting a workflow execution
type ExecuteResult struct {
	TemporalWorkflowID string
	TemporalRunID      string
}

// Notifier defines the interface for sending notifications
type Notifier interface {
	SendSlack(ctx context.Context, channel, message string) error
	SendEmail(ctx context.Context, to, subject, body string) error
}

// TenantContextSetter defines the interface for setting tenant context (RLS)
type TenantContextSetter interface {
	SetTenantContext(ctx context.Context, tenantID uuid.UUID) error
}
