-- name: CreateAlertRule :one
INSERT INTO alert_rules (
    tenant_id, name, description, enabled,
    condition_type, condition_config,
    severity, alert_title_template, alert_message_template,
    trigger_workflow_id, trigger_input_template,
    cooldown_seconds, created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetAlertRule :one
SELECT * FROM alert_rules
WHERE id = $1 AND tenant_id = $2;

-- name: ListAlertRules :many
SELECT * FROM alert_rules
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListEnabledAlertRules :many
SELECT * FROM alert_rules
WHERE tenant_id = $1 AND enabled = true
ORDER BY name;

-- name: ListAlertRulesByConditionType :many
SELECT * FROM alert_rules
WHERE tenant_id = $1 AND condition_type = $2 AND enabled = true;

-- name: UpdateAlertRule :one
UPDATE alert_rules
SET
    name = COALESCE($3, name),
    description = COALESCE($4, description),
    enabled = COALESCE($5, enabled),
    condition_type = COALESCE($6, condition_type),
    condition_config = COALESCE($7, condition_config),
    severity = COALESCE($8, severity),
    alert_title_template = COALESCE($9, alert_title_template),
    alert_message_template = COALESCE($10, alert_message_template),
    trigger_workflow_id = $11,
    trigger_input_template = $12,
    cooldown_seconds = COALESCE($13, cooldown_seconds)
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: UpdateAlertRuleLastTriggered :exec
UPDATE alert_rules
SET last_triggered_at = NOW()
WHERE id = $1;

-- name: DeleteAlertRule :exec
DELETE FROM alert_rules
WHERE id = $1 AND tenant_id = $2;

-- name: CountAlertRules :one
SELECT COUNT(*) FROM alert_rules
WHERE tenant_id = $1;

-- name: GetAlertRulesForMetric :many
SELECT * FROM alert_rules
WHERE tenant_id = $1
    AND enabled = true
    AND condition_type = 'metric_threshold'
    AND condition_config->>'metric_name' = $2
    AND (last_triggered_at IS NULL OR last_triggered_at < NOW() - (cooldown_seconds || ' seconds')::interval);
