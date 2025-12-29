-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Convert metrics table to hypertable
-- Partitions by timestamp with 1-day chunks
-- Note: This will fail if there's existing data without proper handling
-- For production, backup data first
SELECT create_hypertable(
    'metrics',
    'timestamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE,
    migrate_data => TRUE
);

-- Add retention_days column to metric_definitions (default 30 days)
ALTER TABLE metric_definitions ADD COLUMN IF NOT EXISTS retention_days INTEGER DEFAULT 30;

-- Create continuous aggregate for hourly metrics rollup
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_hourly
WITH (timescaledb.continuous) AS
SELECT
    tenant_id,
    name,
    time_bucket('1 hour', timestamp) AS bucket,
    COUNT(*) as count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    SUM(value) as sum_value,
    approx_percentile(0.50, percentile_agg(value)) as p50,
    approx_percentile(0.95, percentile_agg(value)) as p95,
    approx_percentile(0.99, percentile_agg(value)) as p99
FROM metrics
GROUP BY tenant_id, name, time_bucket('1 hour', timestamp)
WITH NO DATA;

-- Create refresh policy for continuous aggregate (refresh every 30 minutes)
SELECT add_continuous_aggregate_policy('metrics_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '30 minutes',
    if_not_exists => TRUE
);

-- Create continuous aggregate for daily metrics rollup
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics_daily
WITH (timescaledb.continuous) AS
SELECT
    tenant_id,
    name,
    time_bucket('1 day', timestamp) AS bucket,
    COUNT(*) as count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    SUM(value) as sum_value
FROM metrics
GROUP BY tenant_id, name, time_bucket('1 day', timestamp)
WITH NO DATA;

-- Create refresh policy for daily aggregate (refresh every 6 hours)
SELECT add_continuous_aggregate_policy('metrics_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '6 hours',
    if_not_exists => TRUE
);

-- Enable compression on the hypertable (compress chunks older than 7 days)
ALTER TABLE metrics SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'tenant_id, name'
);

SELECT add_compression_policy('metrics', INTERVAL '7 days', if_not_exists => TRUE);

-- Note: Retention policies should be managed per-tenant via application logic
-- Default retention: 30 days (configured in metric_definitions.retention_days)
-- Example policy (commented out - use application-managed retention instead):
-- SELECT add_retention_policy('metrics', INTERVAL '30 days', if_not_exists => TRUE);

-- Index on continuous aggregates for fast lookups
CREATE INDEX IF NOT EXISTS idx_metrics_hourly_lookup
ON metrics_hourly (tenant_id, name, bucket DESC);

CREATE INDEX IF NOT EXISTS idx_metrics_daily_lookup
ON metrics_daily (tenant_id, name, bucket DESC);
