-- Add additional fields to alerts table for better context
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS source VARCHAR(255);
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}';

-- Index for filtering by source
CREATE INDEX IF NOT EXISTS idx_alerts_source ON alerts(tenant_id, source);
