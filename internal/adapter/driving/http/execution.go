package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// ExecutionHandler handles execution HTTP requests
type ExecutionHandler struct {
	service port.ExecutionService
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(service port.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{service: service}
}

// Routes registers execution routes
func (h *ExecutionHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/cancel", h.Cancel)

	return r
}

// List returns all executions for the tenant
func (h *ExecutionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.List(ctx, user.TenantID, page, limit)
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

// Get returns a single execution
func (h *ExecutionHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	execution, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrExecutionNotFound) {
			respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		slog.Error("failed to get execution", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get execution")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: execution})
}

// Cancel cancels a running execution
func (h *ExecutionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.Cancel(ctx, id); err != nil {
		if errors.Is(err, domain.ErrExecutionNotFound) {
			respondError(w, http.StatusNotFound, "execution not found")
			return
		}
		if errors.Is(err, domain.ErrExecutionCannotCancel) {
			respondError(w, http.StatusBadRequest, "execution cannot be cancelled")
			return
		}
		slog.Error("failed to cancel execution", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to cancel execution")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
