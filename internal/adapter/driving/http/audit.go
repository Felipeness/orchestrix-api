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

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	service port.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(service port.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// Routes registers audit routes
func (h *AuditHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)

	return r
}

// List returns all audit logs for the tenant
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.List(ctx, user.TenantID, page, limit)
	if err != nil {
		slog.Error("failed to list audit logs", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Logs,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single audit log
func (h *AuditHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	auditLog, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAuditLogNotFound) {
			respondError(w, http.StatusNotFound, "audit log not found")
			return
		}
		slog.Error("failed to get audit log", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get audit log")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: auditLog})
}
