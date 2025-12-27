package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of audit event
type EventType string

const (
	EventLogin            EventType = "AUTH.LOGIN"
	EventLogout           EventType = "AUTH.LOGOUT"
	EventWorkflowCreated  EventType = "WORKFLOW.CREATED"
	EventWorkflowExecuted EventType = "WORKFLOW.EXECUTED"
	EventWorkflowFailed   EventType = "WORKFLOW.FAILED"
	EventConfigChanged    EventType = "CONFIG.CHANGED"
	EventDataExported     EventType = "DATA.EXPORTED"
)

// Outcome represents the result of an action
type Outcome string

const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeFailure Outcome = "FAILURE"
	OutcomeDenied  Outcome = "DENIED"
)

// Event represents an audit log entry
type Event struct {
	ID        uuid.UUID       `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	EventType EventType       `json:"event_type"`
	TenantID  string          `json:"tenant_id"`
	UserID    string          `json:"user_id,omitempty"`
	ServiceID string          `json:"service_id"`
	RequestID string          `json:"request_id"`
	Resource  string          `json:"resource"`
	Action    string          `json:"action"`
	Outcome   Outcome         `json:"outcome"`
	IPAddress string          `json:"ip_address,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
	OldValue  json.RawMessage `json:"old_value,omitempty"`
	NewValue  json.RawMessage `json:"new_value,omitempty"`
}

// Logger handles audit logging
type Logger struct {
	serviceID string
}

// NewLogger creates a new audit logger
func NewLogger(serviceID string) *Logger {
	return &Logger{serviceID: serviceID}
}

// Log records an audit event
func (l *Logger) Log(ctx context.Context, event Event) {
	event.ID = uuid.New()
	event.Timestamp = time.Now().UTC()
	event.ServiceID = l.serviceID

	// In production, this would write to Kafka or a dedicated audit service
	data, _ := json.Marshal(event)
	slog.Info("audit", "event", string(data))
}

// LogSuccess logs a successful action
func (l *Logger) LogSuccess(ctx context.Context, eventType EventType, resource, action string) {
	l.Log(ctx, Event{
		EventType: eventType,
		Resource:  resource,
		Action:    action,
		Outcome:   OutcomeSuccess,
	})
}

// LogFailure logs a failed action
func (l *Logger) LogFailure(ctx context.Context, eventType EventType, resource, action string) {
	l.Log(ctx, Event{
		EventType: eventType,
		Resource:  resource,
		Action:    action,
		Outcome:   OutcomeFailure,
	})
}
