package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// WorkflowHandler handles workflow HTTP requests
type WorkflowHandler struct {
	service port.WorkflowService
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(service port.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{service: service}
}

// Routes registers workflow routes
func (h *WorkflowHandler) Routes() chi.Router {
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

// List returns all workflows for the tenant
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.List(ctx, user.TenantID, page, limit)
	if err != nil {
		slog.Error("failed to list workflows", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list workflows")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Workflows,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single workflow
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	workflow, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		slog.Error("failed to get workflow", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get workflow")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: workflow})
}

// Create creates a new workflow
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	definition, err := json.Marshal(req.Definition)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid definition")
		return
	}

	userID, _ := uuid.Parse(user.ID)

	input := port.CreateWorkflowInput{
		TenantID:    user.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Definition:  definition,
		Schedule:    req.Schedule,
		CreatedBy:   &userID,
	}

	workflow, err := h.service.Create(ctx, input)
	if err != nil {
		slog.Error("failed to create workflow", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create workflow")
		return
	}

	respondJSON(w, http.StatusCreated, DataResponse{Data: workflow})
}

// Update updates a workflow
func (h *WorkflowHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req UpdateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var definition []byte
	if req.Definition != nil {
		definition, _ = json.Marshal(req.Definition)
	}

	var status *domain.WorkflowStatus
	if req.Status != nil {
		s := domain.WorkflowStatus(*req.Status)
		status = &s
	}

	input := port.UpdateWorkflowInput{
		Name:        req.Name,
		Description: req.Description,
		Definition:  definition,
		Schedule:    req.Schedule,
		Status:      status,
	}

	workflow, err := h.service.Update(ctx, id, input)
	if err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		slog.Error("failed to update workflow", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update workflow")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: workflow})
}

// Delete deletes a workflow
func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		slog.Error("failed to delete workflow", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete workflow")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Execute starts a workflow execution
func (h *WorkflowHandler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req ExecuteWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	execution, err := h.service.Execute(ctx, id, user.ID, req.Input)
	if err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			respondError(w, http.StatusNotFound, "workflow not found")
			return
		}
		if errors.Is(err, domain.ErrWorkflowCannotExecute) {
			respondError(w, http.StatusBadRequest, "workflow cannot be executed")
			return
		}
		slog.Error("failed to execute workflow", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to execute workflow")
		return
	}

	respondJSON(w, http.StatusAccepted, DataResponse{Data: execution})
}

// ListExecutions returns executions for a workflow
func (h *WorkflowHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.ListExecutions(ctx, id, page, limit)
	if err != nil {
		slog.Error("failed to list executions", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Executions,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Request/Response types
type CreateWorkflowRequest struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	Definition  map[string]interface{} `json:"definition"`
	Schedule    *string                `json:"schedule"`
}

type UpdateWorkflowRequest struct {
	Name        *string                `json:"name"`
	Description *string                `json:"description"`
	Definition  map[string]interface{} `json:"definition"`
	Schedule    *string                `json:"schedule"`
	Status      *string                `json:"status"`
}

type ExecuteWorkflowRequest struct {
	Input map[string]interface{} `json:"input"`
}

// parsePagination extracts pagination parameters from the request
func parsePagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}
