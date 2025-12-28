package alertrule

import (
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

// Handler handles alert rule HTTP requests
type Handler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewHandler creates a new alert rule handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Routes registers alert rule routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/test", h.Test)

	return r
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// CreateRequest represents a create alert rule request
type CreateRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description,omitempty"`
	Enabled              *bool                  `json:"enabled,omitempty"`
	ConditionType        string                 `json:"condition_type"`
	ConditionConfig      map[string]interface{} `json:"condition_config"`
	Severity             string                 `json:"severity,omitempty"`
	AlertTitleTemplate   string                 `json:"alert_title_template"`
	AlertMessageTemplate string                 `json:"alert_message_template,omitempty"`
	TriggerWorkflowID    *string                `json:"trigger_workflow_id,omitempty"`
	TriggerInputTemplate map[string]interface{} `json:"trigger_input_template,omitempty"`
	CooldownSeconds      *int                   `json:"cooldown_seconds,omitempty"`
}

// List returns all alert rules
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page := int32(1)
	limit := int32(20)

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = int32(v)
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = int32(v)
		}
	}

	offset := (page - 1) * limit

	rules, err := h.queries.ListAlertRules(ctx, db.ListAlertRulesParams{
		TenantID: user.TenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		slog.Error("failed to list alert rules", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list alert rules")
		return
	}

	count, _ := h.queries.CountAlertRules(ctx, user.TenantID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":  rules,
		"total": count,
		"page":  page,
		"limit": limit,
	})
}

// Create creates a new alert rule
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateRequest
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

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	severity := "warning"
	if req.Severity != "" {
		severity = req.Severity
	}

	cooldownSeconds := int32(300)
	if req.CooldownSeconds != nil {
		cooldownSeconds = int32(*req.CooldownSeconds)
	}

	conditionConfig, _ := json.Marshal(req.ConditionConfig)
	triggerInputTemplate, _ := json.Marshal(req.TriggerInputTemplate)

	var triggerWorkflowID pgtype.UUID
	if req.TriggerWorkflowID != nil && *req.TriggerWorkflowID != "" {
		id, err := uuid.Parse(*req.TriggerWorkflowID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid trigger_workflow_id")
			return
		}
		triggerWorkflowID = pgtype.UUID{Bytes: id, Valid: true}
	}

	userID, _ := uuid.Parse(user.ID)
	createdBy := pgtype.UUID{Bytes: userID, Valid: userID != uuid.Nil}

	rule, err := h.queries.CreateAlertRule(ctx, db.CreateAlertRuleParams{
		TenantID:             user.TenantID,
		Name:                 req.Name,
		Description:          stringPtr(req.Description),
		Enabled:              enabled,
		ConditionType:        req.ConditionType,
		ConditionConfig:      conditionConfig,
		Severity:             severity,
		AlertTitleTemplate:   req.AlertTitleTemplate,
		AlertMessageTemplate: stringPtr(req.AlertMessageTemplate),
		TriggerWorkflowID:    triggerWorkflowID,
		TriggerInputTemplate: triggerInputTemplate,
		CooldownSeconds:      cooldownSeconds,
		CreatedBy:            createdBy,
	})
	if err != nil {
		slog.Error("failed to create alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create alert rule")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{"data": rule})
}

// Get returns an alert rule by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
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

	rule, err := h.queries.GetAlertRule(ctx, db.GetAlertRuleParams{
		ID:       id,
		TenantID: user.TenantID,
	})
	if err != nil {
		respondError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": rule})
}

// Update updates an alert rule
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	// Get existing rule first
	existing, err := h.queries.GetAlertRule(ctx, db.GetAlertRuleParams{
		ID:       id,
		TenantID: user.TenantID,
	})
	if err != nil {
		respondError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Merge with existing values
	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}

	description := existing.Description
	if req.Description != "" {
		description = stringPtr(req.Description)
	}

	enabled := existing.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	conditionType := existing.ConditionType
	if req.ConditionType != "" {
		conditionType = req.ConditionType
	}

	conditionConfig := existing.ConditionConfig
	if req.ConditionConfig != nil {
		conditionConfig, _ = json.Marshal(req.ConditionConfig)
	}

	severity := existing.Severity
	if req.Severity != "" {
		severity = req.Severity
	}

	alertTitleTemplate := existing.AlertTitleTemplate
	if req.AlertTitleTemplate != "" {
		alertTitleTemplate = req.AlertTitleTemplate
	}

	alertMessageTemplate := existing.AlertMessageTemplate
	if req.AlertMessageTemplate != "" {
		alertMessageTemplate = stringPtr(req.AlertMessageTemplate)
	}

	triggerWorkflowID := existing.TriggerWorkflowID
	if req.TriggerWorkflowID != nil {
		if *req.TriggerWorkflowID == "" {
			triggerWorkflowID = pgtype.UUID{Valid: false}
		} else {
			wfID, err := uuid.Parse(*req.TriggerWorkflowID)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid trigger_workflow_id")
				return
			}
			triggerWorkflowID = pgtype.UUID{Bytes: wfID, Valid: true}
		}
	}

	triggerInputTemplate := existing.TriggerInputTemplate
	if req.TriggerInputTemplate != nil {
		triggerInputTemplate, _ = json.Marshal(req.TriggerInputTemplate)
	}

	cooldownSeconds := existing.CooldownSeconds
	if req.CooldownSeconds != nil {
		cooldownSeconds = int32(*req.CooldownSeconds)
	}

	rule, err := h.queries.UpdateAlertRule(ctx, db.UpdateAlertRuleParams{
		ID:                   id,
		TenantID:             user.TenantID,
		Name:                 name,
		Description:          description,
		Enabled:              enabled,
		ConditionType:        conditionType,
		ConditionConfig:      conditionConfig,
		Severity:             severity,
		AlertTitleTemplate:   alertTitleTemplate,
		AlertMessageTemplate: alertMessageTemplate,
		TriggerWorkflowID:    triggerWorkflowID,
		TriggerInputTemplate: triggerInputTemplate,
		CooldownSeconds:      cooldownSeconds,
	})
	if err != nil {
		slog.Error("failed to update alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update alert rule")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": rule})
}

// Delete deletes an alert rule
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
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

	err = h.queries.DeleteAlertRule(ctx, db.DeleteAlertRuleParams{
		ID:       id,
		TenantID: user.TenantID,
	})
	if err != nil {
		slog.Error("failed to delete alert rule", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete alert rule")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Test tests an alert rule (simulates triggering)
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
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

	rule, err := h.queries.GetAlertRule(ctx, db.GetAlertRuleParams{
		ID:       id,
		TenantID: user.TenantID,
	})
	if err != nil {
		respondError(w, http.StatusNotFound, "alert rule not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Alert rule test successful",
		"rule":    rule,
		"note":    "In production, this would trigger the configured workflow if set",
	})
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
