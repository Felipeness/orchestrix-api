package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/db"
)

// MetricRepository implements port.MetricRepository
type MetricRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// NewMetricRepository creates a new metric repository
func NewMetricRepository(pool *pgxpool.Pool) *MetricRepository {
	return &MetricRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

// Save saves a single metric
func (r *MetricRepository) Save(ctx context.Context, metric *domain.Metric) error {
	labels, err := json.Marshal(metric.Labels)
	if err != nil {
		return err
	}

	_, err = r.queries.InsertMetric(ctx, db.InsertMetricParams{
		TenantID:  metric.TenantID,
		Name:      metric.Name,
		Value:     metric.Value,
		Labels:    labels,
		Source:    metric.Source,
		Timestamp: metric.Timestamp,
	})
	return err
}

// SaveBatch saves multiple metrics using pgx CopyFrom for high throughput
func (r *MetricRepository) SaveBatch(ctx context.Context, metrics []*domain.Metric) (int, error) {
	if len(metrics) == 0 {
		return 0, nil
	}

	params := make([]db.InsertMetricsBatchParams, len(metrics))
	for i, m := range metrics {
		labels, err := json.Marshal(m.Labels)
		if err != nil {
			return 0, err
		}
		params[i] = db.InsertMetricsBatchParams{
			TenantID:  m.TenantID,
			Name:      m.Name,
			Value:     m.Value,
			Labels:    labels,
			Source:    m.Source,
			Timestamp: m.Timestamp,
		}
	}

	count, err := r.queries.InsertMetricsBatch(ctx, params)
	return int(count), err
}

// FindByQuery finds metrics matching the query
func (r *MetricRepository) FindByQuery(ctx context.Context, query domain.MetricQuery) ([]*domain.Metric, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	var rows []db.Metric
	var err error

	if len(query.Labels) > 0 {
		labels, marshalErr := json.Marshal(query.Labels)
		if marshalErr != nil {
			return nil, marshalErr
		}
		rows, err = r.queries.GetMetricsByLabels(ctx, db.GetMetricsByLabelsParams{
			TenantID:    query.TenantID,
			Name:        query.Name,
			Labels:      labels,
			Timestamp:   query.StartTime,
			Timestamp_2: query.EndTime,
			Limit:       int32(limit),
		})
	} else {
		rows, err = r.queries.GetMetrics(ctx, db.GetMetricsParams{
			TenantID:    query.TenantID,
			Name:        query.Name,
			Timestamp:   query.StartTime,
			Timestamp_2: query.EndTime,
			Limit:       int32(limit),
			Offset:      int32(query.Offset),
		})
	}

	if err != nil {
		return nil, err
	}

	metrics := make([]*domain.Metric, len(rows))
	for i, row := range rows {
		metrics[i] = r.toDomain(row)
	}
	return metrics, nil
}

// CountByQuery counts metrics matching the query
func (r *MetricRepository) CountByQuery(ctx context.Context, query domain.MetricQuery) (int64, error) {
	return r.queries.CountMetrics(ctx, db.CountMetricsParams{
		TenantID:    query.TenantID,
		Name:        query.Name,
		Timestamp:   query.StartTime,
		Timestamp_2: query.EndTime,
	})
}

// FindLatest finds the latest metric value for a given name and labels
func (r *MetricRepository) FindLatest(ctx context.Context, tenantID uuid.UUID, name string, labels map[string]string) (*domain.Metric, error) {
	row, err := r.queries.GetLatestMetric(ctx, db.GetLatestMetricParams{
		TenantID: tenantID,
		Name:     name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMetricNotFound
		}
		return nil, err
	}
	return r.toDomain(row), nil
}

// GetAggregate gets aggregated stats for metrics
func (r *MetricRepository) GetAggregate(ctx context.Context, query domain.MetricQuery) (*domain.MetricAggregate, error) {
	row, err := r.queries.GetMetricsAggregateWithPercentiles(ctx, db.GetMetricsAggregateWithPercentilesParams{
		TenantID:    query.TenantID,
		Name:        query.Name,
		Timestamp:   query.StartTime,
		Timestamp_2: query.EndTime,
	})
	if err != nil {
		return nil, err
	}

	result := &domain.MetricAggregate{
		Count:   row.Count,
		Average: row.AvgValue,
		Sum:     float64(row.SumValue),
	}

	// Handle interface{} types for min/max
	if min, ok := row.MinValue.(float64); ok {
		result.Min = min
	}
	if max, ok := row.MaxValue.(float64); ok {
		result.Max = max
	}

	// Handle percentiles
	if p50, ok := row.P50.(float64); ok {
		result.P50 = &p50
	}
	if p95, ok := row.P95.(float64); ok {
		result.P95 = &p95
	}
	if p99, ok := row.P99.(float64); ok {
		result.P99 = &p99
	}

	return result, nil
}

// GetSeries gets time-bucketed metric data
func (r *MetricRepository) GetSeries(ctx context.Context, query domain.MetricQuery, bucketSize time.Duration) ([]*domain.TimeBucket, error) {
	// Convert duration to pgtype.Interval
	interval := pgtype.Interval{
		Microseconds: bucketSize.Microseconds(),
		Valid:        true,
	}

	rows, err := r.queries.GetMetricsSeries(ctx, db.GetMetricsSeriesParams{
		Column1:     interval,
		TenantID:    query.TenantID,
		Name:        query.Name,
		Timestamp:   query.StartTime,
		Timestamp_2: query.EndTime,
	})
	if err != nil {
		return nil, err
	}

	buckets := make([]*domain.TimeBucket, len(rows))
	for i, row := range rows {
		bucket := &domain.TimeBucket{
			Count:   row.Count,
			Average: row.AvgValue,
			Sum:     float64(row.SumValue),
		}

		// Handle bucket timestamp
		if t, ok := row.Bucket.(time.Time); ok {
			bucket.Bucket = t
		}
		if min, ok := row.MinValue.(float64); ok {
			bucket.Min = min
		}
		if max, ok := row.MaxValue.(float64); ok {
			bucket.Max = max
		}

		buckets[i] = bucket
	}

	return buckets, nil
}

// ListNames lists distinct metric names
func (r *MetricRepository) ListNames(ctx context.Context, tenantID uuid.UUID, prefix string) ([]string, error) {
	if prefix == "" {
		return r.queries.GetMetricNames(ctx, tenantID)
	}
	return r.queries.GetMetricNamesWithPrefix(ctx, db.GetMetricNamesWithPrefixParams{
		TenantID: tenantID,
		Column2:  &prefix,
	})
}

// toDomain converts a db.Metric to domain.Metric
func (r *MetricRepository) toDomain(row db.Metric) *domain.Metric {
	var labels map[string]string
	if len(row.Labels) > 0 {
		if err := json.Unmarshal(row.Labels, &labels); err != nil {
			slog.Warn("failed to unmarshal metric labels", "metric_id", row.ID, "error", err)
		}
	}

	return &domain.Metric{
		ID:        row.ID,
		TenantID:  row.TenantID,
		Name:      row.Name,
		Value:     row.Value,
		Labels:    labels,
		Source:    row.Source,
		Timestamp: row.Timestamp,
		CreatedAt: row.CreatedAt,
	}
}
