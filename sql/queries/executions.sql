-- name: GetExecution :one
SELECT * FROM executions WHERE id = $1;

-- name: GetExecutionByTemporalID :one
SELECT * FROM executions WHERE temporal_workflow_id = $1;

-- name: ListExecutions :many
SELECT * FROM executions
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListExecutionsByWorkflow :many
SELECT * FROM executions
WHERE workflow_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListExecutionsByStatus :many
SELECT * FROM executions
WHERE tenant_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListRecentExecutions :many
SELECT * FROM executions
WHERE tenant_id = $1 AND created_at > $2
ORDER BY created_at DESC
LIMIT $3;

-- name: CreateExecution :one
INSERT INTO executions (tenant_id, workflow_id, temporal_workflow_id, status, input, triggered_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateExecutionStatus :exec
UPDATE executions
SET status = $2, error = $3
WHERE id = $1;

-- name: UpdateExecutionRunID :exec
UPDATE executions
SET temporal_run_id = $2, status = $3, started_at = $4
WHERE id = $1;

-- name: CompleteExecution :one
UPDATE executions
SET status = $2, output = $3, completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: FailExecution :one
UPDATE executions
SET status = 'failed', error = $2, completed_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountExecutions :one
SELECT COUNT(*) FROM executions WHERE tenant_id = $1;

-- name: UpdateExecutionTemporalIDs :exec
UPDATE executions
SET temporal_workflow_id = $2, temporal_run_id = $3
WHERE id = $1;

-- name: CountExecutionsByStatus :one
SELECT COUNT(*) FROM executions WHERE tenant_id = $1 AND status = $2;

-- name: GetExecutionStats :one
SELECT
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'completed') as completed,
    COUNT(*) FILTER (WHERE status = 'failed') as failed,
    COUNT(*) FILTER (WHERE status = 'running') as running,
    COUNT(*) FILTER (WHERE status = 'pending') as pending,
    AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) FILTER (WHERE completed_at IS NOT NULL) as avg_duration_seconds
FROM executions
WHERE tenant_id = $1 AND created_at > $2;
