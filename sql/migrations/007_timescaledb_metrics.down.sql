-- Remove compression policy
SELECT remove_compression_policy('metrics', if_exists => TRUE);

-- Disable compression
ALTER TABLE metrics SET (timescaledb.compress = FALSE);

-- Remove continuous aggregate policies
SELECT remove_continuous_aggregate_policy('metrics_daily', if_not_exists => TRUE);
SELECT remove_continuous_aggregate_policy('metrics_hourly', if_not_exists => TRUE);

-- Drop continuous aggregates
DROP MATERIALIZED VIEW IF EXISTS metrics_daily;
DROP MATERIALIZED VIEW IF EXISTS metrics_hourly;

-- Remove retention_days column
ALTER TABLE metric_definitions DROP COLUMN IF EXISTS retention_days;

-- Note: Cannot easily revert hypertable back to regular table
-- The metrics table will remain a hypertable
-- To fully revert, you would need to:
-- 1. Backup data
-- 2. Drop and recreate the metrics table
-- 3. Restore data
