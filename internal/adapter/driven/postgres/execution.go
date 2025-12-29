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

// ExecutionRepository implements port.ExecutionRepository
type ExecutionRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewExecutionRepository creates a new execution repository
func NewExecutionRepository(pool *pgxpool.Pool) *ExecutionRepository {
	return &ExecutionRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// FindByID finds an execution by ID
func (r *ExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error) {
	row, err := r.queries.GetExecution(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrExecutionNotFound
		}
		return nil, err
	}
	return r.toDomain(row), nil
}

// FindByTenant finds executions by tenant with pagination
func (r *ExecutionRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Execution, error) {
	rows, err := r.queries.ListExecutions(ctx, db.ListExecutionsParams{
		TenantID: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	executions := make([]*domain.Execution, len(rows))
	for i, row := range rows {
		executions[i] = r.toDomain(row)
	}
	return executions, nil
}

// FindByWorkflow finds executions by workflow with pagination
func (r *ExecutionRepository) FindByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*domain.Execution, error) {
	rows, err := r.queries.ListExecutionsByWorkflow(ctx, db.ListExecutionsByWorkflowParams{
		WorkflowID: workflowID,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}

	executions := make([]*domain.Execution, len(rows))
	for i, row := range rows {
		executions[i] = r.toDomain(row)
	}
	return executions, nil
}

// CountByTenant counts executions for a tenant
func (r *ExecutionRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.queries.CountExecutions(ctx, tenantID)
}

// Save saves a new execution
func (r *ExecutionRepository) Save(ctx context.Context, execution *domain.Execution) error {
	_, err := r.queries.CreateExecution(ctx, db.CreateExecutionParams{
		TenantID:           execution.TenantID,
		WorkflowID:         execution.WorkflowID,
		TemporalWorkflowID: execution.TemporalWorkflowID,
		Status:             string(execution.Status),
		Input:              execution.Input,
		TriggeredBy:        execution.TriggeredBy,
	})
	return err
}

// Update updates an existing execution
func (r *ExecutionRepository) Update(ctx context.Context, execution *domain.Execution) error {
	return r.queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
		ID:     execution.ID,
		Status: string(execution.Status),
		Error:  execution.Error,
	})
}

// UpdateStatus updates the status of an execution
func (r *ExecutionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error {
	return r.queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
		ID:     id,
		Status: string(status),
		Error:  errMsg,
	})
}

// UpdateTemporalIDs updates the temporal IDs of an execution
func (r *ExecutionRepository) UpdateTemporalIDs(ctx context.Context, id uuid.UUID, temporalWorkflowID, temporalRunID string) error {
	return r.queries.UpdateExecutionTemporalIDs(ctx, db.UpdateExecutionTemporalIDsParams{
		ID:                 id,
		TemporalWorkflowID: &temporalWorkflowID,
		TemporalRunID:      &temporalRunID,
	})
}

// toDomain converts a db.Execution to domain.Execution
func (r *ExecutionRepository) toDomain(row db.Execution) *domain.Execution {
	var createdBy *uuid.UUID
	if row.CreatedBy.Valid {
		id := uuid.UUID(row.CreatedBy.Bytes)
		createdBy = &id
	}

	var startedAt, completedAt *time.Time
	if row.StartedAt.Valid {
		t := row.StartedAt.Time
		startedAt = &t
	}
	if row.CompletedAt.Valid {
		t := row.CompletedAt.Time
		completedAt = &t
	}

	return &domain.Execution{
		ID:                 row.ID,
		TenantID:           row.TenantID,
		WorkflowID:         row.WorkflowID,
		TemporalWorkflowID: row.TemporalWorkflowID,
		TemporalRunID:      row.TemporalRunID,
		Status:             domain.ExecutionStatus(row.Status),
		Input:              row.Input,
		Output:             row.Output,
		Error:              row.Error,
		StartedAt:          startedAt,
		CompletedAt:        completedAt,
		CreatedBy:          createdBy,
		CreatedAt:          row.CreatedAt,
		TriggeredBy:        row.TriggeredBy,
	}
}
