-- name: CreateAuditLog :one
INSERT INTO audit_logs (tenant_id, user_id, event_type, resource_type, resource_id, action, old_value, new_value, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByResource :many
SELECT * FROM audit_logs
WHERE tenant_id = $1 AND resource_type = $2 AND resource_id = $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: ListAuditLogsByUser :many
SELECT * FROM audit_logs
WHERE tenant_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsByEventType :many
SELECT * FROM audit_logs
WHERE tenant_id = $1 AND event_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsByDateRange :many
SELECT * FROM audit_logs
WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountAuditLogsByTenant :one
SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1;
