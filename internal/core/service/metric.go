package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

const (
	// MaxBatchSize is the maximum number of metrics in a single batch
	MaxBatchSize = 10000
)

// MetricService implements port.MetricService
type MetricService struct {
	metricRepo     port.MetricRepository
	definitionRepo port.MetricDefinitionRepository
	alertRuleSvc   port.AlertRuleService
	tenantSetter   port.TenantContextSetter
}

// NewMetricService creates a new metric service
func NewMetricService(
	metricRepo port.MetricRepository,
	definitionRepo port.MetricDefinitionRepository,
	alertRuleSvc port.AlertRuleService,
	tenantSetter port.TenantContextSetter,
) *MetricService {
	return &MetricService{
		metricRepo:     metricRepo,
		definitionRepo: definitionRepo,
		alertRuleSvc:   alertRuleSvc,
		tenantSetter:   tenantSetter,
	}
}

// Ingest ingests a single metric
func (s *MetricService) Ingest(ctx context.Context, input port.IngestMetricInput) error {
	if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
		return err
	}

	timestamp := time.Now()
	if input.Timestamp != nil {
		timestamp = *input.Timestamp
	}

	metric := &domain.Metric{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		Name:      input.Name,
		Value:     input.Value,
		Labels:    input.Labels,
		Source:    input.Source,
		Timestamp: timestamp,
		CreatedAt: time.Now(),
	}

	if err := s.metricRepo.Save(ctx, metric); err != nil {
		return err
	}

	// Async alert evaluation (don't block ingestion)
	go s.evaluateAlerts(context.Background(), input.TenantID, input.Name, input.Value)

	return nil
}

// IngestBatch ingests multiple metrics with high throughput
func (s *MetricService) IngestBatch(ctx context.Context, input port.IngestMetricBatchInput) (*port.IngestBatchResult, error) {
	if len(input.Metrics) > MaxBatchSize {
		return nil, domain.ErrBatchTooLarge
	}

	if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
		return nil, err
	}

	now := time.Now()
	metrics := make([]*domain.Metric, len(input.Metrics))

	for i, m := range input.Metrics {
		timestamp := now
		if m.Timestamp != nil {
			timestamp = *m.Timestamp
		}

		metrics[i] = &domain.Metric{
			ID:        uuid.New(),
			TenantID:  input.TenantID,
			Name:      m.Name,
			Value:     m.Value,
			Labels:    m.Labels,
			Source:    m.Source,
			Timestamp: timestamp,
			CreatedAt: now,
		}
	}

	count, err := s.metricRepo.SaveBatch(ctx, metrics)
	if err != nil {
		return &port.IngestBatchResult{
			Ingested: 0,
			Failed:   len(input.Metrics),
			Errors:   []string{err.Error()},
		}, err
	}

	// Async alert evaluation for batch (sample or aggregate)
	go s.evaluateBatchAlerts(context.Background(), input)

	return &port.IngestBatchResult{
		Ingested: count,
		Failed:   0,
		Errors:   nil,
	}, nil
}

// Query queries metrics with filters
func (s *MetricService) Query(ctx context.Context, query domain.MetricQuery) (*port.MetricQueryResult, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	if err := s.tenantSetter.SetTenantContext(ctx, query.TenantID); err != nil {
		return nil, err
	}

	metrics, err := s.metricRepo.FindByQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	total, err := s.metricRepo.CountByQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	page := 1
	if query.Offset > 0 && query.Limit > 0 {
		page = (query.Offset / query.Limit) + 1
	}

	return &port.MetricQueryResult{
		Metrics: metrics,
		Total:   total,
		Page:    page,
		Limit:   query.Limit,
	}, nil
}

// GetLatest returns the latest metric value
func (s *MetricService) GetLatest(ctx context.Context, tenantID uuid.UUID, name string, labels map[string]string) (*domain.Metric, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	return s.metricRepo.FindLatest(ctx, tenantID, name, labels)
}

// GetAggregate returns aggregated statistics for a metric
func (s *MetricService) GetAggregate(ctx context.Context, query domain.MetricQuery) (*domain.MetricAggregate, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	if err := s.tenantSetter.SetTenantContext(ctx, query.TenantID); err != nil {
		return nil, err
	}

	return s.metricRepo.GetAggregate(ctx, query)
}

// GetSeries returns time-bucketed metric data
func (s *MetricService) GetSeries(ctx context.Context, query domain.MetricQuery, bucketSize time.Duration) ([]*domain.TimeBucket, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	if err := s.tenantSetter.SetTenantContext(ctx, query.TenantID); err != nil {
		return nil, err
	}

	return s.metricRepo.GetSeries(ctx, query, bucketSize)
}

// ListNames returns distinct metric names
func (s *MetricService) ListNames(ctx context.Context, tenantID uuid.UUID, prefix string) ([]string, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	return s.metricRepo.ListNames(ctx, tenantID, prefix)
}

// ListDefinitions returns paginated metric definitions
func (s *MetricService) ListDefinitions(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.MetricDefinitionListResult, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	definitions, err := s.definitionRepo.FindByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.definitionRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &port.MetricDefinitionListResult{
		Definitions: definitions,
		Total:       total,
		Page:        page,
		Limit:       limit,
	}, nil
}

// GetDefinition returns a metric definition by name
func (s *MetricService) GetDefinition(ctx context.Context, tenantID uuid.UUID, name string) (*domain.MetricDefinition, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	return s.definitionRepo.FindByName(ctx, tenantID, name)
}

// CreateDefinition creates a new metric definition
func (s *MetricService) CreateDefinition(ctx context.Context, input port.CreateMetricDefinitionInput) (*domain.MetricDefinition, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, input.TenantID); err != nil {
		return nil, err
	}

	_, err := s.definitionRepo.FindByName(ctx, input.TenantID, input.Name)
	if err == nil {
		return nil, domain.ErrMetricDefinitionExists
	}
	if !errors.Is(err, domain.ErrMetricDefinitionNotFound) {
		return nil, err
	}

	definition := &domain.MetricDefinition{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		Name:           input.Name,
		DisplayName:    input.DisplayName,
		Description:    input.Description,
		Unit:           input.Unit,
		Type:           input.Type,
		Aggregation:    input.Aggregation,
		AlertThreshold: input.AlertThreshold,
		RetentionDays:  input.RetentionDays,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.definitionRepo.Save(ctx, definition); err != nil {
		return nil, err
	}

	return definition, nil
}

// UpdateDefinition updates a metric definition
func (s *MetricService) UpdateDefinition(ctx context.Context, tenantID uuid.UUID, name string, input port.UpdateMetricDefinitionInput) (*domain.MetricDefinition, error) {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return nil, err
	}

	existing, err := s.definitionRepo.FindByName(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if input.DisplayName != nil {
		existing.DisplayName = input.DisplayName
	}
	if input.Description != nil {
		existing.Description = input.Description
	}
	if input.Unit != nil {
		existing.Unit = input.Unit
	}
	if input.Type != nil {
		existing.Type = *input.Type
	}
	if input.Aggregation != nil {
		existing.Aggregation = *input.Aggregation
	}
	if input.AlertThreshold != nil {
		existing.AlertThreshold = input.AlertThreshold
	}
	if input.RetentionDays != nil {
		existing.RetentionDays = *input.RetentionDays
	}
	existing.UpdatedAt = time.Now()

	if err := s.definitionRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

// DeleteDefinition deletes a metric definition
func (s *MetricService) DeleteDefinition(ctx context.Context, tenantID uuid.UUID, name string) error {
	if err := s.tenantSetter.SetTenantContext(ctx, tenantID); err != nil {
		return err
	}

	// Verify it exists first
	_, err := s.definitionRepo.FindByName(ctx, tenantID, name)
	if err != nil {
		return err
	}

	return s.definitionRepo.Delete(ctx, tenantID, name)
}

// evaluateAlerts evaluates alert rules for a single metric
func (s *MetricService) evaluateAlerts(ctx context.Context, tenantID uuid.UUID, metricName string, value float64) {
	if s.alertRuleSvc == nil {
		return
	}

	// Use a fresh context with timeout for background work
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_ = s.alertRuleSvc.Evaluate(ctx, tenantID, metricName, value)
}

// evaluateBatchAlerts evaluates alert rules for batch metrics
func (s *MetricService) evaluateBatchAlerts(ctx context.Context, input port.IngestMetricBatchInput) {
	if s.alertRuleSvc == nil {
		return
	}

	// Use a fresh context with timeout for background work
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Aggregate metrics by name and evaluate the last value for each
	lastValues := make(map[string]float64)
	for _, m := range input.Metrics {
		lastValues[m.Name] = m.Value
	}

	for name, value := range lastValues {
		_ = s.alertRuleSvc.Evaluate(ctx, input.TenantID, name, value)
	}
}
