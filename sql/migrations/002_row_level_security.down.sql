-- Drop functions
DROP FUNCTION IF EXISTS set_tenant_context;
DROP FUNCTION IF EXISTS get_current_tenant_id;

-- Drop policies
DROP POLICY IF EXISTS tenant_isolation_users ON users;
DROP POLICY IF EXISTS tenant_isolation_workflows ON workflows;
DROP POLICY IF EXISTS tenant_isolation_executions ON executions;
DROP POLICY IF EXISTS tenant_isolation_audit_logs ON audit_logs;
DROP POLICY IF EXISTS tenant_isolation_alerts ON alerts;
DROP POLICY IF EXISTS tenant_isolation_tenants ON tenants;
DROP POLICY IF EXISTS superuser_bypass_users ON users;
DROP POLICY IF EXISTS superuser_bypass_workflows ON workflows;
DROP POLICY IF EXISTS superuser_bypass_executions ON executions;
DROP POLICY IF EXISTS superuser_bypass_audit_logs ON audit_logs;
DROP POLICY IF EXISTS superuser_bypass_alerts ON alerts;
DROP POLICY IF EXISTS superuser_bypass_tenants ON tenants;

-- Disable RLS and remove FORCE
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE workflows NO FORCE ROW LEVEL SECURITY;
ALTER TABLE workflows DISABLE ROW LEVEL SECURITY;
ALTER TABLE executions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE executions DISABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE alerts NO FORCE ROW LEVEL SECURITY;
ALTER TABLE alerts DISABLE ROW LEVEL SECURITY;
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;

-- Revoke and drop app user
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM orchestrix_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM orchestrix_app;
REVOKE USAGE ON SCHEMA public FROM orchestrix_app;
DROP ROLE IF EXISTS orchestrix_app;
