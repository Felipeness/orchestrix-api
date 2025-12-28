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
LIMIT $5;

-- name: GetLatestMetric :one
SELECT * FROM metrics
WHERE tenant_id = $1 AND name = $2
ORDER BY timestamp DESC
LIMIT 1;

-- name: GetMetricNames :many
SELECT DISTINCT name FROM metrics
WHERE tenant_id = $1
ORDER BY name;

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

-- Metric Definitions
-- name: CreateMetricDefinition :one
INSERT INTO metric_definitions (tenant_id, name, display_name, description, unit, type, aggregation, alert_threshold)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (tenant_id, name)
DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    unit = EXCLUDED.unit,
    type = EXCLUDED.type,
    aggregation = EXCLUDED.aggregation,
    alert_threshold = EXCLUDED.alert_threshold
RETURNING *;

-- name: GetMetricDefinition :one
SELECT * FROM metric_definitions
WHERE tenant_id = $1 AND name = $2;

-- name: ListMetricDefinitions :many
SELECT * FROM metric_definitions
WHERE tenant_id = $1
ORDER BY name;

-- name: DeleteMetricDefinition :exec
DELETE FROM metric_definitions
WHERE tenant_id = $1 AND name = $2;
