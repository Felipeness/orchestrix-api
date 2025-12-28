-- Metrics table for time-series data
CREATE TABLE metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    labels JSONB DEFAULT '{}',
    source VARCHAR(255), -- agent/service that sent the metric
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Metric definitions (schema/metadata for metrics)
CREATE TABLE metric_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    unit VARCHAR(50), -- bytes, seconds, percent, count, etc.
    type VARCHAR(50) NOT NULL DEFAULT 'gauge', -- gauge, counter, histogram
    aggregation VARCHAR(50) DEFAULT 'avg', -- avg, sum, min, max, last
    alert_threshold JSONB, -- {"warning": 80, "critical": 95}
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Indexes for high-performance time-series queries
CREATE INDEX idx_metrics_tenant_id ON metrics(tenant_id);
CREATE INDEX idx_metrics_name ON metrics(name);
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp DESC);
CREATE INDEX idx_metrics_tenant_name_time ON metrics(tenant_id, name, timestamp DESC);

-- Composite index for common query patterns
CREATE INDEX idx_metrics_lookup ON metrics(tenant_id, name, timestamp DESC)
    INCLUDE (value, labels);

-- GIN index for label filtering
CREATE INDEX idx_metrics_labels ON metrics USING GIN (labels);

-- Index for metric definitions
CREATE INDEX idx_metric_definitions_tenant_id ON metric_definitions(tenant_id);

-- Apply updated_at trigger to metric_definitions
CREATE TRIGGER update_metric_definitions_updated_at BEFORE UPDATE ON metric_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Partitioning hint: For production, consider partitioning metrics table by timestamp
-- CREATE TABLE metrics (...) PARTITION BY RANGE (timestamp);
