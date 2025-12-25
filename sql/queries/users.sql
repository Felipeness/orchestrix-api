-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByExternalID :one
SELECT * FROM users WHERE external_id = $1 AND tenant_id = $2;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND tenant_id = $2;

-- name: ListUsers :many
SELECT * FROM users
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateUser :one
INSERT INTO users (tenant_id, external_id, email, name, roles)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = $2, name = $3, roles = $4
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: CountUsersByTenant :one
SELECT COUNT(*) FROM users WHERE tenant_id = $1;

-- name: UpsertUser :one
INSERT INTO users (tenant_id, external_id, email, name, roles)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tenant_id, external_id)
DO UPDATE SET email = $3, name = $4, roles = $5, updated_at = NOW()
RETURNING *;
