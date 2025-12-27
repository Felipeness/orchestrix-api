// Package api provides the HTTP API handlers implementing the generated oas.Handler interface.
package api

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/go-faster/jx"
	"github.com/orchestrix/orchestrix-api/internal/api/oas"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// Server implements the oas.Handler interface.
// Embed UnimplementedHandler to get default "not implemented" responses for unimplemented methods.
type Server struct {
	oas.UnimplementedHandler
	queries *db.Queries
	logger  *slog.Logger
}

// NewServer creates a new API server.
func NewServer(queries *db.Queries, logger *slog.Logger) *Server {
	return &Server{
		queries: queries,
		logger:  logger,
	}
}

// Ensure Server implements oas.Handler at compile time.
var _ oas.Handler = (*Server)(nil)

// =============================================================================
// HEALTH HANDLERS
// =============================================================================

// HealthCheck implements oas.Handler.
func (s *Server) HealthCheck(ctx context.Context) (oas.HealthCheckRes, error) {
	result := oas.HealthCheckOK{
		Status: oas.HealthStatusStatusHealthy,
	}
	return &result, nil
}

// LivenessProbe implements oas.Handler.
func (s *Server) LivenessProbe(ctx context.Context) (oas.LivenessProbeRes, error) {
	result := oas.LivenessProbeOK{
		Status: oas.HealthStatusStatusHealthy,
	}
	return &result, nil
}

// ReadinessProbe implements oas.Handler.
func (s *Server) ReadinessProbe(ctx context.Context) (oas.ReadinessProbeRes, error) {
	// For readiness, we return healthy since we don't have a Ping method
	// TODO: Add database health check when Ping is available
	result := oas.ReadinessProbeOK{
		Status: oas.HealthStatusStatusHealthy,
	}
	return &result, nil
}

// =============================================================================
// WORKFLOW HANDLERS
// =============================================================================

// ListWorkflows implements oas.Handler.
func (s *Server) ListWorkflows(ctx context.Context, params oas.ListWorkflowsParams) (oas.ListWorkflowsRes, error) {
	// TODO: Get tenant ID from auth context
	// For now, return empty list
	return &oas.PaginatedWorkflows{
		Data:  []oas.Workflow{},
		Total: 0,
		Page:  params.Page.Or(1),
		Limit: params.Limit.Or(20),
	}, nil
}

// GetWorkflow implements oas.Handler.
func (s *Server) GetWorkflow(ctx context.Context, params oas.GetWorkflowParams) (oas.GetWorkflowRes, error) {
	workflow, err := s.queries.GetWorkflow(ctx, params.ID)
	if err != nil {
		return &oas.GetWorkflowNotFound{
			Code:    "NOT_FOUND",
			Message: "Workflow not found",
		}, nil
	}

	return &oas.WorkflowResponse{
		Data: mapWorkflowToAPI(workflow),
	}, nil
}

// DeleteWorkflow implements oas.Handler.
func (s *Server) DeleteWorkflow(ctx context.Context, params oas.DeleteWorkflowParams) (oas.DeleteWorkflowRes, error) {
	if err := s.queries.DeleteWorkflow(ctx, params.ID); err != nil {
		return &oas.DeleteWorkflowNotFound{
			Code:    "NOT_FOUND",
			Message: "Workflow not found",
		}, nil
	}

	return &oas.NoContent{}, nil
}

// =============================================================================
// EXECUTION HANDLERS
// =============================================================================

// ListExecutions implements oas.Handler.
func (s *Server) ListExecutions(ctx context.Context, params oas.ListExecutionsParams) (oas.ListExecutionsRes, error) {
	return &oas.PaginatedExecutions{
		Data:  []oas.Execution{},
		Total: 0,
		Page:  params.Page.Or(1),
		Limit: params.Limit.Or(20),
	}, nil
}

// =============================================================================
// ALERT HANDLERS
// =============================================================================

// ListAlerts implements oas.Handler.
func (s *Server) ListAlerts(ctx context.Context, params oas.ListAlertsParams) (oas.ListAlertsRes, error) {
	return &oas.PaginatedAlerts{
		Data:  []oas.Alert{},
		Total: 0,
		Page:  params.Page.Or(1),
		Limit: params.Limit.Or(20),
	}, nil
}

// =============================================================================
// AUDIT LOG HANDLERS
// =============================================================================

// ListAuditLogs implements oas.Handler.
func (s *Server) ListAuditLogs(ctx context.Context, params oas.ListAuditLogsParams) (oas.ListAuditLogsRes, error) {
	return &oas.PaginatedAuditLogs{
		Data:  []oas.AuditLog{},
		Total: 0,
		Page:  params.Page.Or(1),
		Limit: params.Limit.Or(20),
	}, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// mapWorkflowToAPI converts a database workflow to an API workflow.
func mapWorkflowToAPI(w db.Workflow) oas.Workflow {
	workflow := oas.Workflow{
		ID:         w.ID,
		TenantID:   w.TenantID,
		Name:       w.Name,
		Definition: rawMessageToDefinition(w.Definition),
		Status:     oas.WorkflowStatus(w.Status),
		Version:    int(w.Version),
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
	}

	if w.Description != nil {
		workflow.Description = oas.NewOptString(*w.Description)
	}
	if w.Schedule != nil {
		workflow.Schedule = oas.NewOptString(*w.Schedule)
	}
	if w.CreatedBy.Valid {
		workflow.CreatedBy = oas.NewOptUUID(w.CreatedBy.Bytes)
	}

	return workflow
}

// rawMessageToDefinition converts json.RawMessage to WorkflowDefinition.
func rawMessageToDefinition(data json.RawMessage) oas.WorkflowDefinition {
	if data == nil {
		return oas.WorkflowDefinition{}
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return oas.WorkflowDefinition{}
	}

	result := make(oas.WorkflowDefinition)
	for k, v := range m {
		result[k] = jx.Raw(v)
	}
	return result
}
