package alert

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// Handler handles alert HTTP requests
type Handler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewHandler creates a new alert handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Routes registers alert routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/", h.Create)
	r.Post("/{id}/acknowledge", h.Acknowledge)
	r.Post("/{id}/resolve", h.Resolve)

	return r
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
	Page  int32       `json:"page"`
	Limit int32       `json:"limit"`
}

// CreateAlertRequest represents a create alert request
type CreateAlertRequest struct {
	Severity string                 `json:"severity"`
	Title    string                 `json:"title"`
	Message  *string                `json:"message"`
	Source   *string                `json:"source"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (h *Handler) setTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	_, err := h.pool.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)",
		tenantID.String())
	return err
}

// List returns all alerts for the tenant
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
	var alerts []db.Alert
	var err error

	if status != "" {
		alerts, err = h.queries.ListAlertsByStatus(ctx, db.ListAlertsByStatusParams{
			TenantID: user.TenantID,
			Status:   status,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
	} else {
		alerts, err = h.queries.ListAlerts(ctx, db.ListAlertsParams{
			TenantID: user.TenantID,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
	}

	if err != nil {
		slog.Error("failed to list alerts", "error", err)
		http.Error(w, "failed to list alerts", http.StatusInternalServerError)
		return
	}

	count, err := h.queries.CountAlerts(ctx, user.TenantID)
	if err != nil {
		slog.Error("failed to count alerts", "error", err)
		http.Error(w, "failed to count alerts", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  alerts,
		Total: count,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single alert
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

	alert, err := h.queries.GetAlert(ctx, id)
	if err != nil {
		http.Error(w, "alert not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": alert})
}

// Create creates a new alert
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

	var req CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	if req.Severity == "" {
		req.Severity = "info"
	}

	var metadata []byte
	if req.Metadata != nil {
		metadata, _ = json.Marshal(req.Metadata)
	} else {
		metadata = []byte("{}")
	}

	alert, err := h.queries.CreateAlert(ctx, db.CreateAlertParams{
		TenantID: user.TenantID,
		Title:    req.Title,
		Message:  req.Message,
		Severity: req.Severity,
		Source:   req.Source,
		Metadata: metadata,
	})
	if err != nil {
		slog.Error("failed to create alert", "error", err)
		http.Error(w, "failed to create alert", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{"data": alert})
}

// Acknowledge acknowledges an alert
func (h *Handler) Acknowledge(w http.ResponseWriter, r *http.Request) {
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

	userID, _ := uuid.Parse(user.ID)

	alert, err := h.queries.AcknowledgeAlert(ctx, db.AcknowledgeAlertParams{
		ID:             id,
		AcknowledgedBy: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		slog.Error("failed to acknowledge alert", "error", err)
		http.Error(w, "failed to acknowledge alert", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": alert})
}

// Resolve resolves an alert
func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
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

	userID, _ := uuid.Parse(user.ID)

	alert, err := h.queries.ResolveAlert(ctx, db.ResolveAlertParams{
		ID:         id,
		ResolvedBy: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		slog.Error("failed to resolve alert", "error", err)
		http.Error(w, "failed to resolve alert", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": alert})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
