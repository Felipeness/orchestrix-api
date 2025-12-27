package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/db"
	"github.com/orchestrix/orchestrix-api/pkg/temporal"
)

// Handler handles workflow HTTP requests
type Handler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewHandler creates a new workflow handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Routes registers workflow routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/execute", h.Execute)
	r.Get("/{id}/executions", h.ListExecutions)

	return r
}

// ListRequest represents pagination params
type ListRequest struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
	Page  int32       `json:"page"`
	Limit int32       `json:"limit"`
}

// CreateRequest represents a create workflow request
type CreateRequest struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	Definition  map[string]interface{} `json:"definition"`
	Schedule    *string                `json:"schedule"`
}

// UpdateRequest represents an update workflow request
type UpdateRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Definition  map[string]interface{} `json:"definition"`
	Schedule    *string                `json:"schedule"`
	Status      *string                `json:"status"`
}

// ExecuteRequest represents an execute workflow request
type ExecuteRequest struct {
	Input map[string]interface{} `json:"input"`
}

func (h *Handler) setTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	_, err := h.pool.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)",
		tenantID.String())
	return err
}

// List returns all workflows for the tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	workflows, err := h.queries.ListWorkflows(ctx, db.ListWorkflowsParams{
		TenantID: user.TenantID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		slog.Error("failed to list workflows", "error", err)
		http.Error(w, "failed to list workflows", http.StatusInternalServerError)
		return
	}

	count, err := h.queries.CountWorkflows(ctx, user.TenantID)
	if err != nil {
		slog.Error("failed to count workflows", "error", err)
		http.Error(w, "failed to count workflows", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  workflows,
		Total: count,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single workflow
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	workflow, err := h.queries.GetWorkflow(ctx, id)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": workflow})
}

// Create creates a new workflow
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	definition, err := json.Marshal(req.Definition)
	if err != nil {
		http.Error(w, "invalid definition", http.StatusBadRequest)
		return
	}

	userID, _ := uuid.Parse(user.ID)

	workflow, err := h.queries.CreateWorkflow(ctx, db.CreateWorkflowParams{
		TenantID:    user.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Definition:  definition,
		Schedule:    req.Schedule,
		Status:      "draft",
		CreatedBy:   uuidToPgtype(userID),
	})
	if err != nil {
		slog.Error("failed to create workflow", "error", err)
		http.Error(w, "failed to create workflow", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{"data": workflow})
}

// Update updates a workflow
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Get existing workflow
	existing, err := h.queries.GetWorkflow(ctx, id)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Apply updates
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}

	definition := existing.Definition
	if req.Definition != nil {
		definition, _ = json.Marshal(req.Definition)
	}

	schedule := existing.Schedule
	if req.Schedule != nil {
		schedule = req.Schedule
	}

	status := existing.Status
	if req.Status != nil {
		status = *req.Status
	}

	workflow, err := h.queries.UpdateWorkflow(ctx, db.UpdateWorkflowParams{
		ID:          id,
		Name:        name,
		Description: description,
		Definition:  definition,
		Schedule:    schedule,
		Status:      status,
	})
	if err != nil {
		slog.Error("failed to update workflow", "error", err)
		http.Error(w, "failed to update workflow", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": workflow})
}

// Delete deletes a workflow
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.queries.DeleteWorkflow(ctx, id); err != nil {
		slog.Error("failed to delete workflow", "error", err)
		http.Error(w, "failed to delete workflow", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Execute starts a workflow execution
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Get workflow
	wf, err := h.queries.GetWorkflow(ctx, workflowID)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	input, _ := json.Marshal(req.Input)
	userID, _ := uuid.Parse(user.ID)

	// Create execution record
	execution, err := h.queries.CreateExecution(ctx, db.CreateExecutionParams{
		TenantID:   user.TenantID,
		WorkflowID: workflowID,
		Status:     "pending",
		Input:      input,
		CreatedBy:  uuidToPgtype(userID),
	})
	if err != nil {
		slog.Error("failed to create execution", "error", err)
		http.Error(w, "failed to create execution", http.StatusInternalServerError)
		return
	}

	// Start Temporal workflow
	temporalWorkflowID := fmt.Sprintf("orchestrix-%s-%s", workflowID.String(), execution.ID.String())
	params := make(map[string]string)
	for k, v := range req.Input {
		if s, ok := v.(string); ok {
			params[k] = s
		}
	}

	workflowInput := ProcessWorkflowInput{
		ID:     execution.ID.String(),
		Name:   wf.Name,
		Params: params,
	}

	run, err := temporal.ExecuteWorkflow(ctx, temporalWorkflowID, ProcessWorkflow, workflowInput)
	if err != nil {
		slog.Error("failed to start workflow", "error", err)
		// Update execution status to failed
		h.queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
			ID:     execution.ID,
			Status: "failed",
		})
		http.Error(w, "failed to start workflow", http.StatusInternalServerError)
		return
	}

	// Update execution with Temporal IDs
	temporalRunID := run.GetRunID()
	h.queries.UpdateExecutionTemporalIDs(ctx, db.UpdateExecutionTemporalIDsParams{
		ID:                 execution.ID,
		TemporalWorkflowID: &temporalWorkflowID,
		TemporalRunID:      &temporalRunID,
	})

	// Update status to running
	h.queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
		ID:     execution.ID,
		Status: "running",
	})

	// Get updated execution
	execution, _ = h.queries.GetExecution(ctx, execution.ID)

	respondJSON(w, http.StatusAccepted, map[string]interface{}{"data": execution})
}

// ListExecutions returns executions for a workflow
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.setTenantContext(ctx, user.TenantID); err != nil {
		slog.Error("failed to set tenant context", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	workflowID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	executions, err := h.queries.ListExecutionsByWorkflow(ctx, db.ListExecutionsByWorkflowParams{
		WorkflowID: workflowID,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		slog.Error("failed to list executions", "error", err)
		http.Error(w, "failed to list executions", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  executions,
		Total: int64(len(executions)),
		Page:  int32(page),
		Limit: int32(limit),
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	if id == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}
