ALTER TABLE alerts DROP COLUMN IF EXISTS triggered_workflow_execution_id;
ALTER TABLE alerts DROP COLUMN IF EXISTS triggered_by_rule_id;
DROP TRIGGER IF EXISTS update_alert_rules_updated_at ON alert_rules;
DROP TABLE IF EXISTS alert_rules;
