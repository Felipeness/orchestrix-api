package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/auth"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// Handler handles metrics HTTP requests
type Handler struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

// NewHandler creates a new metrics handler
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{
		queries: db.New(pool),
		pool:    pool,
	}
}

// Routes registers metrics routes
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Metric data ingestion
	r.Post("/ingest", h.Ingest)
	r.Post("/ingest/batch", h.IngestBatch)

	// Metric querying
	r.Get("/", h.Query)
	r.Get("/names", h.ListNames)
	r.Get("/latest/{name}", h.GetLatest)
	r.Get("/aggregate/{name}", h.GetAggregate)

	// Metric definitions
	r.Get("/definitions", h.ListDefinitions)
	r.Post("/definitions", h.CreateDefinition)
	r.Get("/definitions/{name}", h.GetDefinition)
	r.Delete("/definitions/{name}", h.DeleteDefinition)

	return r
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Source    string            `json:"source,omitempty"`
	Timestamp *time.Time        `json:"timestamp,omitempty"`
}

// IngestRequest represents a single metric ingestion request
type IngestRequest struct {
	MetricPoint
}

// IngestBatchRequest represents a batch metric ingestion request
type IngestBatchRequest struct {
	Metrics []MetricPoint `json:"metrics"`
}

// AggregateResponse represents aggregated metric data
type AggregateResponse struct {
	Count   int64   `json:"count"`
	Average float64 `json:"avg"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Sum     float64 `json:"sum"`
}

// DefinitionRequest represents a metric definition request
type DefinitionRequest struct {
	Name           string          `json:"name"`
	DisplayName    string          `json:"display_name,omitempty"`
	Description    string          `json:"description,omitempty"`
	Unit           string          `json:"unit,omitempty"`
	Type           string          `json:"type,omitempty"`
	Aggregation    string          `json:"aggregation,omitempty"`
	AlertThreshold json.RawMessage `json:"alert_threshold,omitempty"`
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

// Ingest handles single metric ingestion
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	labelsJSON, _ := json.Marshal(req.Labels)

	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	metric, err := h.queries.InsertMetric(ctx, db.InsertMetricParams{
		TenantID:  user.TenantID,
		Name:      req.Name,
		Value:     req.Value,
		Labels:    labelsJSON,
		Source:    stringPtr(req.Source),
		Timestamp: timestamp,
	})
	if err != nil {
		slog.Error("failed to insert metric", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to insert metric")
		return
	}

	slog.Info("metric ingested", "name", req.Name, "value", req.Value)
	respondJSON(w, http.StatusCreated, map[string]interface{}{"data": metric})
}

// IngestBatch handles batch metric ingestion
func (h *Handler) IngestBatch(w http.ResponseWriter, r *http.Request) {
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
		respondError(w, http.StatusBadRequest, "at least one metric is required")
		return
	}

	inserted := 0
	for _, m := range req.Metrics {
		if m.Name == "" {
			continue
		}

		labelsJSON, _ := json.Marshal(m.Labels)

		timestamp := time.Now()
		if m.Timestamp != nil {
			timestamp = *m.Timestamp
		}

		_, err := h.queries.InsertMetric(ctx, db.InsertMetricParams{
			TenantID:  user.TenantID,
			Name:      m.Name,
			Value:     m.Value,
			Labels:    labelsJSON,
			Source:    stringPtr(m.Source),
			Timestamp: timestamp,
		})
		if err != nil {
			slog.Error("failed to insert metric", "name", m.Name, "error", err)
			continue
		}
		inserted++
	}

	slog.Info("batch metrics ingested", "total", len(req.Metrics), "inserted", inserted)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"inserted": inserted,
		"total":    len(req.Metrics),
	})
}

// Query queries metrics
func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = t
		}
	}

	limit := int32(1000)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 10000 {
			limit = int32(parsed)
		}
	}

	metrics, err := h.queries.GetMetrics(ctx, db.GetMetricsParams{
		TenantID:    user.TenantID,
		Name:        name,
		Timestamp:   startTime,
		Timestamp_2: endTime,
		Limit:       limit,
	})
	if err != nil {
		slog.Error("failed to query metrics", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to query metrics")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":       metrics,
		"name":       name,
		"start_time": startTime,
		"end_time":   endTime,
		"count":      len(metrics),
	})
}

// ListNames lists all metric names for a tenant
func (h *Handler) ListNames(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	names, err := h.queries.GetMetricNames(ctx, user.TenantID)
	if err != nil {
		slog.Error("failed to list metric names", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list metric names")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": names})
}

// GetLatest gets the latest metric value
func (h *Handler) GetLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	metric, err := h.queries.GetLatestMetric(ctx, db.GetLatestMetricParams{
		TenantID: user.TenantID,
		Name:     name,
	})
	if err != nil {
		slog.Error("failed to get latest metric", "error", err)
		respondError(w, http.StatusNotFound, "metric not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": metric})
}

// GetAggregate gets aggregated metric values
func (h *Handler) GetAggregate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = t
		}
	}

	agg, err := h.queries.GetMetricsAggregate(ctx, db.GetMetricsAggregateParams{
		TenantID:    user.TenantID,
		Name:        name,
		Timestamp:   startTime,
		Timestamp_2: endTime,
	})
	if err != nil {
		slog.Error("failed to get metric aggregate", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get metric aggregate")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data": AggregateResponse{
			Count:   agg.Count,
			Average: floatFromInterface(agg.AvgValue),
			Min:     floatFromInterface(agg.MinValue),
			Max:     floatFromInterface(agg.MaxValue),
			Sum:     floatFromInterface(agg.SumValue),
		},
		"name":       name,
		"start_time": startTime,
		"end_time":   endTime,
	})
}

// ListDefinitions lists all metric definitions
func (h *Handler) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	definitions, err := h.queries.ListMetricDefinitions(ctx, user.TenantID)
	if err != nil {
		slog.Error("failed to list metric definitions", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to list metric definitions")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": definitions})
}

// CreateDefinition creates a new metric definition
func (h *Handler) CreateDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req DefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	if req.Type == "" {
		req.Type = "gauge"
	}
	if req.Aggregation == "" {
		req.Aggregation = "avg"
	}

	definition, err := h.queries.CreateMetricDefinition(ctx, db.CreateMetricDefinitionParams{
		TenantID:       user.TenantID,
		Name:           req.Name,
		DisplayName:    stringPtr(req.DisplayName),
		Description:    stringPtr(req.Description),
		Unit:           stringPtr(req.Unit),
		Type:           req.Type,
		Aggregation:    stringPtr(req.Aggregation),
		AlertThreshold: req.AlertThreshold,
	})
	if err != nil {
		slog.Error("failed to create metric definition", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to create metric definition")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{"data": definition})
}

// GetDefinition gets a metric definition by name
func (h *Handler) GetDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	definition, err := h.queries.GetMetricDefinition(ctx, db.GetMetricDefinitionParams{
		TenantID: user.TenantID,
		Name:     name,
	})
	if err != nil {
		slog.Error("failed to get metric definition", "error", err)
		respondError(w, http.StatusNotFound, "metric definition not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"data": definition})
}

// DeleteDefinition deletes a metric definition
func (h *Handler) DeleteDefinition(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.FromContext(ctx)
	if user == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "metric name is required")
		return
	}

	err := h.queries.DeleteMetricDefinition(ctx, db.DeleteMetricDefinitionParams{
		TenantID: user.TenantID,
		Name:     name,
	})
	if err != nil {
		slog.Error("failed to delete metric definition", "error", err)
		respondError(w, http.StatusInternalServerError, "failed to delete metric definition")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func floatFromInterface(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case *float64:
		if val != nil {
			return *val
		}
		return 0
	default:
		return 0
	}
}
