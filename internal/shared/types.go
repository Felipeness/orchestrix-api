package shared

import "github.com/google/uuid"

// TenantID represents a tenant identifier
type TenantID = uuid.UUID

// UserID represents a user identifier
type UserID = uuid.UUID

// WorkflowID represents a workflow identifier
type WorkflowID = uuid.UUID

// ExecutionID represents an execution identifier
type ExecutionID = uuid.UUID
