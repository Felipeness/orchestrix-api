package execution

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/db"
	"github.com/orchestrix/orchestrix-api/pkg/temporal"
)

// Handler handles execution HTTP requests
type Handler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewHandler creates a new execution handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Routes registers execution routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/cancel", h.Cancel)

	return r
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
	Page  int32       `json:"page"`
	Limit int32       `json:"limit"`
}

func (h *Handler) setTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	_, err := h.pool.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)",
		tenantID.String())
	return err
}

// List returns all executions for the tenant
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

	status := r.URL.Query().Get("status")
	var executions []db.Execution
	var err error

	if status != "" {
		executions, err = h.queries.ListExecutionsByStatus(ctx, db.ListExecutionsByStatusParams{
			TenantID: user.TenantID,
			Status:   status,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
	} else {
		executions, err = h.queries.ListExecutions(ctx, db.ListExecutionsParams{
			TenantID: user.TenantID,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
	}

	if err != nil {
		slog.Error("failed to list executions", "error", err)
		http.Error(w, "failed to list executions", http.StatusInternalServerError)
		return
	}

	count, err := h.queries.CountExecutions(ctx, user.TenantID)
	if err != nil {
		slog.Error("failed to count executions", "error", err)
		http.Error(w, "failed to count executions", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  executions,
		Total: count,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single execution
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

	execution, err := h.queries.GetExecution(ctx, id)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": execution})
}

// Cancel cancels a running execution
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
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

	execution, err := h.queries.GetExecution(ctx, id)
	if err != nil {
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	if execution.Status != "running" && execution.Status != "pending" {
		http.Error(w, "execution cannot be cancelled", http.StatusBadRequest)
		return
	}

	// Cancel in Temporal
	if execution.TemporalWorkflowID != nil && execution.TemporalRunID != nil {
		if err := temporal.CancelWorkflow(ctx, *execution.TemporalWorkflowID, *execution.TemporalRunID); err != nil {
			slog.Error("failed to cancel workflow in temporal", "error", err)
			// Continue anyway to update status
		}
	}

	// Update status
	h.queries.UpdateExecutionStatus(ctx, db.UpdateExecutionStatusParams{
		ID:     id,
		Status: "cancelled",
	})

	// Get updated execution
	execution, _ = h.queries.GetExecution(ctx, id)

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": execution})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
