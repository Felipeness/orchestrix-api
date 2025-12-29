-- name: InsertMetric :one
INSERT INTO metrics (tenant_id, name, value, labels, source, timestamp)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: InsertMetricsBatch :copyfrom
INSERT INTO metrics (tenant_id, name, value, labels, source, timestamp)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetMetrics :many
SELECT * FROM metrics
WHERE tenant_id = $1
    AND name = $2
    AND timestamp >= $3
    AND timestamp <= $4
ORDER BY timestamp DESC
LIMIT $5 OFFSET $6;

-- name: CountMetrics :one
SELECT COUNT(*) FROM metrics
WHERE tenant_id = $1
    AND name = $2
    AND timestamp >= $3
    AND timestamp <= $4;

-- name: GetLatestMetric :one
SELECT * FROM metrics
WHERE tenant_id = $1 AND name = $2
ORDER BY timestamp DESC
LIMIT 1;

-- name: GetMetricNames :many
SELECT DISTINCT name FROM metrics
WHERE tenant_id = $1
ORDER BY name;

-- name: GetMetricNamesWithPrefix :many
SELECT DISTINCT name FROM metrics
WHERE tenant_id = $1
    AND name LIKE $2 || '%'
ORDER BY name
LIMIT 100;

-- name: GetMetricsAggregate :one
SELECT
    COUNT(*) as count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    SUM(value) as sum_value
FROM metrics
WHERE tenant_id = $1
    AND name = $2
    AND timestamp >= $3
    AND timestamp <= $4;

-- name: GetMetricsByLabels :many
SELECT * FROM metrics
WHERE tenant_id = $1
    AND name = $2
    AND labels @> $3
    AND timestamp >= $4
    AND timestamp <= $5
ORDER BY timestamp DESC
LIMIT $6;

-- name: DeleteOldMetrics :exec
DELETE FROM metrics
WHERE tenant_id = $1 AND timestamp < $2;

-- TimescaleDB Time Bucket Queries
-- name: GetMetricsSeries :many
SELECT
    time_bucket($1::interval, timestamp) as bucket,
    COUNT(*) as count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    SUM(value) as sum_value
FROM metrics
WHERE tenant_id = $2
    AND name = $3
    AND timestamp >= $4
    AND timestamp <= $5
GROUP BY bucket
ORDER BY bucket DESC;

-- name: GetMetricsSeriesWithLabels :many
SELECT
    time_bucket($1::interval, timestamp) as bucket,
    COUNT(*) as count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    SUM(value) as sum_value
FROM metrics
WHERE tenant_id = $2
    AND name = $3
    AND labels @> $4
    AND timestamp >= $5
    AND timestamp <= $6
GROUP BY bucket
ORDER BY bucket DESC;

-- name: GetMetricsAggregateWithPercentiles :one
SELECT
    COUNT(*) as count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    SUM(value) as sum_value,
    approx_percentile(0.50, percentile_agg(value)) as p50,
    approx_percentile(0.95, percentile_agg(value)) as p95,
    approx_percentile(0.99, percentile_agg(value)) as p99
FROM metrics
WHERE tenant_id = $1
    AND name = $2
    AND timestamp >= $3
    AND timestamp <= $4;

-- Hourly Pre-aggregated Data (from continuous aggregate)
-- name: GetMetricsHourly :many
SELECT * FROM metrics_hourly
WHERE tenant_id = $1
    AND name = $2
    AND bucket >= $3
    AND bucket <= $4
ORDER BY bucket DESC;

-- Daily Pre-aggregated Data (from continuous aggregate)
-- name: GetMetricsDaily :many
SELECT * FROM metrics_daily
WHERE tenant_id = $1
    AND name = $2
    AND bucket >= $3
    AND bucket <= $4
ORDER BY bucket DESC;

-- Metric Definitions
-- name: CreateMetricDefinition :one
INSERT INTO metric_definitions (tenant_id, name, display_name, description, unit, type, aggregation, alert_threshold, retention_days)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateMetricDefinition :one
UPDATE metric_definitions SET
    display_name = COALESCE($3, display_name),
    description = COALESCE($4, description),
    unit = COALESCE($5, unit),
    type = COALESCE($6, type),
    aggregation = COALESCE($7, aggregation),
    alert_threshold = COALESCE($8, alert_threshold),
    retention_days = COALESCE($9, retention_days),
    updated_at = NOW()
WHERE tenant_id = $1 AND name = $2
RETURNING *;

-- name: GetMetricDefinition :one
SELECT * FROM metric_definitions
WHERE tenant_id = $1 AND name = $2;

-- name: ListMetricDefinitions :many
SELECT * FROM metric_definitions
WHERE tenant_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: CountMetricDefinitions :one
SELECT COUNT(*) FROM metric_definitions
WHERE tenant_id = $1;

-- name: DeleteMetricDefinition :exec
DELETE FROM metric_definitions
WHERE tenant_id = $1 AND name = $2;
