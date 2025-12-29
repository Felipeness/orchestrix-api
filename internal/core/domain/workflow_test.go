package domain

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_CanExecute(t *testing.T) {
	tests := []struct {
		name     string
		status   WorkflowStatus
		expected bool
	}{
		{
			name:     "active workflow can execute",
			status:   WorkflowStatusActive,
			expected: true,
		},
		{
			name:     "draft workflow cannot execute",
			status:   WorkflowStatusDraft,
			expected: false,
		},
		{
			name:     "inactive workflow cannot execute",
			status:   WorkflowStatusInactive,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Workflow{
				ID:     uuid.New(),
				Status: tt.status,
			}

			result := w.CanExecute()

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflow_Activate(t *testing.T) {
	t.Run("activates workflow with valid definition", func(t *testing.T) {
		definition := WorkflowDefinition{
			Steps: []WorkflowStep{
				{Name: "step1", Type: "http", Config: map[string]interface{}{}},
			},
		}
		defJSON, _ := json.Marshal(definition)

		w := &Workflow{
			ID:         uuid.New(),
			Status:     WorkflowStatusDraft,
			Definition: defJSON,
		}

		err := w.Activate()

		assert.NoError(t, err)
		assert.Equal(t, WorkflowStatusActive, w.Status)
	})

	t.Run("returns error for empty definition", func(t *testing.T) {
		w := &Workflow{
			ID:         uuid.New(),
			Status:     WorkflowStatusDraft,
			Definition: json.RawMessage(`{"steps":[]}`),
		}

		err := w.Activate()

		assert.Error(t, err)
		assert.Equal(t, ErrNoSteps, err)
		assert.Equal(t, WorkflowStatusDraft, w.Status)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		w := &Workflow{
			ID:         uuid.New(),
			Status:     WorkflowStatusDraft,
			Definition: json.RawMessage(`invalid json`),
		}

		err := w.Activate()

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidDefinition, err)
	})
}

func TestWorkflow_Deactivate(t *testing.T) {
	t.Run("deactivates active workflow", func(t *testing.T) {
		w := &Workflow{
			ID:     uuid.New(),
			Status: WorkflowStatusActive,
		}

		w.Deactivate()

		assert.Equal(t, WorkflowStatusInactive, w.Status)
	})
}

func TestWorkflow_ParseDefinition(t *testing.T) {
	t.Run("parses valid definition", func(t *testing.T) {
		definition := WorkflowDefinition{
			Steps: []WorkflowStep{
				{Name: "step1", Type: "http", Config: map[string]interface{}{"url": "http://example.com"}},
				{Name: "step2", Type: "email", Config: map[string]interface{}{"to": "test@example.com"}},
			},
		}
		defJSON, _ := json.Marshal(definition)

		w := &Workflow{
			Definition: defJSON,
		}

		result, err := w.ParseDefinition()

		assert.NoError(t, err)
		assert.Len(t, result.Steps, 2)
		assert.Equal(t, "step1", result.Steps[0].Name)
		assert.Equal(t, "http", result.Steps[0].Type)
	})

	t.Run("returns empty definition for nil", func(t *testing.T) {
		w := &Workflow{
			Definition: nil,
		}

		result, err := w.ParseDefinition()

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Steps)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		w := &Workflow{
			Definition: json.RawMessage(`not valid json`),
		}

		result, err := w.ParseDefinition()

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, ErrInvalidDefinition, err)
	})
}

func TestWorkflow_HasDynamicDefinition(t *testing.T) {
	t.Run("returns true when has steps", func(t *testing.T) {
		definition := WorkflowDefinition{
			Steps: []WorkflowStep{
				{Name: "step1", Type: "http"},
			},
		}
		defJSON, _ := json.Marshal(definition)

		w := &Workflow{
			Definition: defJSON,
		}

		assert.True(t, w.HasDynamicDefinition())
	})

	t.Run("returns false when no steps", func(t *testing.T) {
		w := &Workflow{
			Definition: json.RawMessage(`{"steps":[]}`),
		}

		assert.False(t, w.HasDynamicDefinition())
	})

	t.Run("returns false for invalid JSON", func(t *testing.T) {
		w := &Workflow{
			Definition: json.RawMessage(`invalid`),
		}

		assert.False(t, w.HasDynamicDefinition())
	})
}
