package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
)

// ============================================================================
// PRIMARY PORTS (Driving)
// These interfaces define what the application OFFERS to the outside world.
// They are IMPLEMENTED by the core services.
// They are CALLED by adapters (http handlers, cli, tests, etc.)
// ============================================================================

// WorkflowService defines the primary port for workflow operations
type WorkflowService interface {
	List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*WorkflowListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error)
	Create(ctx context.Context, input CreateWorkflowInput) (*domain.Workflow, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateWorkflowInput) (*domain.Workflow, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Execute(ctx context.Context, id uuid.UUID, userID string, input map[string]interface{}) (*domain.Execution, error)
	ListExecutions(ctx context.Context, workflowID uuid.UUID, page, limit int) (*ExecutionListResult, error)
}

// ExecutionService defines the primary port for execution operations
type ExecutionService interface {
	List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*ExecutionListResult, error)
	ListByWorkflow(ctx context.Context, workflowID uuid.UUID, page, limit int) (*ExecutionListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error)
	Cancel(ctx context.Context, id uuid.UUID) error
}

// AlertService defines the primary port for alert operations
type AlertService interface {
	List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*AlertListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error)
	Create(ctx context.Context, input CreateAlertInput) (*domain.Alert, error)
	Acknowledge(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Alert, error)
	Resolve(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Alert, error)
}

// AlertRuleService defines the primary port for alert rule operations
type AlertRuleService interface {
	List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*AlertRuleListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error)
	Create(ctx context.Context, input CreateAlertRuleInput) (*domain.AlertRule, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateAlertRuleInput) (*domain.AlertRule, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Evaluate(ctx context.Context, tenantID uuid.UUID, metricName string, value float64) error
}

// AuditService defines the primary port for audit log operations
type AuditService interface {
	List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*AuditListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error)
	Log(ctx context.Context, log *domain.AuditLog) error
}

// MetricService defines the primary port for metrics operations
type MetricService interface {
	// Ingestion
	Ingest(ctx context.Context, input IngestMetricInput) error
	IngestBatch(ctx context.Context, input IngestMetricBatchInput) (*IngestBatchResult, error)

	// Queries
	Query(ctx context.Context, query domain.MetricQuery) (*MetricQueryResult, error)
	GetLatest(ctx context.Context, tenantID uuid.UUID, name string, labels map[string]string) (*domain.Metric, error)
	GetAggregate(ctx context.Context, query domain.MetricQuery) (*domain.MetricAggregate, error)
	GetSeries(ctx context.Context, query domain.MetricQuery, bucketSize time.Duration) ([]*domain.TimeBucket, error)
	ListNames(ctx context.Context, tenantID uuid.UUID, prefix string) ([]string, error)

	// Metric Definitions (metadata)
	ListDefinitions(ctx context.Context, tenantID uuid.UUID, page, limit int) (*MetricDefinitionListResult, error)
	GetDefinition(ctx context.Context, tenantID uuid.UUID, name string) (*domain.MetricDefinition, error)
	CreateDefinition(ctx context.Context, input CreateMetricDefinitionInput) (*domain.MetricDefinition, error)
	UpdateDefinition(ctx context.Context, tenantID uuid.UUID, name string, input UpdateMetricDefinitionInput) (*domain.MetricDefinition, error)
	DeleteDefinition(ctx context.Context, tenantID uuid.UUID, name string) error
}

// ============================================================================
// DTOs - Data Transfer Objects for Primary Ports
// ============================================================================

// Workflow DTOs

type CreateWorkflowInput struct {
	TenantID    uuid.UUID
	Name        string
	Description *string
	Definition  []byte
	Schedule    *string
	CreatedBy   *uuid.UUID
}

type UpdateWorkflowInput struct {
	Name        *string
	Description *string
	Definition  []byte
	Schedule    *string
	Status      *domain.WorkflowStatus
}

type WorkflowListResult struct {
	Workflows []*domain.Workflow
	Total     int64
	Page      int
	Limit     int
}

// Execution DTOs

type ExecutionListResult struct {
	Executions []*domain.Execution
	Total      int64
	Page       int
	Limit      int
}

// Alert DTOs

type CreateAlertInput struct {
	TenantID          uuid.UUID
	WorkflowID        *uuid.UUID
	ExecutionID       *uuid.UUID
	Severity          domain.AlertSeverity
	Title             string
	Message           *string
	Source            *string
	TriggeredByRuleID *uuid.UUID
}

type AlertListResult struct {
	Alerts []*domain.Alert
	Total  int64
	Page   int
	Limit  int
}

// AlertRule DTOs

type CreateAlertRuleInput struct {
	TenantID             uuid.UUID
	Name                 string
	Description          *string
	ConditionType        string
	ConditionConfig      []byte
	Severity             domain.AlertSeverity
	AlertTitleTemplate   string
	AlertMessageTemplate *string
	TriggerWorkflowID    *uuid.UUID
	TriggerInputTemplate []byte
	CooldownSeconds      int32
	CreatedBy            uuid.UUID
}

type UpdateAlertRuleInput struct {
	Name                 *string
	Description          *string
	Enabled              *bool
	ConditionType        *string
	ConditionConfig      []byte
	Severity             *domain.AlertSeverity
	AlertTitleTemplate   *string
	AlertMessageTemplate *string
	TriggerWorkflowID    *uuid.UUID
	TriggerInputTemplate []byte
	CooldownSeconds      *int32
}

type AlertRuleListResult struct {
	Rules []*domain.AlertRule
	Total int64
	Page  int
	Limit int
}

// Audit DTOs

type AuditListResult struct {
	Logs  []*domain.AuditLog
	Total int64
	Page  int
	Limit int
}

// Metric DTOs

type IngestMetricInput struct {
	TenantID  uuid.UUID
	Name      string
	Value     float64
	Labels    map[string]string
	Source    *string
	Timestamp *time.Time
}

type IngestMetricBatchInput struct {
	TenantID uuid.UUID
	Metrics  []IngestMetricInput
}

type IngestBatchResult struct {
	Ingested int
	Failed   int
	Errors   []string
}

type MetricQueryResult struct {
	Metrics []*domain.Metric
	Total   int64
	Page    int
	Limit   int
}

type MetricDefinitionListResult struct {
	Definitions []*domain.MetricDefinition
	Total       int64
	Page        int
	Limit       int
}

type CreateMetricDefinitionInput struct {
	TenantID       uuid.UUID
	Name           string
	DisplayName    *string
	Description    *string
	Unit           *string
	Type           domain.MetricType
	Aggregation    domain.AggregationType
	AlertThreshold *domain.AlertThreshold
	RetentionDays  int
}

type UpdateMetricDefinitionInput struct {
	DisplayName    *string
	Description    *string
	Unit           *string
	Type           *domain.MetricType
	Aggregation    *domain.AggregationType
	AlertThreshold *domain.AlertThreshold
	RetentionDays  *int
}
