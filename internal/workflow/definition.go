package workflow

import (
	"encoding/json"
	"fmt"
)

// StepType defines the type of workflow step
type StepType string

const (
	StepTypeHTTP       StepType = "http"
	StepTypeDelay      StepType = "delay"
	StepTypeLog        StepType = "log"
	StepTypeValidate   StepType = "validate"
	StepTypeProcess    StepType = "process"
	StepTypeNotify     StepType = "notify"
	StepTypeCondition  StepType = "condition"
	StepTypeScript     StepType = "script"
	StepTypeParallel   StepType = "parallel"
)

// WorkflowDefinition represents the structure of a workflow
type WorkflowDefinition struct {
	Version     string           `json:"version"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Timeout     string           `json:"timeout,omitempty"` // e.g., "30m", "1h"
	RetryPolicy *RetryPolicyDef  `json:"retry_policy,omitempty"`
	Steps       []StepDefinition `json:"steps"`
	OnError     []StepDefinition `json:"on_error,omitempty"`
	OnSuccess   []StepDefinition `json:"on_success,omitempty"`
}

// RetryPolicyDef defines retry behavior
type RetryPolicyDef struct {
	MaxAttempts     int    `json:"max_attempts"`
	InitialInterval string `json:"initial_interval"` // e.g., "1s", "5s"
	MaxInterval     string `json:"max_interval"`     // e.g., "1m", "5m"
	Multiplier      float64 `json:"multiplier"`
}

// StepDefinition represents a single step in the workflow
type StepDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        StepType               `json:"type"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Timeout     string                 `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicyDef        `json:"retry_policy,omitempty"`
	ContinueOnError bool               `json:"continue_on_error,omitempty"`

	// Conditional fields
	Condition   string           `json:"condition,omitempty"` // e.g., "${previous.success} == true"
	OnTrue      []StepDefinition `json:"on_true,omitempty"`
	OnFalse     []StepDefinition `json:"on_false,omitempty"`

	// Parallel execution
	Parallel    []StepDefinition `json:"parallel,omitempty"`
}

// HTTPConfig for HTTP step type
type HTTPConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"` // GET, POST, PUT, DELETE
	Headers     map[string]string `json:"headers,omitempty"`
	Body        interface{}       `json:"body,omitempty"`
	Timeout     string            `json:"timeout,omitempty"`
	SuccessCodes []int            `json:"success_codes,omitempty"` // default [200, 201, 202, 204]
}

// DelayConfig for delay step type
type DelayConfig struct {
	Duration string `json:"duration"` // e.g., "5s", "1m", "1h"
}

// LogConfig for log step type
type LogConfig struct {
	Level   string `json:"level"` // info, warn, error
	Message string `json:"message"`
}

// NotifyConfig for notification step type
type NotifyConfig struct {
	Channel string            `json:"channel"` // slack, email, webhook
	Target  string            `json:"target"`  // channel/email/url
	Message string            `json:"message"`
	Data    map[string]string `json:"data,omitempty"`
}

// ScriptConfig for script execution step type
type ScriptConfig struct {
	Language string `json:"language"` // javascript, python (future)
	Code     string `json:"code"`
}

// ParseDefinition parses a JSON definition into a WorkflowDefinition
func ParseDefinition(data json.RawMessage) (*WorkflowDefinition, error) {
	var def WorkflowDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse workflow definition: %w", err)
	}

	// Validate required fields
	if len(def.Steps) == 0 {
		return nil, fmt.Errorf("workflow definition must have at least one step")
	}

	// Validate each step
	for i, step := range def.Steps {
		if step.Type == "" {
			return nil, fmt.Errorf("step %d: type is required", i)
		}
		if step.ID == "" {
			def.Steps[i].ID = fmt.Sprintf("step_%d", i+1)
		}
		if step.Name == "" {
			def.Steps[i].Name = string(step.Type)
		}
	}

	return &def, nil
}

// ParseHTTPConfig parses the config map into HTTPConfig
func ParseHTTPConfig(config map[string]interface{}) (*HTTPConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var cfg HTTPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if len(cfg.SuccessCodes) == 0 {
		cfg.SuccessCodes = []int{200, 201, 202, 204}
	}
	return &cfg, nil
}

// ParseDelayConfig parses the config map into DelayConfig
func ParseDelayConfig(config map[string]interface{}) (*DelayConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var cfg DelayConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Duration == "" {
		cfg.Duration = "1s"
	}
	return &cfg, nil
}

// ParseLogConfig parses the config map into LogConfig
func ParseLogConfig(config map[string]interface{}) (*LogConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var cfg LogConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	return &cfg, nil
}

// ParseNotifyConfig parses the config map into NotifyConfig
func ParseNotifyConfig(config map[string]interface{}) (*NotifyConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var cfg NotifyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
