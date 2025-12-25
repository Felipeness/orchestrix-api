-- name: GetTenant :one
SELECT * FROM tenants WHERE id = $1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1;

-- name: ListTenants :many
SELECT * FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CreateTenant :one
INSERT INTO tenants (name, slug, plan, settings)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateTenant :one
UPDATE tenants
SET name = $2, plan = $3, settings = $4
WHERE id = $1
RETURNING *;

-- name: DeleteTenant :exec
DELETE FROM tenants WHERE id = $1;

-- name: CountTenants :one
SELECT COUNT(*) FROM tenants;
