package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
	"github.com/orchestrix/orchestrix-api/internal/core/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowService_List(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns paginated workflows", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		// Add test workflows
		for i := 0; i < 5; i++ {
			workflowRepo.AddWorkflow(&domain.Workflow{
				ID:       uuid.New(),
				TenantID: tenantID,
				Name:     "Test Workflow",
				Status:   domain.WorkflowStatusActive,
			})
		}

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.List(ctx, tenantID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, int64(5), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Limit)
		assert.Len(t, result.Workflows, 5)
		assert.True(t, tenantSetter.SetCalled)
	})

	t.Run("returns empty list when no workflows", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.List(ctx, tenantID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Len(t, result.Workflows, 0)
	})

	t.Run("returns error when tenant context fails", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()
		tenantSetter.SetErr = domain.ErrUnauthorized

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.List(ctx, tenantID, 1, 10)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrUnauthorized, err)
	})
}

func TestWorkflowService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns workflow when found", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		expected := &domain.Workflow{
			ID:       workflowID,
			TenantID: uuid.New(),
			Name:     "Test Workflow",
			Status:   domain.WorkflowStatusActive,
		}
		workflowRepo.AddWorkflow(expected)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.GetByID(ctx, workflowID)

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, expected.Name, result.Name)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.GetByID(ctx, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}

func TestWorkflowService_Create(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("creates workflow successfully", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		input := port.CreateWorkflowInput{
			TenantID:    tenantID,
			Name:        "New Workflow",
			Description: stringPtr("Test description"),
			Definition:  json.RawMessage(`{"steps":[]}`),
			CreatedBy:   &userID,
		}

		result, err := svc.Create(ctx, input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, input.Name, result.Name)
		assert.Equal(t, domain.WorkflowStatusDraft, result.Status)
		assert.Equal(t, int32(1), result.Version)
		assert.True(t, workflowRepo.SaveCalled)
		assert.True(t, auditService.LogCalled)
	})

	t.Run("returns error when save fails", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()
		workflowRepo.SaveErr = domain.ErrInternal

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		input := port.CreateWorkflowInput{
			TenantID: tenantID,
			Name:     "New Workflow",
		}

		result, err := svc.Create(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkflowService_Update(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("updates workflow successfully", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		existing := &domain.Workflow{
			ID:       workflowID,
			TenantID: tenantID,
			Name:     "Original Name",
			Status:   domain.WorkflowStatusDraft,
		}
		workflowRepo.AddWorkflow(existing)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		newName := "Updated Name"
		input := port.UpdateWorkflowInput{
			Name: &newName,
		}

		result, err := svc.Update(ctx, workflowID, input)

		require.NoError(t, err)
		assert.Equal(t, newName, result.Name)
		assert.True(t, workflowRepo.UpdateCalled)
		assert.True(t, auditService.LogCalled)
	})

	t.Run("updates status to active", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		existing := &domain.Workflow{
			ID:       workflowID,
			TenantID: tenantID,
			Name:     "Test Workflow",
			Status:   domain.WorkflowStatusDraft,
		}
		workflowRepo.AddWorkflow(existing)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		newStatus := domain.WorkflowStatusActive
		input := port.UpdateWorkflowInput{
			Status: &newStatus,
		}

		result, err := svc.Update(ctx, workflowID, input)

		require.NoError(t, err)
		assert.Equal(t, domain.WorkflowStatusActive, result.Status)
	})

	t.Run("returns error when workflow not found", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		input := port.UpdateWorkflowInput{}

		result, err := svc.Update(ctx, uuid.New(), input)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkflowService_Delete(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("deletes workflow successfully", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		existing := &domain.Workflow{
			ID:       workflowID,
			TenantID: tenantID,
			Name:     "Test Workflow",
		}
		workflowRepo.AddWorkflow(existing)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		err := svc.Delete(ctx, workflowID)

		require.NoError(t, err)
		assert.True(t, workflowRepo.DeleteCalled)
		assert.True(t, auditService.LogCalled)
	})

	t.Run("returns error when workflow not found", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		err := svc.Delete(ctx, uuid.New())

		require.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}

func TestWorkflowService_Execute(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New().String()

	t.Run("executes active workflow successfully", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		workflow := &domain.Workflow{
			ID:       workflowID,
			TenantID: tenantID,
			Name:     "Test Workflow",
			Status:   domain.WorkflowStatusActive,
		}
		workflowRepo.AddWorkflow(workflow)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		input := map[string]interface{}{"key": "value"}
		result, err := svc.Execute(ctx, workflowID, userID, input)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, domain.ExecutionStatusRunning, result.Status)
		assert.True(t, executionRepo.SaveCalled)
		assert.True(t, executor.ExecuteCalled)
		assert.True(t, auditService.LogCalled)
	})

	t.Run("returns error when workflow is not active", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		workflow := &domain.Workflow{
			ID:       workflowID,
			TenantID: tenantID,
			Name:     "Draft Workflow",
			Status:   domain.WorkflowStatusDraft,
		}
		workflowRepo.AddWorkflow(workflow)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.Execute(ctx, workflowID, userID, nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrWorkflowCannotExecute, err)
	})

	t.Run("returns error when workflow not found", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.Execute(ctx, uuid.New(), userID, nil)

		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when executor fails", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		executor.ExecuteErr = domain.ErrInternal
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		workflowID := uuid.New()
		workflow := &domain.Workflow{
			ID:       workflowID,
			TenantID: tenantID,
			Name:     "Test Workflow",
			Status:   domain.WorkflowStatusActive,
		}
		workflowRepo.AddWorkflow(workflow)

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.Execute(ctx, workflowID, userID, nil)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestWorkflowService_ListExecutions(t *testing.T) {
	ctx := context.Background()
	workflowID := uuid.New()

	t.Run("returns paginated executions", func(t *testing.T) {
		workflowRepo := mocks.NewMockWorkflowRepository()
		executionRepo := mocks.NewMockExecutionRepository()
		executor := mocks.NewMockWorkflowExecutor()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewWorkflowService(workflowRepo, executionRepo, executor, auditService, tenantSetter)

		result, err := svc.ListExecutions(ctx, workflowID, 1, 10)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Limit)
	})
}
