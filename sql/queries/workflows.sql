-- name: GetWorkflow :one
SELECT * FROM workflows WHERE id = $1;

-- name: ListWorkflows :many
SELECT * FROM workflows
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListWorkflowsByStatus :many
SELECT * FROM workflows
WHERE tenant_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CreateWorkflow :one
INSERT INTO workflows (tenant_id, name, description, definition, schedule, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateWorkflow :one
UPDATE workflows
SET name = $2, description = $3, definition = $4, schedule = $5, status = $6, version = version + 1
WHERE id = $1
RETURNING *;

-- name: UpdateWorkflowStatus :one
UPDATE workflows
SET status = $2
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflow :exec
DELETE FROM workflows WHERE id = $1;

-- name: CountWorkflows :one
SELECT COUNT(*) FROM workflows WHERE tenant_id = $1;

-- name: CountWorkflowsByStatus :one
SELECT COUNT(*) FROM workflows WHERE tenant_id = $1 AND status = $2;

-- name: GetScheduledWorkflows :many
SELECT * FROM workflows
WHERE status = 'active' AND schedule IS NOT NULL
ORDER BY created_at;
