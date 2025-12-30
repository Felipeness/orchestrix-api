package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
	"github.com/orchestrix/orchestrix-api/internal/core/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricService_Ingest(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("ingests metric successfully", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.IngestMetricInput{
			TenantID: tenantID,
			Name:     "cpu_usage",
			Value:    75.5,
			Labels:   map[string]string{"host": "server-1"},
		}

		err := svc.Ingest(ctx, input)

		require.NoError(t, err)
		assert.True(t, metricRepo.SaveCalled)
		assert.True(t, tenantSetter.SetCalled)
	})

	t.Run("ingests metric with custom timestamp", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		customTime := time.Now().Add(-1 * time.Hour)
		input := port.IngestMetricInput{
			TenantID:  tenantID,
			Name:      "cpu_usage",
			Value:     75.5,
			Timestamp: &customTime,
		}

		err := svc.Ingest(ctx, input)

		require.NoError(t, err)
		assert.True(t, metricRepo.SaveCalled)
	})

	t.Run("returns error when tenant context fails", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()
		tenantSetter.SetErr = domain.ErrUnauthorized

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.IngestMetricInput{
			TenantID: tenantID,
			Name:     "cpu_usage",
			Value:    75.5,
		}

		err := svc.Ingest(ctx, input)

		require.Error(t, err)
		assert.Equal(t, domain.ErrUnauthorized, err)
	})

	t.Run("returns error when save fails", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		metricRepo.SaveErr = domain.ErrInternal
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.IngestMetricInput{
			TenantID: tenantID,
			Name:     "cpu_usage",
			Value:    75.5,
		}

		err := svc.Ingest(ctx, input)

		require.Error(t, err)
	})
}

func TestMetricService_IngestBatch(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("ingests batch successfully", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.IngestMetricBatchInput{
			TenantID: tenantID,
			Metrics: []port.IngestMetricInput{
				{TenantID: tenantID, Name: "cpu_usage", Value: 75.5},
				{TenantID: tenantID, Name: "memory_usage", Value: 60.0},
				{TenantID: tenantID, Name: "disk_usage", Value: 45.0},
			},
		}

		result, err := svc.IngestBatch(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, 3, result.Ingested)
		assert.Equal(t, 0, result.Failed)
		assert.True(t, metricRepo.SaveBatchCalled)
	})

	t.Run("returns error when batch too large", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		// Create batch larger than MaxBatchSize (10000)
		metrics := make([]port.IngestMetricInput, 10001)
		for i := range metrics {
			metrics[i] = port.IngestMetricInput{
				TenantID: tenantID,
				Name:     "metric",
				Value:    float64(i),
			}
		}

		input := port.IngestMetricBatchInput{
			TenantID: tenantID,
			Metrics:  metrics,
		}

		result, err := svc.IngestBatch(ctx, input)

		require.Error(t, err)
		assert.Equal(t, domain.ErrBatchTooLarge, err)
		assert.Nil(t, result)
	})

	t.Run("returns error when save batch fails", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		metricRepo.SaveErr = domain.ErrInternal
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.IngestMetricBatchInput{
			TenantID: tenantID,
			Metrics: []port.IngestMetricInput{
				{TenantID: tenantID, Name: "cpu_usage", Value: 75.5},
			},
		}

		result, err := svc.IngestBatch(ctx, input)

		require.Error(t, err)
		assert.Equal(t, 0, result.Ingested)
		assert.Equal(t, 1, result.Failed)
	})
}

func TestMetricService_Query(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("queries metrics successfully", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		// Add test metrics
		for i := 0; i < 5; i++ {
			metricRepo.AddMetric(&domain.Metric{
				ID:        uuid.New(),
				TenantID:  tenantID,
				Name:      "cpu_usage",
				Value:     float64(50 + i*10),
				Timestamp: time.Now(),
			})
		}

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		query := domain.MetricQuery{
			TenantID:  tenantID,
			Name:      "cpu_usage",
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Limit:     10,
		}

		result, err := svc.Query(ctx, query)

		require.NoError(t, err)
		assert.Equal(t, int64(5), result.Total)
		assert.Len(t, result.Metrics, 5)
	})

	t.Run("returns error for invalid query", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		// Query without name should fail validation
		query := domain.MetricQuery{
			TenantID: tenantID,
			Name:     "", // Empty name
		}

		result, err := svc.Query(ctx, query)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestMetricService_GetLatest(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns latest metric", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		expectedMetric := &domain.Metric{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      "cpu_usage",
			Value:     90.0,
			Timestamp: time.Now(),
		}
		metricRepo.AddMetric(expectedMetric)

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		result, err := svc.GetLatest(ctx, tenantID, "cpu_usage", nil)

		require.NoError(t, err)
		assert.Equal(t, expectedMetric.Value, result.Value)
	})

	t.Run("returns error when metric not found", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		result, err := svc.GetLatest(ctx, tenantID, "nonexistent", nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrMetricNotFound, err)
	})
}

func TestMetricService_GetAggregate(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns aggregated stats", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		query := domain.MetricQuery{
			TenantID:  tenantID,
			Name:      "cpu_usage",
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
		}

		result, err := svc.GetAggregate(ctx, query)

		require.NoError(t, err)
		assert.Equal(t, int64(100), result.Count)
		assert.Equal(t, 50.5, result.Average)
		assert.Equal(t, 10.0, result.Min)
		assert.Equal(t, 90.0, result.Max)
	})
}

func TestMetricService_GetSeries(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns time series data", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		metricRepo.Series = []*domain.TimeBucket{
			{Bucket: time.Now().Add(-30 * time.Minute), Count: 10, Average: 50.0},
			{Bucket: time.Now().Add(-15 * time.Minute), Count: 10, Average: 55.0},
			{Bucket: time.Now(), Count: 10, Average: 60.0},
		}
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		query := domain.MetricQuery{
			TenantID:  tenantID,
			Name:      "cpu_usage",
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
		}

		result, err := svc.GetSeries(ctx, query, 5*time.Minute)

		require.NoError(t, err)
		assert.Len(t, result, 3)
	})
}

func TestMetricService_ListNames(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns metric names", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		metricRepo.Names = []string{"cpu_usage", "memory_usage", "disk_usage"}
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		result, err := svc.ListNames(ctx, tenantID, "")

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Contains(t, result, "cpu_usage")
	})
}

func TestMetricService_CreateDefinition(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("creates definition successfully", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.CreateMetricDefinitionInput{
			TenantID:      tenantID,
			Name:          "cpu_usage",
			Type:          domain.MetricTypeGauge,
			Aggregation:   domain.AggregationAvg,
			RetentionDays: 30,
		}

		result, err := svc.CreateDefinition(ctx, input)

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, input.Name, result.Name)
		assert.True(t, defRepo.SaveCalled)
	})

	t.Run("returns error when definition exists", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		defRepo.AddDefinition(&domain.MetricDefinition{
			ID:       uuid.New(),
			TenantID: tenantID,
			Name:     "cpu_usage",
		})
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.CreateMetricDefinitionInput{
			TenantID:      tenantID,
			Name:          "cpu_usage",
			Type:          domain.MetricTypeGauge,
			Aggregation:   domain.AggregationAvg,
			RetentionDays: 30,
		}

		result, err := svc.CreateDefinition(ctx, input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrMetricDefinitionExists, err)
	})
}

func TestMetricService_UpdateDefinition(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("updates definition successfully", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		defRepo.AddDefinition(&domain.MetricDefinition{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Name:          "cpu_usage",
			Type:          domain.MetricTypeGauge,
			Aggregation:   domain.AggregationAvg,
			RetentionDays: 30,
		})
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		newDisplayName := "CPU Usage %"
		newRetention := 60
		input := port.UpdateMetricDefinitionInput{
			DisplayName:   &newDisplayName,
			RetentionDays: &newRetention,
		}

		result, err := svc.UpdateDefinition(ctx, tenantID, "cpu_usage", input)

		require.NoError(t, err)
		assert.Equal(t, newDisplayName, *result.DisplayName)
		assert.Equal(t, newRetention, result.RetentionDays)
		assert.True(t, defRepo.UpdateCalled)
	})

	t.Run("returns error when definition not found", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		input := port.UpdateMetricDefinitionInput{}

		result, err := svc.UpdateDefinition(ctx, tenantID, "nonexistent", input)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrMetricDefinitionNotFound, err)
	})
}

func TestMetricService_DeleteDefinition(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("deletes definition successfully", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		defRepo.AddDefinition(&domain.MetricDefinition{
			ID:       uuid.New(),
			TenantID: tenantID,
			Name:     "cpu_usage",
		})
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		err := svc.DeleteDefinition(ctx, tenantID, "cpu_usage")

		require.NoError(t, err)
		assert.True(t, defRepo.DeleteCalled)
	})

	t.Run("returns error when definition not found", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		err := svc.DeleteDefinition(ctx, tenantID, "nonexistent")

		require.Error(t, err)
		assert.Equal(t, domain.ErrMetricDefinitionNotFound, err)
	})
}

func TestMetricService_ListDefinitions(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns paginated definitions", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		for i := 0; i < 3; i++ {
			defRepo.AddDefinition(&domain.MetricDefinition{
				ID:       uuid.New(),
				TenantID: tenantID,
				Name:     "metric_" + string(rune('a'+i)),
			})
		}
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		result, err := svc.ListDefinitions(ctx, tenantID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
		assert.Len(t, result.Definitions, 3)
	})
}

func TestMetricService_GetDefinition(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns definition", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		expected := &domain.MetricDefinition{
			ID:       uuid.New(),
			TenantID: tenantID,
			Name:     "cpu_usage",
			Type:     domain.MetricTypeGauge,
		}
		defRepo.AddDefinition(expected)
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		result, err := svc.GetDefinition(ctx, tenantID, "cpu_usage")

		require.NoError(t, err)
		assert.Equal(t, expected.Name, result.Name)
		assert.Equal(t, expected.Type, result.Type)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		metricRepo := mocks.NewMockMetricRepository()
		defRepo := mocks.NewMockMetricDefinitionRepository()
		alertRuleSvc := mocks.NewMockAlertRuleService()
		tenantSetter := mocks.NewMockTenantContextSetter()

		svc := NewMetricService(metricRepo, defRepo, alertRuleSvc, tenantSetter)

		result, err := svc.GetDefinition(ctx, tenantID, "nonexistent")

		require.Error(t, err)
		assert.Nil(t, result)
	})
}
