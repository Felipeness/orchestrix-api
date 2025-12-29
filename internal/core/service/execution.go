package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// ExecutionService implements port.ExecutionService
type ExecutionService struct {
	executionRepo port.ExecutionRepository
	executor      port.WorkflowExecutor
	tenantSetter  port.TenantContextSetter
}

// NewExecutionService creates a new execution service
func NewExecutionService(
	executionRepo port.ExecutionRepository,
	executor port.WorkflowExecutor,
	tenantSetter port.TenantContextSetter,
) *ExecutionService {
	return &ExecutionService{
		executionRepo: executionRepo,
		executor:      executor,
		tenantSetter:  tenantSetter,
	}
}

// List returns paginated executions for a tenant
func (s *ExecutionService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.ExecutionListResult, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	executions, err := s.executionRepo.FindByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.executionRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &port.ExecutionListResult{
		Executions: executions,
		Total:      total,
		Page:       page,
		Limit:      limit,
	}, nil
}

// ListByWorkflow returns paginated executions for a workflow
func (s *ExecutionService) ListByWorkflow(ctx context.Context, workflowID uuid.UUID, page, limit int) (*port.ExecutionListResult, error) {
	offset := (page - 1) * limit

	executions, err := s.executionRepo.FindByWorkflow(ctx, workflowID, limit, offset)
	if err != nil {
		return nil, err
	}

	return &port.ExecutionListResult{
		Executions: executions,
		Total:      int64(len(executions)),
		Page:       page,
		Limit:      limit,
	}, nil
}

// GetByID returns an execution by ID
func (s *ExecutionService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error) {
	return s.executionRepo.FindByID(ctx, id)
}

// Cancel cancels a running execution
func (s *ExecutionService) Cancel(ctx context.Context, id uuid.UUID) error {
	execution, err := s.executionRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if !execution.CanCancel() {
		return domain.ErrExecutionCannotCancel
	}

	// Cancel in Temporal
	if execution.TemporalWorkflowID != nil {
		if err := s.executor.Cancel(ctx, *execution.TemporalWorkflowID); err != nil {
			return err
		}
	}

	execution.MarkAsCancelled()
	return s.executionRepo.Update(ctx, execution)
}
