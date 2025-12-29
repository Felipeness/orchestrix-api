package mocks

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// ============================================================================
// MOCK WORKFLOW REPOSITORY
// ============================================================================

type MockWorkflowRepository struct {
	mu        sync.RWMutex
	workflows map[uuid.UUID]*domain.Workflow

	// For assertions
	SaveCalled   bool
	UpdateCalled bool
	DeleteCalled bool
	SaveErr      error
	UpdateErr    error
	DeleteErr    error
	FindErr      error
}

func NewMockWorkflowRepository() *MockWorkflowRepository {
	return &MockWorkflowRepository{
		workflows: make(map[uuid.UUID]*domain.Workflow),
	}
}

func (m *MockWorkflowRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Workflow, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if w, ok := m.workflows[id]; ok {
		return w, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockWorkflowRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Workflow, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Workflow
	for _, w := range m.workflows {
		if w.TenantID == tenantID {
			result = append(result, w)
		}
	}
	// Apply pagination
	if offset >= len(result) {
		return []*domain.Workflow{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *MockWorkflowRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if m.FindErr != nil {
		return 0, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, w := range m.workflows {
		if w.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockWorkflowRepository) Save(ctx context.Context, workflow *domain.Workflow) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *MockWorkflowRepository) Update(ctx context.Context, workflow *domain.Workflow) error {
	m.UpdateCalled = true
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[workflow.ID] = workflow
	return nil
}

func (m *MockWorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.DeleteCalled = true
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.workflows, id)
	return nil
}

// AddWorkflow adds a workflow to the mock repository (for test setup)
func (m *MockWorkflowRepository) AddWorkflow(w *domain.Workflow) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workflows[w.ID] = w
}

// ============================================================================
// MOCK EXECUTION REPOSITORY
// ============================================================================

type MockExecutionRepository struct {
	mu         sync.RWMutex
	executions map[uuid.UUID]*domain.Execution

	SaveCalled   bool
	UpdateCalled bool
	SaveErr      error
	UpdateErr    error
	FindErr      error
}

func NewMockExecutionRepository() *MockExecutionRepository {
	return &MockExecutionRepository{
		executions: make(map[uuid.UUID]*domain.Execution),
	}
}

func (m *MockExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Execution, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.executions[id]; ok {
		return e, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockExecutionRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Execution, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Execution
	for _, e := range m.executions {
		if e.TenantID == tenantID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockExecutionRepository) FindByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*domain.Execution, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Execution
	for _, e := range m.executions {
		if e.WorkflowID == workflowID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *MockExecutionRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if m.FindErr != nil {
		return 0, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, e := range m.executions {
		if e.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockExecutionRepository) Save(ctx context.Context, execution *domain.Execution) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions[execution.ID] = execution
	return nil
}

func (m *MockExecutionRepository) Update(ctx context.Context, execution *domain.Execution) error {
	m.UpdateCalled = true
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions[execution.ID] = execution
	return nil
}

func (m *MockExecutionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ExecutionStatus, errMsg *string) error {
	m.UpdateCalled = true
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.executions[id]; ok {
		e.Status = status
		e.Error = errMsg
	}
	return nil
}

func (m *MockExecutionRepository) UpdateTemporalIDs(ctx context.Context, id uuid.UUID, temporalWorkflowID, temporalRunID string) error {
	m.UpdateCalled = true
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.executions[id]; ok {
		e.TemporalWorkflowID = &temporalWorkflowID
		e.TemporalRunID = &temporalRunID
	}
	return nil
}

// ============================================================================
// MOCK WORKFLOW EXECUTOR
// ============================================================================

type MockWorkflowExecutor struct {
	ExecuteCalled bool
	CancelCalled  bool
	ExecuteErr    error
	CancelErr     error
	ExecuteResult *port.ExecuteResult
}

func NewMockWorkflowExecutor() *MockWorkflowExecutor {
	return &MockWorkflowExecutor{
		ExecuteResult: &port.ExecuteResult{
			TemporalWorkflowID: "temporal-workflow-123",
			TemporalRunID:      "temporal-run-456",
		},
	}
}

func (m *MockWorkflowExecutor) Execute(ctx context.Context, workflow *domain.Workflow, input map[string]interface{}) (*port.ExecuteResult, error) {
	m.ExecuteCalled = true
	if m.ExecuteErr != nil {
		return nil, m.ExecuteErr
	}
	return m.ExecuteResult, nil
}

func (m *MockWorkflowExecutor) Cancel(ctx context.Context, temporalWorkflowID string) error {
	m.CancelCalled = true
	return m.CancelErr
}

func (m *MockWorkflowExecutor) GetStatus(ctx context.Context, temporalWorkflowID string) (string, error) {
	return "running", nil
}

// ============================================================================
// MOCK AUDIT SERVICE
// ============================================================================

type MockAuditService struct {
	LogCalled bool
	Logs      []*domain.AuditLog
	LogErr    error
}

func NewMockAuditService() *MockAuditService {
	return &MockAuditService{
		Logs: make([]*domain.AuditLog, 0),
	}
}

func (m *MockAuditService) List(ctx context.Context, tenantID uuid.UUID, page, limit int) (*port.AuditListResult, error) {
	return &port.AuditListResult{
		Logs:  m.Logs,
		Total: int64(len(m.Logs)),
		Page:  page,
		Limit: limit,
	}, nil
}

func (m *MockAuditService) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	for _, log := range m.Logs {
		if log.ID == id {
			return log, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *MockAuditService) Log(ctx context.Context, log *domain.AuditLog) error {
	m.LogCalled = true
	if m.LogErr != nil {
		return m.LogErr
	}
	m.Logs = append(m.Logs, log)
	return nil
}

// ============================================================================
// MOCK TENANT CONTEXT SETTER
// ============================================================================

type MockTenantContextSetter struct {
	SetCalled bool
	SetErr    error
	TenantID  uuid.UUID
}

func NewMockTenantContextSetter() *MockTenantContextSetter {
	return &MockTenantContextSetter{}
}

func (m *MockTenantContextSetter) SetTenantContext(ctx context.Context, tenantID uuid.UUID) error {
	m.SetCalled = true
	m.TenantID = tenantID
	return m.SetErr
}

// ============================================================================
// MOCK ALERT REPOSITORY
// ============================================================================

type MockAlertRepository struct {
	mu     sync.RWMutex
	alerts map[uuid.UUID]*domain.Alert

	SaveCalled   bool
	UpdateCalled bool
	SaveErr      error
	UpdateErr    error
	FindErr      error
}

func NewMockAlertRepository() *MockAlertRepository {
	return &MockAlertRepository{
		alerts: make(map[uuid.UUID]*domain.Alert),
	}
}

func (m *MockAlertRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if a, ok := m.alerts[id]; ok {
		return a, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockAlertRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Alert, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Alert
	for _, a := range m.alerts {
		if a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *MockAlertRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if m.FindErr != nil {
		return 0, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, a := range m.alerts {
		if a.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockAlertRepository) Save(ctx context.Context, alert *domain.Alert) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[alert.ID] = alert
	return nil
}

func (m *MockAlertRepository) Update(ctx context.Context, alert *domain.Alert) error {
	m.UpdateCalled = true
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[alert.ID] = alert
	return nil
}

func (m *MockAlertRepository) AddAlert(a *domain.Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts[a.ID] = a
}

// ============================================================================
// MOCK AUDIT REPOSITORY
// ============================================================================

type MockAuditRepository struct {
	mu   sync.RWMutex
	logs map[uuid.UUID]*domain.AuditLog

	SaveCalled bool
	SaveErr    error
	FindErr    error
}

func NewMockAuditRepository() *MockAuditRepository {
	return &MockAuditRepository{
		logs: make(map[uuid.UUID]*domain.AuditLog),
	}
}

func (m *MockAuditRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if l, ok := m.logs[id]; ok {
		return l, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockAuditRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	if m.FindErr != nil {
		return nil, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.TenantID == tenantID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *MockAuditRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	if m.FindErr != nil {
		return 0, m.FindErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var count int64
	for _, l := range m.logs {
		if l.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *MockAuditRepository) Save(ctx context.Context, log *domain.AuditLog) error {
	m.SaveCalled = true
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs[log.ID] = log
	return nil
}
