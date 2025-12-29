//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	pgadapter "github.com/orchestrix/orchestrix-api/internal/adapter/driven/postgres"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
)

// TestContext holds the test database and cleanup functions
type TestContext struct {
	Pool      *pgxpool.Pool
	Container testcontainers.Container
	Ctx       context.Context
}

// setupTestDB creates a test database container
func setupTestDB(t *testing.T) *TestContext {
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("orchestrix_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Run migrations
	err = runMigrations(ctx, pool)
	require.NoError(t, err)

	return &TestContext{
		Pool:      pool,
		Container: container,
		Ctx:       ctx,
	}
}

// runMigrations runs the database schema setup
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Create tables for testing
	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE IF NOT EXISTS tenants (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS workflows (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		tenant_id UUID NOT NULL REFERENCES tenants(id),
		name TEXT NOT NULL,
		description TEXT,
		status TEXT NOT NULL DEFAULT 'draft',
		version INTEGER NOT NULL DEFAULT 1,
		definition JSONB,
		trigger_config JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS executions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		tenant_id UUID NOT NULL REFERENCES tenants(id),
		workflow_id UUID NOT NULL REFERENCES workflows(id),
		temporal_workflow_id TEXT,
		temporal_run_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		input JSONB,
		output JSONB,
		error TEXT,
		triggered_by TEXT,
		started_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS alerts (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		tenant_id UUID NOT NULL REFERENCES tenants(id),
		workflow_id UUID REFERENCES workflows(id),
		execution_id UUID REFERENCES executions(id),
		severity TEXT NOT NULL DEFAULT 'warning',
		title TEXT NOT NULL,
		message TEXT,
		status TEXT NOT NULL DEFAULT 'triggered',
		acknowledged_at TIMESTAMPTZ,
		acknowledged_by UUID,
		resolved_at TIMESTAMPTZ,
		resolved_by UUID,
		triggered_by_rule_id UUID,
		triggered_workflow_execution_id UUID,
		source TEXT,
		metadata JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS alert_rules (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		tenant_id UUID NOT NULL REFERENCES tenants(id),
		name TEXT NOT NULL,
		description TEXT,
		enabled BOOLEAN NOT NULL DEFAULT true,
		condition_type TEXT NOT NULL,
		condition_config JSONB NOT NULL,
		severity TEXT NOT NULL DEFAULT 'warning',
		alert_title_template TEXT NOT NULL,
		alert_message_template TEXT,
		trigger_workflow_id UUID REFERENCES workflows(id),
		trigger_input_template JSONB,
		cooldown_seconds INTEGER NOT NULL DEFAULT 300,
		last_triggered_at TIMESTAMPTZ,
		created_by UUID,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		tenant_id UUID NOT NULL REFERENCES tenants(id),
		user_id UUID,
		event_type TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id UUID,
		action TEXT NOT NULL,
		old_value JSONB,
		new_value JSONB,
		ip_address TEXT,
		user_agent TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	`

	_, err := pool.Exec(ctx, schema)
	return err
}

// cleanup closes connections and terminates container
func (tc *TestContext) cleanup(t *testing.T) {
	tc.Pool.Close()
	if err := tc.Container.Terminate(tc.Ctx); err != nil {
		t.Logf("failed to terminate container: %v", err)
	}
}

// createTestTenant creates a tenant for testing
func createTestTenant(ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	tenantID := uuid.New()
	_, _ = pool.Exec(ctx, "INSERT INTO tenants (id, name) VALUES ($1, $2)", tenantID, "Test Tenant")
	return tenantID
}

func TestWorkflowRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := setupTestDB(t)
	defer tc.cleanup(t)

	repo := pgadapter.NewWorkflowRepository(tc.Pool)
	tenantID := createTestTenant(tc.Ctx, tc.Pool)

	t.Run("Create and Find Workflow", func(t *testing.T) {
		definition, _ := json.Marshal(map[string]interface{}{
			"steps": []map[string]interface{}{
				{"name": "step1", "type": "http"},
			},
		})

		workflow := &domain.Workflow{
			ID:         uuid.New(),
			TenantID:   tenantID,
			Name:       "Integration Test Workflow",
			Status:     domain.WorkflowStatusDraft,
			Version:    1,
			Definition: definition,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		err := repo.Save(tc.Ctx, workflow)
		require.NoError(t, err)

		found, err := repo.FindByID(tc.Ctx, workflow.ID)
		require.NoError(t, err)
		assert.Equal(t, workflow.Name, found.Name)
		assert.Equal(t, workflow.Status, found.Status)
	})

	t.Run("List Workflows by Tenant", func(t *testing.T) {
		workflows, err := repo.FindByTenant(tc.Ctx, tenantID, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(workflows), 1)
	})

	t.Run("Count Workflows by Tenant", func(t *testing.T) {
		count, err := repo.CountByTenant(tc.Ctx, tenantID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})

	t.Run("Update Workflow", func(t *testing.T) {
		workflows, err := repo.FindByTenant(tc.Ctx, tenantID, 1, 0)
		require.NoError(t, err)
		require.Len(t, workflows, 1)

		workflow := workflows[0]
		workflow.Name = "Updated Workflow Name"
		workflow.Status = domain.WorkflowStatusActive

		err = repo.Update(tc.Ctx, workflow)
		require.NoError(t, err)

		found, err := repo.FindByID(tc.Ctx, workflow.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Workflow Name", found.Name)
		assert.Equal(t, domain.WorkflowStatusActive, found.Status)
	})

	t.Run("Delete Workflow", func(t *testing.T) {
		workflows, err := repo.FindByTenant(tc.Ctx, tenantID, 1, 0)
		require.NoError(t, err)
		require.Len(t, workflows, 1)

		err = repo.Delete(tc.Ctx, workflows[0].ID)
		require.NoError(t, err)

		_, err = repo.FindByID(tc.Ctx, workflows[0].ID)
		assert.Error(t, err)
	})
}

func TestAlertRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tc := setupTestDB(t)
	defer tc.cleanup(t)

	repo := pgadapter.NewAlertRepository(tc.Pool)
	tenantID := createTestTenant(tc.Ctx, tc.Pool)

	t.Run("Create and Find Alert", func(t *testing.T) {
		alert := &domain.Alert{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Title:     "Test Alert",
			Severity:  domain.AlertSeverityWarning,
			Status:    domain.AlertStatusTriggered,
			CreatedAt: time.Now(),
		}

		err := repo.Save(tc.Ctx, alert)
		require.NoError(t, err)

		found, err := repo.FindByID(tc.Ctx, alert.ID)
		require.NoError(t, err)
		assert.Equal(t, alert.Title, found.Title)
		assert.Equal(t, alert.Severity, found.Severity)
	})

	t.Run("List Alerts by Tenant", func(t *testing.T) {
		alerts, err := repo.FindByTenant(tc.Ctx, tenantID, 10, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(alerts), 1)
	})

	t.Run("Count Alerts by Tenant", func(t *testing.T) {
		count, err := repo.CountByTenant(tc.Ctx, tenantID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})
}
