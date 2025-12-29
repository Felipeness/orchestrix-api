package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Alert represents an alert in the domain
type Alert struct {
	ID                           uuid.UUID
	TenantID                     uuid.UUID
	WorkflowID                   *uuid.UUID
	ExecutionID                  *uuid.UUID
	Severity                     AlertSeverity
	Title                        string
	Message                      *string
	Status                       AlertStatus
	AcknowledgedAt               *time.Time
	AcknowledgedBy               *uuid.UUID
	ResolvedAt                   *time.Time
	ResolvedBy                   *uuid.UUID
	CreatedAt                    time.Time
	TriggeredByRuleID            *uuid.UUID
	TriggeredWorkflowExecutionID *uuid.UUID
	Source                       *string
	Metadata                     json.RawMessage
}

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusOpen         AlertStatus = "open"
	AlertStatusTriggered    AlertStatus = "triggered"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// CanAcknowledge checks if the alert can be acknowledged
func (a *Alert) CanAcknowledge() bool {
	return a.Status == AlertStatusOpen || a.Status == AlertStatusTriggered
}

// CanResolve checks if the alert can be resolved
func (a *Alert) CanResolve() bool {
	return a.Status == AlertStatusOpen || a.Status == AlertStatusTriggered || a.Status == AlertStatusAcknowledged
}

// Acknowledge acknowledges the alert
func (a *Alert) Acknowledge(userID uuid.UUID) error {
	if !a.CanAcknowledge() {
		return ErrAlertAlreadyAcknowledged
	}
	a.Status = AlertStatusAcknowledged
	now := time.Now()
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = &userID
	return nil
}

// Resolve resolves the alert
func (a *Alert) Resolve(userID uuid.UUID) error {
	if !a.CanResolve() {
		return ErrAlertAlreadyResolved
	}
	a.Status = AlertStatusResolved
	now := time.Now()
	a.ResolvedAt = &now
	a.ResolvedBy = &userID
	return nil
}
