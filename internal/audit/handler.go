package audit

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

// Handler handles audit log HTTP requests
type Handler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewHandler creates a new audit handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Routes registers audit routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/resource/{type}/{id}", h.ListByResource)
	r.Get("/user/{userId}", h.ListByUser)

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

// List returns all audit logs for the tenant
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

	eventType := r.URL.Query().Get("event_type")
	var logs []db.AuditLog
	var err error

	if eventType != "" {
		logs, err = h.queries.ListAuditLogsByEventType(ctx, db.ListAuditLogsByEventTypeParams{
			TenantID:  user.TenantID,
			EventType: eventType,
			Limit:     int32(limit),
			Offset:    int32(offset),
		})
	} else {
		logs, err = h.queries.ListAuditLogs(ctx, db.ListAuditLogsParams{
			TenantID: user.TenantID,
			Limit:    int32(limit),
			Offset:   int32(offset),
		})
	}

	if err != nil {
		slog.Error("failed to list audit logs", "error", err)
		http.Error(w, "failed to list audit logs", http.StatusInternalServerError)
		return
	}

	count, err := h.queries.CountAuditLogsByTenant(ctx, user.TenantID)
	if err != nil {
		slog.Error("failed to count audit logs", "error", err)
		http.Error(w, "failed to count audit logs", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  logs,
		Total: count,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// ListByResource returns audit logs for a specific resource
func (h *Handler) ListByResource(w http.ResponseWriter, r *http.Request) {
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

	resourceType := chi.URLParam(r, "type")
	resourceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid resource id", http.StatusBadRequest)
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

	logs, err := h.queries.ListAuditLogsByResource(ctx, db.ListAuditLogsByResourceParams{
		TenantID:     user.TenantID,
		ResourceType: resourceType,
		ResourceID:   uuidToPgtype(resourceID),
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		slog.Error("failed to list audit logs", "error", err)
		http.Error(w, "failed to list audit logs", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  logs,
		Total: int64(len(logs)),
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// ListByUser returns audit logs for a specific user
func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) {
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

	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
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

	logs, err := h.queries.ListAuditLogsByUser(ctx, db.ListAuditLogsByUserParams{
		TenantID: user.TenantID,
		UserID:   uuidToPgtype(userID),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		slog.Error("failed to list audit logs", "error", err)
		http.Error(w, "failed to list audit logs", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  logs,
		Total: int64(len(logs)),
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
