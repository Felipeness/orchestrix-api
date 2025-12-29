package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// WorkflowService implements port.WorkflowService
type WorkflowService struct {
	workflowRepo  port.WorkflowRepository
	executionRepo port.ExecutionRepository
	executor      port.WorkflowExecutor
	auditService  port.AuditService
	tenantSetter  port.TenantContextSetter
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(
	workflowRepo port.WorkflowRepository,
	executionRepo port.ExecutionRepository,
	executor port.WorkflowExecutor,
	auditService port.AuditService,
	tenantSetter port.TenantContextSetter,
) *WorkflowService {
	return &WorkflowService{
		workflowRepo:  workflowRepo,
		executionRepo: executionRepo,
		executor:      executor,
		auditService:  auditService,
		tenantSetter:  tenantSetter,
	}
}

// List returns paginated workflows for a tenant
func (s *WorkflowService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.WorkflowListResult, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	workflows, err := s.workflowRepo.FindByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.workflowRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &port.WorkflowListResult{
		Workflows: workflows,
		Total:     total,
		Page:      page,
		Limit:     limit,
	}, nil
}

// GetByID returns a workflow by ID
func (s *WorkflowService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	return s.workflowRepo.FindByID(ctx, id)
}

// Create creates a new workflow
func (s *WorkflowService) Create(ctx context.Context, input port.CreateWorkflowInput) (*domain.Workflow, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
		return nil, err
	}

	workflow := &domain.Workflow{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		Name:        input.Name,
		Description: input.Description,
		Definition:  input.Definition,
		Schedule:    input.Schedule,
		Status:      domain.WorkflowStatusDraft,
		Version:     1,
		CreatedBy:   input.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.workflowRepo.Save(ctx, workflow); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, input.TenantID, input.CreatedBy, domain.AuditEventWorkflowCreated, workflow.ID, nil, workflow)

	return workflow, nil
}

// Update updates an existing workflow
func (s *WorkflowService) Update(ctx context.Context, id uuid.UUID, input port.UpdateWorkflowInput) (*domain.Workflow, error) {
	workflow, err := s.workflowRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	oldWorkflow := *workflow

	if input.Name != nil {
		workflow.Name = *input.Name
	}
	if input.Description != nil {
		workflow.Description = input.Description
	}
	if input.Definition != nil {
		workflow.Definition = input.Definition
	}
	if input.Schedule != nil {
		workflow.Schedule = input.Schedule
	}
	if input.Status != nil {
		workflow.Status = *input.Status
	}
	workflow.UpdatedAt = time.Now()

	if err := s.workflowRepo.Update(ctx, workflow); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(ctx, workflow.TenantID, nil, domain.AuditEventWorkflowUpdated, workflow.ID, &oldWorkflow, workflow)

	return workflow, nil
}

// Delete deletes a workflow
func (s *WorkflowService) Delete(ctx context.Context, id uuid.UUID) error {
	workflow, err := s.workflowRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.workflowRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Log audit
	s.logAudit(ctx, workflow.TenantID, nil, domain.AuditEventWorkflowDeleted, workflow.ID, workflow, nil)

	return nil
}

// Execute starts a workflow execution
func (s *WorkflowService) Execute(ctx context.Context, id uuid.UUID, userID string, input map[string]interface{}) (*domain.Execution, error) {
	workflow, err := s.workflowRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !workflow.CanExecute() {
		return nil, domain.ErrWorkflowCannotExecute
	}

	// Create execution record
	inputJSON, _ := json.Marshal(input)
	userUUID, _ := uuid.Parse(userID)

	execution := &domain.Execution{
		ID:          uuid.New(),
		TenantID:    workflow.TenantID,
		WorkflowID:  workflow.ID,
		Status:      domain.ExecutionStatusPending,
		Input:       inputJSON,
		CreatedBy:   &userUUID,
		CreatedAt:   time.Now(),
		TriggeredBy: stringPtr("user:" + userID),
	}

	if err := s.executionRepo.Save(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// Execute via Temporal
	result, err := s.executor.Execute(ctx, workflow, input)
	if err != nil {
		errMsg := err.Error()
		execution.MarkAsFailed(errMsg)
		s.executionRepo.Update(ctx, execution)
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	// Update with Temporal IDs
	execution.TemporalWorkflowID = &result.TemporalWorkflowID
	execution.TemporalRunID = &result.TemporalRunID
	execution.MarkAsRunning()

	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to update execution: %w", err)
	}

	// Log audit
	s.logAudit(ctx, workflow.TenantID, &userUUID, domain.AuditEventWorkflowExecuted, workflow.ID, nil, execution)

	return execution, nil
}

// ListExecutions returns paginated executions for a workflow
func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID uuid.UUID, page, limit int) (*port.ExecutionListResult, error) {
	offset := (page - 1) * limit

	executions, err := s.executionRepo.FindByWorkflow(ctx, workflowID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Count total executions for this workflow
	total := int64(len(executions))

	return &port.ExecutionListResult{
		Executions: executions,
		Total:      total,
		Page:       page,
		Limit:      limit,
	}, nil
}

func (s *WorkflowService) logAudit(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, eventType string, resourceID uuid.UUID, oldValue, newValue interface{}) {
	if s.auditService == nil {
		return
	}

	log := domain.NewAuditLog(tenantID, userID, eventType, domain.ResourceTypeWorkflow, &resourceID, domain.ActionCreate).
		WithOldValue(oldValue).
		WithNewValue(newValue)

	s.auditService.Log(ctx, log)
}

func stringPtr(s string) *string {
	return &s
}
