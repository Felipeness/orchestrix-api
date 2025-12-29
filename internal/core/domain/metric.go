package domain

import (
	"time"

	"github.com/google/uuid"
)

// Metric represents a single metric data point
type Metric struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Value     float64
	Labels    map[string]string
	Source    *string
	Timestamp time.Time
	CreatedAt time.Time
}

// IsValid checks if the metric has required fields
func (m *Metric) IsValid() bool {
	return m.Name != "" && m.TenantID != uuid.Nil
}

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// IsValid checks if the metric type is valid
func (t MetricType) IsValid() bool {
	switch t {
	case MetricTypeGauge, MetricTypeCounter, MetricTypeHistogram, MetricTypeSummary:
		return true
	default:
		return false
	}
}

// AggregationType defines how to aggregate metrics
type AggregationType string

const (
	AggregationAvg   AggregationType = "avg"
	AggregationSum   AggregationType = "sum"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationLast  AggregationType = "last"
	AggregationCount AggregationType = "count"
	AggregationP50   AggregationType = "p50"
	AggregationP95   AggregationType = "p95"
	AggregationP99   AggregationType = "p99"
)

// IsValid checks if the aggregation type is valid
func (a AggregationType) IsValid() bool {
	switch a {
	case AggregationAvg, AggregationSum, AggregationMin, AggregationMax,
		AggregationLast, AggregationCount, AggregationP50, AggregationP95, AggregationP99:
		return true
	default:
		return false
	}
}

// MetricDefinition represents metadata about a metric
type MetricDefinition struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	Name           string
	DisplayName    *string
	Description    *string
	Unit           *string
	Type           MetricType
	Aggregation    AggregationType
	AlertThreshold *AlertThreshold
	RetentionDays  int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AlertThreshold defines threshold values for alerting
type AlertThreshold struct {
	Warning  *float64 `json:"warning,omitempty"`
	Critical *float64 `json:"critical,omitempty"`
}

// MetricAggregate represents aggregated metric results
type MetricAggregate struct {
	Count   int64
	Average float64
	Min     float64
	Max     float64
	Sum     float64
	P50     *float64
	P95     *float64
	P99     *float64
}

// TimeBucket represents a time-bucketed aggregation
type TimeBucket struct {
	Bucket    time.Time
	Count     int64
	Average   float64
	Min       float64
	Max       float64
	Sum       float64
}

// MetricQuery represents a query for metrics
type MetricQuery struct {
	TenantID    uuid.UUID
	Name        string
	Labels      map[string]string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
	BucketSize  *time.Duration
	Aggregation *AggregationType
}

// Validate checks if the query has required fields
func (q *MetricQuery) Validate() error {
	if q.TenantID == uuid.Nil {
		return ErrInvalidMetricQuery
	}
	if q.Name == "" {
		return ErrInvalidMetricName
	}
	if !q.StartTime.IsZero() && !q.EndTime.IsZero() && q.StartTime.After(q.EndTime) {
		return ErrInvalidTimeRange
	}
	return nil
}
