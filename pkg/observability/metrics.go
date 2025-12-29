package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all application metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Workflow metrics
	WorkflowExecutionsTotal   *prometheus.CounterVec
	WorkflowExecutionDuration *prometheus.HistogramVec
	WorkflowsActive           prometheus.Gauge

	// Alert metrics
	AlertsTotal     *prometheus.CounterVec
	AlertsActive    prometheus.Gauge
	AlertsTriggered *prometheus.CounterVec

	// Database metrics
	DBQueriesTotal    *prometheus.CounterVec
	DBQueryDuration   *prometheus.HistogramVec
	DBConnectionsOpen prometheus.Gauge
}

// metrics is the global metrics instance
var metrics *Metrics

// InitMetrics initializes Prometheus metrics
func InitMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "orchestrix"
	}

	metrics = &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
		),

		// Workflow metrics
		WorkflowExecutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "workflow",
				Name:      "executions_total",
				Help:      "Total number of workflow executions",
			},
			[]string{"workflow_id", "status"},
		),
		WorkflowExecutionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "workflow",
				Name:      "execution_duration_seconds",
				Help:      "Workflow execution duration in seconds",
				Buckets:   []float64{.1, .5, 1, 2, 5, 10, 30, 60, 120, 300},
			},
			[]string{"workflow_id"},
		),
		WorkflowsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "workflow",
				Name:      "active",
				Help:      "Number of currently running workflows",
			},
		),

		// Alert metrics
		AlertsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "alert",
				Name:      "total",
				Help:      "Total number of alerts created",
			},
			[]string{"severity"},
		),
		AlertsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "alert",
				Name:      "active",
				Help:      "Number of currently active alerts",
			},
		),
		AlertsTriggered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "alert",
				Name:      "triggered_total",
				Help:      "Total number of alerts triggered by rules",
			},
			[]string{"rule_id", "severity"},
		),

		// Database metrics
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "table"},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),
		DBConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "connections_open",
				Help:      "Number of open database connections",
			},
		),
	}

	return metrics
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	if metrics == nil {
		return InitMetrics("")
	}
	return metrics
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
