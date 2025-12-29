package apperror

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
)

// ErrorResponse is the JSON structure returned to clients
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Handler handles error responses in HTTP handlers
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a new error handler
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

// Handle writes an error response to the client
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, err error) {
	appErr := h.toAppError(err)

	// Log internal errors with full details
	if appErr.HTTPStatus >= 500 {
		h.logger.Error("internal error",
			"error", appErr.Error(),
			"code", appErr.Code,
			"path", r.URL.Path,
			"method", r.Method,
		)
	} else {
		h.logger.Debug("client error",
			"code", appErr.Code,
			"message", appErr.Message,
			"path", r.URL.Path,
		)
	}

	h.writeError(w, appErr)
}

// HandleWithContext handles error with additional context
func (h *Handler) HandleWithContext(w http.ResponseWriter, r *http.Request, err error, context map[string]interface{}) {
	appErr := h.toAppError(err)

	// Merge context into details
	if appErr.Details == nil {
		appErr.Details = context
	} else {
		for k, v := range context {
			appErr.Details[k] = v
		}
	}

	h.Handle(w, r, appErr)
}

// toAppError converts any error to an AppError
func (h *Handler) toAppError(err error) *AppError {
	// Check if already an AppError
	if appErr, ok := GetAppError(err); ok {
		return appErr
	}

	// Map domain errors to AppErrors
	return h.mapDomainError(err)
}

// mapDomainError maps domain errors to AppErrors
func (h *Handler) mapDomainError(err error) *AppError {
	switch err {
	// Workflow errors
	case domain.ErrWorkflowNotFound:
		return NotFound("workflow")
	case domain.ErrWorkflowCannotExecute:
		return BadRequest("workflow cannot be executed in current state")
	case domain.ErrInvalidDefinition:
		return Validation("invalid workflow definition")
	case domain.ErrNoSteps:
		return Validation("workflow must have at least one step")

	// Execution errors
	case domain.ErrExecutionNotFound:
		return NotFound("execution")
	case domain.ErrExecutionNotRunning:
		return BadRequest("execution is not running")
	case domain.ErrExecutionCannotCancel:
		return BadRequest("execution cannot be cancelled")

	// Alert errors
	case domain.ErrAlertNotFound:
		return NotFound("alert")
	case domain.ErrAlertAlreadyAcknowledged:
		return Conflict("alert is already acknowledged")
	case domain.ErrAlertAlreadyResolved:
		return Conflict("alert is already resolved")

	// AlertRule errors
	case domain.ErrAlertRuleNotFound:
		return NotFound("alert rule")
	case domain.ErrInvalidConditionType:
		return Validation("invalid condition type")
	case domain.ErrInvalidConditionConfig:
		return Validation("invalid condition configuration")
	case domain.ErrInvalidOperator:
		return Validation("invalid operator")
	case domain.ErrRuleOnCooldown:
		return TooManyRequests("alert rule is on cooldown")

	// Audit errors
	case domain.ErrAuditLogNotFound:
		return NotFound("audit log")

	// General errors
	case domain.ErrNotFound:
		return NotFound("resource")
	case domain.ErrUnauthorized:
		return Unauthorized("")
	case domain.ErrForbidden:
		return Forbidden("")

	default:
		return Internal(err)
	}
}

// writeError writes the error response
func (h *Handler) writeError(w http.ResponseWriter, appErr *AppError) {
	response := ErrorResponse{
		Code:    string(appErr.Code),
		Message: appErr.Message,
		Details: appErr.Details,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode error response", "error", err)
	}
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// WriteSuccess writes a successful JSON response
func WriteSuccess(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusOK, data)
}

// WriteCreated writes a created JSON response
func WriteCreated(w http.ResponseWriter, data interface{}) error {
	return WriteJSON(w, http.StatusCreated, data)
}

// WriteNoContent writes a no content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
