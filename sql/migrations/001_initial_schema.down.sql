-- Drop triggers
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column;

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS workflows;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

-- Drop extension
DROP EXTENSION IF EXISTS "uuid-ossp";
