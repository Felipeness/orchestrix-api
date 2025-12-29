package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Execution represents a workflow execution in the domain
type Execution struct {
	ID                 uuid.UUID
	TenantID           uuid.UUID
	WorkflowID         uuid.UUID
	TemporalWorkflowID *string
	TemporalRunID      *string
	Status             ExecutionStatus
	Input              json.RawMessage
	Output             json.RawMessage
	Error              *string
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedBy          *uuid.UUID
	CreatedAt          time.Time
	TriggeredBy        *string
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// IsTerminal checks if the execution is in a terminal state
func (e *Execution) IsTerminal() bool {
	return e.Status == ExecutionStatusCompleted ||
		e.Status == ExecutionStatusFailed ||
		e.Status == ExecutionStatusCancelled
}

// CanCancel checks if the execution can be cancelled
func (e *Execution) CanCancel() bool {
	return e.Status == ExecutionStatusPending || e.Status == ExecutionStatusRunning
}

// MarkAsRunning marks the execution as running
func (e *Execution) MarkAsRunning() {
	e.Status = ExecutionStatusRunning
	now := time.Now()
	e.StartedAt = &now
}

// MarkAsCompleted marks the execution as completed
func (e *Execution) MarkAsCompleted(output json.RawMessage) {
	e.Status = ExecutionStatusCompleted
	e.Output = output
	now := time.Now()
	e.CompletedAt = &now
}

// MarkAsFailed marks the execution as failed
func (e *Execution) MarkAsFailed(errorMsg string) {
	e.Status = ExecutionStatusFailed
	e.Error = &errorMsg
	now := time.Now()
	e.CompletedAt = &now
}

// MarkAsCancelled marks the execution as cancelled
func (e *Execution) MarkAsCancelled() {
	e.Status = ExecutionStatusCancelled
	now := time.Now()
	e.CompletedAt = &now
}
