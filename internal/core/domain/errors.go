package domain

import "errors"

// Domain errors - these are business logic errors
var (
	// Workflow errors
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrWorkflowCannotExecute = errors.New("workflow cannot be executed")
	ErrInvalidDefinition    = errors.New("invalid workflow definition")
	ErrNoSteps              = errors.New("workflow has no steps")

	// Execution errors
	ErrExecutionNotFound    = errors.New("execution not found")
	ErrExecutionNotRunning  = errors.New("execution is not running")
	ErrExecutionCannotCancel = errors.New("execution cannot be cancelled")

	// Alert errors
	ErrAlertNotFound          = errors.New("alert not found")
	ErrAlertAlreadyAcknowledged = errors.New("alert is already acknowledged")
	ErrAlertAlreadyResolved   = errors.New("alert is already resolved")

	// AlertRule errors
	ErrAlertRuleNotFound      = errors.New("alert rule not found")
	ErrInvalidConditionType   = errors.New("invalid condition type")
	ErrInvalidConditionConfig = errors.New("invalid condition config")
	ErrInvalidOperator        = errors.New("invalid operator")
	ErrRuleOnCooldown         = errors.New("rule is on cooldown")

	// Audit errors
	ErrAuditLogNotFound = errors.New("audit log not found")

	// General errors
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInternal     = errors.New("internal error")
)
