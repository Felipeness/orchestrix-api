-- name: GetAlert :one
SELECT * FROM alerts WHERE id = $1;

-- name: ListAlerts :many
SELECT * FROM alerts
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAlertsByStatus :many
SELECT * FROM alerts
WHERE tenant_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAlertsBySeverity :many
SELECT * FROM alerts
WHERE tenant_id = $1 AND severity = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListOpenAlerts :many
SELECT * FROM alerts
WHERE tenant_id = $1 AND status = 'open'
ORDER BY
    CASE severity
        WHEN 'critical' THEN 1
        WHEN 'warning' THEN 2
        WHEN 'info' THEN 3
    END,
    created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateAlert :one
INSERT INTO alerts (tenant_id, workflow_id, execution_id, severity, title, message, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: AcknowledgeAlert :one
UPDATE alerts
SET status = 'acknowledged', acknowledged_at = NOW(), acknowledged_by = $2
WHERE id = $1
RETURNING *;

-- name: ResolveAlert :one
UPDATE alerts
SET status = 'resolved', resolved_at = NOW(), resolved_by = $2
WHERE id = $1
RETURNING *;

-- name: CountAlerts :one
SELECT COUNT(*) FROM alerts WHERE tenant_id = $1;

-- name: CountAlertsByStatus :one
SELECT COUNT(*) FROM alerts WHERE tenant_id = $1 AND status = $2;

-- name: CountOpenAlertsBySeverity :many
SELECT severity, COUNT(*) as count
FROM alerts
WHERE tenant_id = $1 AND status = 'open'
GROUP BY severity;
