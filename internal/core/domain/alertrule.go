package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AlertRule represents an alert rule in the domain
type AlertRule struct {
	ID                   uuid.UUID
	TenantID             uuid.UUID
	Name                 string
	Description          *string
	Enabled              bool
	ConditionType        string
	ConditionConfig      json.RawMessage
	Severity             AlertSeverity
	AlertTitleTemplate   string
	AlertMessageTemplate *string
	TriggerWorkflowID    *uuid.UUID
	TriggerInputTemplate json.RawMessage
	CooldownSeconds      int32
	LastTriggeredAt      *time.Time
	CreatedBy            *uuid.UUID
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ThresholdCondition represents a threshold-based condition
type ThresholdCondition struct {
	MetricName string  `json:"metric_name"`
	Operator   string  `json:"operator"` // gt, gte, lt, lte, eq, neq
	Threshold  float64 `json:"threshold"`
}

// CanTrigger checks if the rule can be triggered (respects cooldown)
func (r *AlertRule) CanTrigger() bool {
	if !r.Enabled {
		return false
	}
	if r.LastTriggeredAt == nil {
		return true
	}
	cooldown := time.Duration(r.CooldownSeconds) * time.Second
	return time.Since(*r.LastTriggeredAt) >= cooldown
}

// MarkTriggered marks the rule as triggered
func (r *AlertRule) MarkTriggered() {
	now := time.Now()
	r.LastTriggeredAt = &now
}

// ParseConditionConfig parses the condition config based on condition type
func (r *AlertRule) ParseThresholdCondition() (*ThresholdCondition, error) {
	if r.ConditionType != "threshold" {
		return nil, ErrInvalidConditionType
	}
	var cond ThresholdCondition
	if err := json.Unmarshal(r.ConditionConfig, &cond); err != nil {
		return nil, ErrInvalidConditionConfig
	}
	return &cond, nil
}

// EvaluateThreshold evaluates if the value triggers the threshold condition
func (r *AlertRule) EvaluateThreshold(value float64) (bool, error) {
	cond, err := r.ParseThresholdCondition()
	if err != nil {
		return false, err
	}

	switch cond.Operator {
	case "gt":
		return value > cond.Threshold, nil
	case "gte":
		return value >= cond.Threshold, nil
	case "lt":
		return value < cond.Threshold, nil
	case "lte":
		return value <= cond.Threshold, nil
	case "eq":
		return value == cond.Threshold, nil
	case "neq":
		return value != cond.Threshold, nil
	default:
		return false, ErrInvalidOperator
	}
}
