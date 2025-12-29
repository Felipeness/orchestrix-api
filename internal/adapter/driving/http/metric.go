package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// MetricHandler handles metric HTTP requests
type MetricHandler struct {
	service port.MetricService
}

// NewMetricHandler creates a new metric handler
func NewMetricHandler(service port.MetricService) *MetricHandler {
	return &MetricHandler{service: service}
}

// Routes registers metric routes
func (h *MetricHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Ingestion endpoints
	r.Post("/ingest", h.Ingest)
	r.Post("/ingest/batch", h.IngestBatch)

	// Query endpoints
	r.Get("/", h.Query)
	r.Get("/names", h.ListNames)
	r.Get("/latest/{name}", h.GetLatest)
	r.Get("/aggregate/{name}", h.GetAggregate)
	r.Get("/series/{name}", h.GetSeries)

	// Metric Definitions (metadata)
	r.Route("/definitions", func(r chi.Router) {
		r.Get("/", h.ListDefinitions)
		r.Post("/", h.CreateDefinition)
		r.Get("/{name}", h.GetDefinition)
		r.Put("/{name}", h.UpdateDefinition)
		r.Delete("/{name}", h.DeleteDefinition)
	})

	return r
}

// Ingest ingests a single metric
func (h *MetricHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req IngestMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	input := port.IngestMetricInput{
		TenantID:  user.TenantID,
		Name:      req.Name,
		Value:     req.Value,
		Labels:    req.Labels,
		Source:    req.Source,
		Timestamp: req.Timestamp,
	}

	if err := h.service.Ingest(ctx, input); err != nil {
		slog.Error("failed to ingest metric", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to ingest metric")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]string{"status": "ingested"})
}

// IngestBatch ingests multiple metrics
func (h *MetricHandler) IngestBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req IngestBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Metrics) == 0 {
		respondError(w, http.StatusBadRequest, "metrics array is required")
		return
	}

	metrics := make([]port.IngestMetricInput, len(req.Metrics))
	for i, m := range req.Metrics {
		if m.Name == "" {
			respondError(w, http.StatusBadRequest, "name is required for all metrics")
			return
		}
		metrics[i] = port.IngestMetricInput{
			TenantID:  user.TenantID,
			Name:      m.Name,
			Value:     m.Value,
			Labels:    m.Labels,
			Source:    m.Source,
			Timestamp: m.Timestamp,
		}
	}

	result, err := h.service.IngestBatch(ctx, port.IngestMetricBatchInput{
		TenantID: user.TenantID,
		Metrics:  metrics,
	})
	if err != nil {
		if errors.Is(err, domain.ErrBatchTooLarge) {
			respondError(w, http.StatusBadRequest, "batch size exceeds maximum limit (10000)")
			return
		}
		slog.Error("failed to ingest metrics batch", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to ingest metrics")
		return
	}

	respondJSON(w, http.StatusCreated, result)
}

// Query queries metrics with filters
func (h *MetricHandler) Query(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	start, end := parseTimeRangeWithDefaults(r, time.Hour)
	page, limit := parsePagination(r)

	query := domain.MetricQuery{
		TenantID:  user.TenantID,
		Name:      name,
		StartTime: start,
		EndTime:   end,
		Limit:     limit,
		Offset:    (page - 1) * limit,
	}

	if labelsStr := r.URL.Query().Get("labels"); labelsStr != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err == nil {
			query.Labels = labels
		}
	}

	result, err := h.service.Query(ctx, query)
	if err != nil {
		slog.Error("failed to query metrics", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to query metrics")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Metrics,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// ListNames lists distinct metric names
func (h *MetricHandler) ListNames(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	prefix := r.URL.Query().Get("prefix")

	names, err := h.service.ListNames(ctx, user.TenantID, prefix)
	if err != nil {
		slog.Error("failed to list metric names", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list metric names")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: names})
}

// GetLatest returns the latest metric value
func (h *MetricHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")

	// Parse labels (optional)
	var labels map[string]string
	if labelsStr := r.URL.Query().Get("labels"); labelsStr != "" {
		if err := json.Unmarshal([]byte(labelsStr), &labels); err != nil {
			respondError(w, http.StatusBadRequest, "invalid labels format")
			return
		}
	}

	metric, err := h.service.GetLatest(ctx, user.TenantID, name, labels)
	if err != nil {
		if errors.Is(err, domain.ErrMetricNotFound) {
			respondError(w, http.StatusNotFound, "metric not found")
			return
		}
		slog.Error("failed to get latest metric", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get latest metric")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: metric})
}

// GetAggregate returns aggregated metric statistics
func (h *MetricHandler) GetAggregate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")
	start, end := parseTimeRangeWithDefaults(r, time.Hour)

	query := domain.MetricQuery{
		TenantID:  user.TenantID,
		Name:      name,
		StartTime: start,
		EndTime:   end,
	}

	aggregate, err := h.service.GetAggregate(ctx, query)
	if err != nil {
		slog.Error("failed to get aggregate", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get aggregate")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: aggregate})
}

// GetSeries returns time-bucketed metric series
func (h *MetricHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")
	start, end := parseTimeRangeWithDefaults(r, time.Hour)

	query := domain.MetricQuery{
		TenantID:  user.TenantID,
		Name:      name,
		StartTime: start,
		EndTime:   end,
	}

	bucketSize := 5 * time.Minute
	if bucketStr := r.URL.Query().Get("bucket"); bucketStr != "" {
		if d, err := time.ParseDuration(bucketStr); err == nil && d > 0 {
			bucketSize = d
		}
	}

	series, err := h.service.GetSeries(ctx, query, bucketSize)
	if err != nil {
		slog.Error("failed to get series", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get series")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: series})
}

// ListDefinitions lists metric definitions
func (h *MetricHandler) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, limit := parsePagination(r)

	result, err := h.service.ListDefinitions(ctx, user.TenantID, page, limit)
	if err != nil {
		slog.Error("failed to list metric definitions", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list metric definitions")
		return
	}

	respondJSON(w, http.StatusOK, PaginatedResponse{
		Data:  result.Definitions,
		Total: result.Total,
		Page:  int32(page),
		Limit: int32(limit),
	})
}

// GetDefinition returns a metric definition
func (h *MetricHandler) GetDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")

	def, err := h.service.GetDefinition(ctx, user.TenantID, name)
	if err != nil {
		if errors.Is(err, domain.ErrMetricDefinitionNotFound) {
			respondError(w, http.StatusNotFound, "metric definition not found")
			return
		}
		slog.Error("failed to get metric definition", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get metric definition")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: def})
}

// CreateDefinition creates a metric definition
func (h *MetricHandler) CreateDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateMetricDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Validate type
	metricType := domain.MetricType(req.Type)
	if req.Type != "" && !metricType.IsValid() {
		respondError(w, http.StatusBadRequest, "invalid metric type")
		return
	}
	if req.Type == "" {
		metricType = domain.MetricTypeGauge
	}

	// Validate aggregation
	aggregation := domain.AggregationType(req.Aggregation)
	if req.Aggregation != "" && !aggregation.IsValid() {
		respondError(w, http.StatusBadRequest, "invalid aggregation type")
		return
	}
	if req.Aggregation == "" {
		aggregation = domain.AggregationAvg
	}

	retentionDays := 30
	if req.RetentionDays != nil && *req.RetentionDays > 0 {
		retentionDays = *req.RetentionDays
	}

	input := port.CreateMetricDefinitionInput{
		TenantID:       user.TenantID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Unit:           req.Unit,
		Type:           metricType,
		Aggregation:    aggregation,
		AlertThreshold: req.AlertThreshold,
		RetentionDays:  retentionDays,
	}

	def, err := h.service.CreateDefinition(ctx, input)
	if err != nil {
		if errors.Is(err, domain.ErrMetricDefinitionExists) {
			respondError(w, http.StatusConflict, "metric definition already exists")
			return
		}
		slog.Error("failed to create metric definition", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create metric definition")
		return
	}

	respondJSON(w, http.StatusCreated, DataResponse{Data: def})
}

// UpdateDefinition updates a metric definition
func (h *MetricHandler) UpdateDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")

	var req UpdateMetricDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate type if provided
	var metricType *domain.MetricType
	if req.Type != nil {
		mt := domain.MetricType(*req.Type)
		if !mt.IsValid() {
			respondError(w, http.StatusBadRequest, "invalid metric type")
			return
		}
		metricType = &mt
	}

	// Validate aggregation if provided
	var aggregation *domain.AggregationType
	if req.Aggregation != nil {
		agg := domain.AggregationType(*req.Aggregation)
		if !agg.IsValid() {
			respondError(w, http.StatusBadRequest, "invalid aggregation type")
			return
		}
		aggregation = &agg
	}

	input := port.UpdateMetricDefinitionInput{
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Unit:           req.Unit,
		Type:           metricType,
		Aggregation:    aggregation,
		AlertThreshold: req.AlertThreshold,
		RetentionDays:  req.RetentionDays,
	}

	def, err := h.service.UpdateDefinition(ctx, user.TenantID, name, input)
	if err != nil {
		if errors.Is(err, domain.ErrMetricDefinitionNotFound) {
			respondError(w, http.StatusNotFound, "metric definition not found")
			return
		}
		slog.Error("failed to update metric definition", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to update metric definition")
		return
	}

	respondJSON(w, http.StatusOK, DataResponse{Data: def})
}

// DeleteDefinition deletes a metric definition
func (h *MetricHandler) DeleteDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")

	if err := h.service.DeleteDefinition(ctx, user.TenantID, name); err != nil {
		if errors.Is(err, domain.ErrMetricDefinitionNotFound) {
			respondError(w, http.StatusNotFound, "metric definition not found")
			return
		}
		slog.Error("failed to delete metric definition", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete metric definition")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Request types

type IngestMetricRequest struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Source    *string           `json:"source,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
}

type IngestBatchRequest struct {
	Metrics []IngestMetricRequest `json:"metrics"`
}

type CreateMetricDefinitionRequest struct {
	Name           string                 `json:"name"`
	DisplayName    *string                `json:"display_name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Unit           *string                `json:"unit,omitempty"`
	Type           string                 `json:"type,omitempty"`
	Aggregation    string                 `json:"aggregation,omitempty"`
	AlertThreshold *domain.AlertThreshold `json:"alert_threshold,omitempty"`
	RetentionDays  *int                   `json:"retention_days,omitempty"`
}

type UpdateMetricDefinitionRequest struct {
	DisplayName    *string                `json:"display_name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	Unit           *string                `json:"unit,omitempty"`
	Type           *string                `json:"type,omitempty"`
	Aggregation    *string                `json:"aggregation,omitempty"`
	AlertThreshold *domain.AlertThreshold `json:"alert_threshold,omitempty"`
	RetentionDays  *int                   `json:"retention_days,omitempty"`
}

func parseTimeRangeWithDefaults(r *http.Request, defaultDuration time.Duration) (start, end time.Time) {
	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}
	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = end.Add(-defaultDuration)
	}
	return start, end
}

