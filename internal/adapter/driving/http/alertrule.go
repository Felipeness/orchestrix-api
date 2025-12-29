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

// AlertRuleHandler handles alert rule HTTP requests
type AlertRuleHandler struct {
	service port.AlertRuleService
}

// NewAlertRuleHandler creates a new alert rule handler
func NewAlertRuleHandler(service port.AlertRuleService) *AlertRuleHandler {
	return &AlertRuleHandler{service: service}
}

// Routes registers alert rule routes
func (h *AlertRuleHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// List returns all alert rules for the tenant
func (h *AlertRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.List(ctx, user.TenantID, page, limit)
	if err != nil {
		slog.Error("failed to list alert rules", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list alert rules")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Rules,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// Get returns a single alert rule
func (h *AlertRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	rule, err := h.service.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrAlertRuleNotFound) {
			respondError(w, http.StatusNotFound, "alert rule not found")
			return
		}
		slog.Error("failed to get alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get alert rule")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: rule})
}

// Create creates a new alert rule
func (h *AlertRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.ConditionType == "" {
		respondError(w, http.StatusBadRequest, "condition_type is required")
		return
	}
	if req.AlertTitleTemplate == "" {
		respondError(w, http.StatusBadRequest, "alert_title_template is required")
		return
	}

	conditionConfig, _ := json.Marshal(req.ConditionConfig)
	triggerInputTemplate, _ := json.Marshal(req.TriggerInputTemplate)

	severity := domain.AlertSeverityWarning
	if req.Severity != "" {
		severity = domain.AlertSeverity(req.Severity)
	}

	cooldownSeconds := int32(300)
	if req.CooldownSeconds != nil {
		cooldownSeconds = int32(*req.CooldownSeconds)
	}

	userID, _ := uuid.Parse(user.ID)

	input := port.CreateAlertRuleInput{
		TenantID:             user.TenantID,
		Name:                 req.Name,
		Description:          req.Description,
		ConditionType:        req.ConditionType,
		ConditionConfig:      conditionConfig,
		Severity:             severity,
		AlertTitleTemplate:   req.AlertTitleTemplate,
		AlertMessageTemplate: req.AlertMessageTemplate,
		TriggerWorkflowID:    req.TriggerWorkflowID,
		TriggerInputTemplate: triggerInputTemplate,
		CooldownSeconds:      cooldownSeconds,
		CreatedBy:            userID,
	}

	rule, err := h.service.Create(ctx, input)
	if err != nil {
		slog.Error("failed to create alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create alert rule")
		return
	}

	respondJSON(w, http.StatusCreated, DataResponse{Data: rule})
}

// Update updates an alert rule
func (h *AlertRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var conditionConfig json.RawMessage
	if req.ConditionConfig != nil {
		conditionConfig, _ = json.Marshal(req.ConditionConfig)
	}

	var triggerInputTemplate json.RawMessage
	if req.TriggerInputTemplate != nil {
		triggerInputTemplate, _ = json.Marshal(req.TriggerInputTemplate)
	}

	var severity *domain.AlertSeverity
	if req.Severity != nil {
		s := domain.AlertSeverity(*req.Severity)
		severity = &s
	}

	var cooldownSeconds *int32
	if req.CooldownSeconds != nil {
		cs := int32(*req.CooldownSeconds)
		cooldownSeconds = &cs
	}

	input := port.UpdateAlertRuleInput{
		Name:                 req.Name,
		Description:          req.Description,
		Enabled:              req.Enabled,
		ConditionType:        req.ConditionType,
		ConditionConfig:      conditionConfig,
		Severity:             severity,
		AlertTitleTemplate:   req.AlertTitleTemplate,
		AlertMessageTemplate: req.AlertMessageTemplate,
		TriggerWorkflowID:    req.TriggerWorkflowID,
		TriggerInputTemplate: triggerInputTemplate,
		CooldownSeconds:      cooldownSeconds,
	}

	rule, err := h.service.Update(ctx, id, input)
	if err != nil {
		if errors.Is(err, domain.ErrAlertRuleNotFound) {
			respondError(w, http.StatusNotFound, "alert rule not found")
			return
		}
		slog.Error("failed to update alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update alert rule")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: rule})
}

// Delete deletes an alert rule
func (h *AlertRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, domain.ErrAlertRuleNotFound) {
			respondError(w, http.StatusNotFound, "alert rule not found")
			return
		}
		slog.Error("failed to delete alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete alert rule")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Request types
type CreateAlertRuleRequest struct {
	Name                 string                 `json:"name"`
	Description          *string                `json:"description,omitempty"`
	ConditionType        string                 `json:"condition_type"`
	ConditionConfig      map[string]interface{} `json:"condition_config"`
	Severity             string                 `json:"severity,omitempty"`
	AlertTitleTemplate   string                 `json:"alert_title_template"`
	AlertMessageTemplate *string                `json:"alert_message_template,omitempty"`
	TriggerWorkflowID    *uuid.UUID             `json:"trigger_workflow_id,omitempty"`
	TriggerInputTemplate map[string]interface{} `json:"trigger_input_template,omitempty"`
	CooldownSeconds      *int                   `json:"cooldown_seconds,omitempty"`
}

type UpdateAlertRuleRequest struct {
	Name                 *string                `json:"name,omitempty"`
	Description          *string                `json:"description,omitempty"`
	Enabled              *bool                  `json:"enabled,omitempty"`
	ConditionType        *string                `json:"condition_type,omitempty"`
	ConditionConfig      map[string]interface{} `json:"condition_config,omitempty"`
	Severity             *string                `json:"severity,omitempty"`
	AlertTitleTemplate   *string                `json:"alert_title_template,omitempty"`
	AlertMessageTemplate *string                `json:"alert_message_template,omitempty"`
	TriggerWorkflowID    *uuid.UUID             `json:"trigger_workflow_id,omitempty"`
	TriggerInputTemplate map[string]interface{} `json:"trigger_input_template,omitempty"`
	CooldownSeconds      *int                   `json:"cooldown_seconds,omitempty"`
}
