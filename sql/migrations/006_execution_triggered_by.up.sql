-- Add triggered_by column to track what triggered an execution
ALTER TABLE executions ADD COLUMN IF NOT EXISTS triggered_by VARCHAR(255);

-- Index for filtering by trigger source
CREATE INDEX IF NOT EXISTS idx_executions_triggered_by ON executions(triggered_by);
