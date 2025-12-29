package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Workflow represents a workflow entity in the domain
type Workflow struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description *string
	Definition  json.RawMessage
	Schedule    *string
	Status      WorkflowStatus
	Version     int32
	CreatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusInactive WorkflowStatus = "inactive"
)

// WorkflowDefinition represents the structure of a workflow definition
type WorkflowDefinition struct {
	Steps []WorkflowStep `json:"steps"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// CanExecute checks if the workflow can be executed
func (w *Workflow) CanExecute() bool {
	return w.Status == WorkflowStatusActive
}

// Activate activates the workflow if it has a valid definition
func (w *Workflow) Activate() error {
	def, err := w.ParseDefinition()
	if err != nil {
		return ErrInvalidDefinition
	}
	if len(def.Steps) == 0 {
		return ErrNoSteps
	}
	w.Status = WorkflowStatusActive
	return nil
}

// Deactivate deactivates the workflow
func (w *Workflow) Deactivate() {
	w.Status = WorkflowStatusInactive
}

// ParseDefinition parses the workflow definition JSON
func (w *Workflow) ParseDefinition() (*WorkflowDefinition, error) {
	if len(w.Definition) == 0 {
		return &WorkflowDefinition{}, nil
	}
	var def WorkflowDefinition
	if err := json.Unmarshal(w.Definition, &def); err != nil {
		return nil, ErrInvalidDefinition
	}
	return &def, nil
}

// HasDynamicDefinition checks if the workflow has a dynamic definition with steps
func (w *Workflow) HasDynamicDefinition() bool {
	def, err := w.ParseDefinition()
	if err != nil {
		return false
	}
	return len(def.Steps) > 0
}
