DROP INDEX IF EXISTS idx_executions_triggered_by;
ALTER TABLE executions DROP COLUMN IF EXISTS triggered_by;
