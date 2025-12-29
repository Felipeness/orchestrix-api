package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
	"github.com/orchestrix/orchestrix-api/internal/core/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertService_List(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns paginated alerts", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		// Add test alerts
		for i := 0; i < 3; i++ {
			alertRepo.AddAlert(&domain.Alert{
				ID:       uuid.New(),
				TenantID: tenantID,
				Title:    "Test Alert",
				Severity: domain.AlertSeverityWarning,
				Status:   domain.AlertStatusOpen,
			})
		}

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.List(ctx, tenantID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.Alerts, 3)
		assert.True(t, tenantSetter.SetCalled)
	})

	t.Run("returns empty list when no alerts", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.List(ctx, tenantID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
	})
}

func TestAlertService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns alert when found", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		alertID := uuid.New()
		expected := &domain.Alert{
			ID:       alertID,
			TenantID: uuid.New(),
			Title:    "Test Alert",
			Severity: domain.AlertSeverityCritical,
			Status:   domain.AlertStatusOpen,
		}
		alertRepo.AddAlert(expected)

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.GetByID(ctx, alertID)

		require.NoError(t, err)
		assert.Equal(t, expected.ID, result.ID)
		assert.Equal(t, expected.Title, result.Title)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.GetByID(ctx, uuid.New())

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAlertService_Create(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("creates alert successfully", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		input := port.CreateAlertInput{
			TenantID: tenantID,
			Title:    "New Alert",
			Severity: domain.AlertSeverityWarning,
		}

		result, err := svc.Create(ctx, input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, input.Title, result.Title)
		assert.Equal(t, domain.AlertStatusTriggered, result.Status)
		assert.True(t, alertRepo.SaveCalled)
	})

	t.Run("returns error when save fails", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		alertRepo.SaveErr = domain.ErrInternal
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		input := port.CreateAlertInput{
			TenantID: tenantID,
			Title:    "New Alert",
			Severity: domain.AlertSeverityWarning,
		}

		result, err := svc.Create(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAlertService_Acknowledge(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("acknowledges triggered alert", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		alertID := uuid.New()
		alert := &domain.Alert{
			ID:       alertID,
			TenantID: tenantID,
			Title:    "Test Alert",
			Severity: domain.AlertSeverityWarning,
			Status:   domain.AlertStatusTriggered,
		}
		alertRepo.AddAlert(alert)

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.Acknowledge(ctx, alertID, userID)

		require.NoError(t, err)
		assert.Equal(t, domain.AlertStatusAcknowledged, result.Status)
		assert.NotNil(t, result.AcknowledgedBy)
		assert.NotNil(t, result.AcknowledgedAt)
		assert.True(t, alertRepo.UpdateCalled)
	})

	t.Run("returns error when alert not found", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.Acknowledge(ctx, uuid.New(), userID)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAlertService_Resolve(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("resolves acknowledged alert", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		alertID := uuid.New()
		alert := &domain.Alert{
			ID:       alertID,
			TenantID: tenantID,
			Title:    "Test Alert",
			Severity: domain.AlertSeverityWarning,
			Status:   domain.AlertStatusAcknowledged,
		}
		alertRepo.AddAlert(alert)

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.Resolve(ctx, alertID, userID)

		require.NoError(t, err)
		assert.Equal(t, domain.AlertStatusResolved, result.Status)
		assert.NotNil(t, result.ResolvedBy)
		assert.NotNil(t, result.ResolvedAt)
		assert.True(t, alertRepo.UpdateCalled)
	})

	t.Run("resolves triggered alert directly", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		alertID := uuid.New()
		alert := &domain.Alert{
			ID:       alertID,
			TenantID: tenantID,
			Title:    "Test Alert",
			Severity: domain.AlertSeverityInfo,
			Status:   domain.AlertStatusTriggered,
		}
		alertRepo.AddAlert(alert)

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.Resolve(ctx, alertID, userID)

		require.NoError(t, err)
		assert.Equal(t, domain.AlertStatusResolved, result.Status)
	})

	t.Run("returns error when alert not found", func(t *testing.T) {
		alertRepo := mocks.NewMockAlertRepository()
		auditService := mocks.NewMockAuditService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewAlertService(alertRepo, auditService, tenantSetter)

		result, err := svc.Resolve(ctx, uuid.New(), userID)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}
