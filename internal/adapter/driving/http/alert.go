package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// AlertHandler handles alert HTTP requests
type AlertHandler struct {
	service port.AlertService
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(service port.AlertService) *AlertHandler {
	return &AlertHandler{service: service}
}

// Routes registers alert routes
func (h *AlertHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/", h.Create)
	r.Post("/{id}/acknowledge", h.Acknowledge)
	r.Post("/{id}/resolve", h.Resolve)

	return r
}

// List returns all alerts for the tenant
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.List(ctx, user.TenantID, page, limit)
	if err != nil {
		slog.Error("failed to list alerts", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list alerts")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Alerts,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single alert
func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	alert, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAlertNotFound) {
			respondError(w, http.StatusNotFound, "alert not found")
			return
		}
		slog.Error("failed to get alert", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get alert")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: alert})
}

// Create creates a new alert
func (h *AlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Severity == "" {
		respondError(w, http.StatusBadRequest, "severity is required")
		return
	}

	input := port.CreateAlertInput{
		TenantID:    user.TenantID,
		WorkflowID:  req.WorkflowID,
		ExecutionID: req.ExecutionID,
		Severity:    domain.AlertSeverity(req.Severity),
		Title:       req.Title,
		Message:     req.Message,
		Source:      req.Source,
	}

	alert, err := h.service.Create(ctx, input)
	if err != nil {
		slog.Error("failed to create alert", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create alert")
		return
	}

	respondJSON(w, http.StatusCreated, DataResponse{Data: alert})
}

// Acknowledge acknowledges an alert
func (h *AlertHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
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

	userID, _ := uuid.Parse(user.ID)

	alert, err := h.service.Acknowledge(ctx, id, userID)
	if err != nil {
		if errors.Is(err, domain.ErrAlertNotFound) {
			respondError(w, http.StatusNotFound, "alert not found")
			return
		}
		if errors.Is(err, domain.ErrAlertAlreadyAcknowledged) {
			respondError(w, http.StatusBadRequest, "alert already acknowledged")
			return
		}
		slog.Error("failed to acknowledge alert", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to acknowledge alert")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: alert})
}

// Resolve resolves an alert
func (h *AlertHandler) Resolve(w http.ResponseWriter, r *http.Request) {
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

	userID, _ := uuid.Parse(user.ID)

	alert, err := h.service.Resolve(ctx, id, userID)
	if err != nil {
		if errors.Is(err, domain.ErrAlertNotFound) {
			respondError(w, http.StatusNotFound, "alert not found")
			return
		}
		if errors.Is(err, domain.ErrAlertAlreadyResolved) {
			respondError(w, http.StatusBadRequest, "alert already resolved")
			return
		}
		slog.Error("failed to resolve alert", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to resolve alert")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: alert})
}

// Request types
type CreateAlertRequest struct {
	WorkflowID  *uuid.UUID `json:"workflow_id"`
	ExecutionID *uuid.UUID `json:"execution_id"`
	Severity    string     `json:"severity"`
	Title       string     `json:"title"`
	Message     *string    `json:"message"`
	Source      *string    `json:"source"`
}
