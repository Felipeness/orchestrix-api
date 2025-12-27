-- Enable Row Level Security for multi-tenancy

-- Enable RLS on tables (FORCE ensures RLS applies even to table owner)
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflows FORCE ROW LEVEL SECURITY;
ALTER TABLE executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE executions FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;
ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts FORCE ROW LEVEL SECURITY;

-- Create app user for RLS (not superuser)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'orchestrix_app') THEN
        CREATE ROLE orchestrix_app WITH LOGIN PASSWORD 'orchestrix_app_password';
    END IF;
END
$$;

-- Grant permissions to app user
GRANT USAGE ON SCHEMA public TO orchestrix_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO orchestrix_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO orchestrix_app;

-- RLS Policies for users table
CREATE POLICY tenant_isolation_users ON users
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- RLS Policies for workflows table
CREATE POLICY tenant_isolation_workflows ON workflows
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- RLS Policies for executions table
CREATE POLICY tenant_isolation_executions ON executions
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- RLS Policies for audit_logs table
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- RLS Policies for alerts table
CREATE POLICY tenant_isolation_alerts ON alerts
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Tenants table doesn't need RLS (managed by superuser or specific queries)
-- But we can add a policy for tenant admins to only see their own tenant
CREATE POLICY tenant_isolation_tenants ON tenants
    FOR SELECT
    USING (id = current_setting('app.current_tenant_id', true)::uuid);

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;

-- Superuser bypass policies (for migrations and admin operations)
-- These allow superusers to access all data when needed
CREATE POLICY superuser_bypass_users ON users
    FOR ALL TO postgres
    USING (true) WITH CHECK (true);

CREATE POLICY superuser_bypass_workflows ON workflows
    FOR ALL TO postgres
    USING (true) WITH CHECK (true);

CREATE POLICY superuser_bypass_executions ON executions
    FOR ALL TO postgres
    USING (true) WITH CHECK (true);

CREATE POLICY superuser_bypass_audit_logs ON audit_logs
    FOR ALL TO postgres
    USING (true) WITH CHECK (true);

CREATE POLICY superuser_bypass_alerts ON alerts
    FOR ALL TO postgres
    USING (true) WITH CHECK (true);

CREATE POLICY superuser_bypass_tenants ON tenants
    FOR ALL TO postgres
    USING (true) WITH CHECK (true);

-- Function to set tenant context
CREATE OR REPLACE FUNCTION set_tenant_context(p_tenant_id UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::text, false);
END;
$$ LANGUAGE plpgsql;

-- Function to get current tenant
CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true)::uuid;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;
