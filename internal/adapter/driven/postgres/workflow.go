package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// WorkflowRepository implements port.WorkflowRepository
type WorkflowRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(pool *pgxpool.Pool) *WorkflowRepository {
	return &WorkflowRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// FindByID finds a workflow by ID
func (r *WorkflowRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	row, err := r.queries.GetWorkflow(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrWorkflowNotFound
		}
		return nil, err
	}
	return r.toDomain(row), nil
}

// FindByTenant finds workflows by tenant with pagination
func (r *WorkflowRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Workflow, error) {
	rows, err := r.queries.ListWorkflows(ctx, db.ListWorkflowsParams{
		TenantID: tenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	workflows := make([]*domain.Workflow, len(rows))
	for i, row := range rows {
		workflows[i] = r.toDomain(row)
	}
	return workflows, nil
}

// CountByTenant counts workflows for a tenant
func (r *WorkflowRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	return r.queries.CountWorkflows(ctx, tenantID)
}

// Save saves a new workflow
func (r *WorkflowRepository) Save(ctx context.Context, workflow *domain.Workflow) error {
	_, err := r.queries.CreateWorkflow(ctx, db.CreateWorkflowParams{
		TenantID:    workflow.TenantID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Definition:  workflow.Definition,
		Schedule:    workflow.Schedule,
		Status:      string(workflow.Status),
		CreatedBy:   uuidToPgtype(workflow.CreatedBy),
	})
	return err
}

// Update updates an existing workflow
func (r *WorkflowRepository) Update(ctx context.Context, workflow *domain.Workflow) error {
	_, err := r.queries.UpdateWorkflow(ctx, db.UpdateWorkflowParams{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Definition:  workflow.Definition,
		Schedule:    workflow.Schedule,
		Status:      string(workflow.Status),
	})
	return err
}

// Delete deletes a workflow
func (r *WorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteWorkflow(ctx, id)
}

// toDomain converts a db.Workflow to domain.Workflow
func (r *WorkflowRepository) toDomain(row db.Workflow) *domain.Workflow {
	var createdBy *uuid.UUID
	if row.CreatedBy.Valid {
		id := uuid.UUID(row.CreatedBy.Bytes)
		createdBy = &id
	}

	return &domain.Workflow{
		ID:          row.ID,
		TenantID:    row.TenantID,
		Name:        row.Name,
		Description: row.Description,
		Definition:  row.Definition,
		Schedule:    row.Schedule,
		Status:      domain.WorkflowStatus(row.Status),
		Version:     row.Version,
		CreatedBy:   createdBy,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

// uuidToPgtype converts *uuid.UUID to pgtype.UUID
func uuidToPgtype(id *uuid.UUID) pgtype.UUID {
	if id == nil || *id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
