-- Alert rules table for defining when to trigger alerts and workflows
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,

    -- Condition: what triggers the rule
    condition_type VARCHAR(50) NOT NULL, -- 'metric_threshold', 'metric_anomaly', 'event_pattern'
    condition_config JSONB NOT NULL, -- {"metric_name": "cpu_usage", "operator": "gt", "threshold": 90}

    -- Alert configuration
    severity VARCHAR(20) NOT NULL DEFAULT 'warning', -- info, warning, critical
    alert_title_template VARCHAR(255) NOT NULL, -- "High CPU: ${metric_name} at ${value}%"
    alert_message_template TEXT,

    -- Auto-remediation: workflow to trigger
    trigger_workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    trigger_input_template JSONB, -- Template for workflow input, can reference alert data

    -- Cooldown to prevent alert storms
    cooldown_seconds INT NOT NULL DEFAULT 300, -- 5 minutes default
    last_triggered_at TIMESTAMPTZ,

    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_alert_rules_tenant_id ON alert_rules(tenant_id);
CREATE INDEX idx_alert_rules_enabled ON alert_rules(tenant_id, enabled);
CREATE INDEX idx_alert_rules_condition_type ON alert_rules(condition_type);

-- Updated_at trigger
CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add trigger_workflow_id to alerts table
ALTER TABLE alerts ADD COLUMN triggered_by_rule_id UUID REFERENCES alert_rules(id) ON DELETE SET NULL;
ALTER TABLE alerts ADD COLUMN triggered_workflow_execution_id UUID REFERENCES executions(id) ON DELETE SET NULL;
