package domain

import (
	"encoding/json"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry in the domain
type AuditLog struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	UserID       *uuid.UUID
	EventType    string
	ResourceType string
	ResourceID   *uuid.UUID
	Action       string
	OldValue     json.RawMessage
	NewValue     json.RawMessage
	IPAddress    *netip.Addr
	UserAgent    *string
	CreatedAt    time.Time
}

// Common audit event types
const (
	AuditEventWorkflowCreated   = "workflow.created"
	AuditEventWorkflowUpdated   = "workflow.updated"
	AuditEventWorkflowDeleted   = "workflow.deleted"
	AuditEventWorkflowExecuted  = "workflow.executed"
	AuditEventAlertCreated      = "alert.created"
	AuditEventAlertAcknowledged = "alert.acknowledged"
	AuditEventAlertResolved     = "alert.resolved"
	AuditEventAlertRuleCreated  = "alertrule.created"
	AuditEventAlertRuleUpdated  = "alertrule.updated"
	AuditEventAlertRuleDeleted  = "alertrule.deleted"
)

// Common resource types
const (
	ResourceTypeWorkflow  = "workflow"
	ResourceTypeExecution = "execution"
	ResourceTypeAlert     = "alert"
	ResourceTypeAlertRule = "alertrule"
)

// Common actions
const (
	ActionCreate      = "create"
	ActionUpdate      = "update"
	ActionDelete      = "delete"
	ActionExecute     = "execute"
	ActionAcknowledge = "acknowledge"
	ActionResolve     = "resolve"
)

// NewAuditLog creates a new audit log entry
func NewAuditLog(
	tenantID uuid.UUID,
	userID *uuid.UUID,
	eventType string,
	resourceType string,
	resourceID *uuid.UUID,
	action string,
) *AuditLog {
	return &AuditLog{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		CreatedAt:    time.Now(),
	}
}

// WithOldValue sets the old value for the audit log
func (a *AuditLog) WithOldValue(v interface{}) *AuditLog {
	if v != nil {
		a.OldValue, _ = json.Marshal(v)
	}
	return a
}

// WithNewValue sets the new value for the audit log
func (a *AuditLog) WithNewValue(v interface{}) *AuditLog {
	if v != nil {
		a.NewValue, _ = json.Marshal(v)
	}
	return a
}

// WithIPAddress sets the IP address for the audit log
func (a *AuditLog) WithIPAddress(ip *netip.Addr) *AuditLog {
	a.IPAddress = ip
	return a
}

// WithUserAgent sets the user agent for the audit log
func (a *AuditLog) WithUserAgent(ua *string) *AuditLog {
	a.UserAgent = ua
	return a
}
