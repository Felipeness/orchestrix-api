package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// ============================================================================
// MOCK METRIC REPOSITORY
// ============================================================================

type MockMetricRepository struct {
	mu      sync.RWMutex
	metrics []*domain.Metric

	SaveCalled      bool
	SaveBatchCalled bool
	SaveErr         error
	FindErr         error
	Aggregate       *domain.MetricAggregate
	Series          []*domain.TimeBucket
	Names           []string
}

func NewMockMetricRepository() *MockMetricRepository {
	return &MockMetricRepository{
		metrics: make([]*domain.Metric, 0),
		Aggregate: &domain.MetricAggregate{
			Count:   100,
			Average: 50.5,
			Min:     10.0,
			Max:     90.0,
			Sum:     5050.0,
		},
		Series: make([]*domain.TimeBucket, 0),
		Names:  make([]string, 0),
	}
}

func (m *MockMetricRepository) Save(ctx context.Context, metric *domain.Metric) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *MockMetricRepository) SaveBatch(ctx context.Context, metrics []*domain.Metric) (int, error) {
	m.SaveBatchCalled = true
	if m.SaveErr != nil {
		return 0, m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metrics...)
	return len(metrics), nil
}

func (m *MockMetricRepository) FindByQuery(ctx context.Context, query domain.MetricQuery) ([]*domain.Metric, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Metric
	for _, metric := range m.metrics {
		if metric.TenantID == query.TenantID && metric.Name == query.Name {
			result = append(result, metric)
		}
	}
	return result, nil
}

func (m *MockMetricRepository) CountByQuery(ctx context.Context, query domain.MetricQuery) (int64, error) {
	if m.FindErr != nil {
		return 0, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, metric := range m.metrics {
		if metric.TenantID == query.TenantID && metric.Name == query.Name {
			count++
		}
	}
	return count, nil
}

func (m *MockMetricRepository) FindLatest(ctx context.Context, tenantID uuid.UUID, name string, labels map[string]string) (*domain.Metric, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := len(m.metrics) - 1; i >= 0; i-- {
		if m.metrics[i].TenantID == tenantID && m.metrics[i].Name == name {
			return m.metrics[i], nil
		}
	}
	return nil, domain.ErrMetricNotFound
}

func (m *MockMetricRepository) GetAggregate(ctx context.Context, query domain.MetricQuery) (*domain.MetricAggregate, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	return m.Aggregate, nil
}

func (m *MockMetricRepository) GetSeries(ctx context.Context, query domain.MetricQuery, bucketSize time.Duration) ([]*domain.TimeBucket, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	return m.Series, nil
}

func (m *MockMetricRepository) ListNames(ctx context.Context, tenantID uuid.UUID, prefix string) ([]string, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	return m.Names, nil
}

func (m *MockMetricRepository) AddMetric(metric *domain.Metric) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, metric)
}

func (m *MockMetricRepository) AddName(name string) {
	m.Names = append(m.Names, name)
}

// ============================================================================
// MOCK METRIC DEFINITION REPOSITORY
// ============================================================================

type MockMetricDefinitionRepository struct {
	mu          sync.RWMutex
	definitions map[string]*domain.MetricDefinition

	SaveCalled   bool
	UpdateCalled bool
	DeleteCalled bool
	SaveErr      error
	UpdateErr    error
	DeleteErr    error
	FindErr      error
}

func NewMockMetricDefinitionRepository() *MockMetricDefinitionRepository {
	return &MockMetricDefinitionRepository{
		definitions: make(map[string]*domain.MetricDefinition),
	}
}

func (m *MockMetricDefinitionRepository) FindByName(ctx context.Context, tenantID uuid.UUID, name string) (*domain.MetricDefinition, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := tenantID.String() + ":" + name
	if def, ok := m.definitions[key]; ok {
		return def, nil
	}
	return nil, domain.ErrMetricDefinitionNotFound
}

func (m *MockMetricDefinitionRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.MetricDefinition, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.MetricDefinition
	for _, def := range m.definitions {
		if def.TenantID == tenantID {
			result = append(result, def)
		}
	}
	return result, nil
}

func (m *MockMetricDefinitionRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if m.FindErr != nil {
		return 0, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, def := range m.definitions {
		if def.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockMetricDefinitionRepository) Save(ctx context.Context, def *domain.MetricDefinition) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := def.TenantID.String() + ":" + def.Name
	m.definitions[key] = def
	return nil
}

func (m *MockMetricDefinitionRepository) Update(ctx context.Context, def *domain.MetricDefinition) error {
	m.UpdateCalled = true
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := def.TenantID.String() + ":" + def.Name
	m.definitions[key] = def
	return nil
}

func (m *MockMetricDefinitionRepository) Delete(ctx context.Context, tenantID uuid.UUID, name string) error {
	m.DeleteCalled = true
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := tenantID.String() + ":" + name
	delete(m.definitions, key)
	return nil
}

func (m *MockMetricDefinitionRepository) AddDefinition(def *domain.MetricDefinition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := def.TenantID.String() + ":" + def.Name
	m.definitions[key] = def
}

// ============================================================================
// MOCK ALERT RULE SERVICE
// ============================================================================

type MockAlertRuleService struct {
	mu    sync.RWMutex
	rules map[uuid.UUID]*domain.AlertRule

	EvaluateCalled bool
	EvaluateErr    error
}

func NewMockAlertRuleService() *MockAlertRuleService {
	return &MockAlertRuleService{
		rules: make(map[uuid.UUID]*domain.AlertRule),
	}
}

func (m *MockAlertRuleService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.AlertRuleListResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.AlertRule
	for _, r := range m.rules {
		if r.TenantID == tenantID {
			result = append(result, r)
		}
	}
	return &port.AlertRuleListResult{
		Rules: result,
		Total: int64(len(result)),
		Page:  page,
		Limit: limit,
	}, nil
}

func (m *MockAlertRuleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.AlertRule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.rules[id]; ok {
		return r, nil
	}
	return nil, domain.ErrAlertRuleNotFound
}

func (m *MockAlertRuleService) Create(ctx context.Context, input port.CreateAlertRuleInput) (*domain.AlertRule, error) {
	return nil, nil
}

func (m *MockAlertRuleService) Update(ctx context.Context, id uuid.UUID, input port.UpdateAlertRuleInput) (*domain.AlertRule, error) {
	return nil, nil
}

func (m *MockAlertRuleService) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *MockAlertRuleService) Evaluate(ctx context.Context, tenantID uuid.UUID, metricName string, value float64) error {
	m.EvaluateCalled = true
	return m.EvaluateErr
}
