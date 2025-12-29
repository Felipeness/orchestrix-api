package validation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/orchestrix/orchestrix-api/pkg/apperror"
)

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validator validates input data
type Validator struct {
	errors []FieldError
}

// New creates a new Validator
func New() *Validator {
	return &Validator{
		errors: make([]FieldError, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, FieldError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Error returns the validation error as an AppError
func (v *Validator) Error() *apperror.AppError {
	if !v.HasErrors() {
		return nil
	}

	fieldErrors := make(map[string]string)
	for _, e := range v.errors {
		fieldErrors[e.Field] = e.Message
	}

	return apperror.ValidationWithFields(fieldErrors)
}

// Errors returns the validation errors
func (v *Validator) Errors() []FieldError {
	return v.errors
}

// Required validates that a string is not empty
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, fmt.Sprintf("%s is required", field))
	}
	return v
}

// RequiredPointer validates that a pointer is not nil
func (v *Validator) RequiredPointer(field string, value interface{}) *Validator {
	if value == nil {
		v.AddError(field, fmt.Sprintf("%s is required", field))
	}
	return v
}

// MinLength validates minimum string length
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if utf8.RuneCountInString(value) < min {
		v.AddError(field, fmt.Sprintf("%s must be at least %d characters", field, min))
	}
	return v
}

// MaxLength validates maximum string length
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.AddError(field, fmt.Sprintf("%s must not exceed %d characters", field, max))
	}
	return v
}

// Length validates exact string length
func (v *Validator) Length(field, value string, length int) *Validator {
	if utf8.RuneCountInString(value) != length {
		v.AddError(field, fmt.Sprintf("%s must be exactly %d characters", field, length))
	}
	return v
}

// Min validates minimum numeric value
func (v *Validator) Min(field string, value, min int) *Validator {
	if value < min {
		v.AddError(field, fmt.Sprintf("%s must be at least %d", field, min))
	}
	return v
}

// Max validates maximum numeric value
func (v *Validator) Max(field string, value, max int) *Validator {
	if value > max {
		v.AddError(field, fmt.Sprintf("%s must not exceed %d", field, max))
	}
	return v
}

// Range validates value is within range
func (v *Validator) Range(field string, value, min, max int) *Validator {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("%s must be between %d and %d", field, min, max))
	}
	return v
}

// UUID validates that a string is a valid UUID
func (v *Validator) UUID(field, value string) *Validator {
	if _, err := uuid.Parse(value); err != nil {
		v.AddError(field, fmt.Sprintf("%s must be a valid UUID", field))
	}
	return v
}

// Email validates email format
func (v *Validator) Email(field, value string) *Validator {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.AddError(field, fmt.Sprintf("%s must be a valid email address", field))
	}
	return v
}

// Pattern validates against a regex pattern
func (v *Validator) Pattern(field, value, pattern, message string) *Validator {
	re, err := regexp.Compile(pattern)
	if err != nil {
		v.AddError(field, "invalid pattern")
		return v
	}
	if !re.MatchString(value) {
		v.AddError(field, message)
	}
	return v
}

// Enum validates that a value is in a list of allowed values
func (v *Validator) Enum(field, value string, allowed []string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.AddError(field, fmt.Sprintf("%s must be one of: %s", field, strings.Join(allowed, ", ")))
	return v
}

// JSON validates that a string is valid JSON
func (v *Validator) JSON(field, value string) *Validator {
	var js json.RawMessage
	if err := json.Unmarshal([]byte(value), &js); err != nil {
		v.AddError(field, fmt.Sprintf("%s must be valid JSON", field))
	}
	return v
}

// CronExpression validates a cron expression format
func (v *Validator) CronExpression(field, value string) *Validator {
	if value == "" {
		return v
	}

	// Allow special cron shortcuts
	if strings.HasPrefix(value, "@") {
		validShortcuts := []string{"@yearly", "@annually", "@monthly", "@weekly", "@daily", "@midnight", "@hourly"}
		for _, s := range validShortcuts {
			if value == s {
				return v
			}
		}
		v.AddError(field, fmt.Sprintf("%s is not a valid cron shortcut", field))
		return v
	}

	// Standard cron format: minute hour day month weekday
	parts := strings.Fields(value)
	if len(parts) != 5 && len(parts) != 6 {
		v.AddError(field, fmt.Sprintf("%s must be a valid cron expression", field))
	}

	return v
}

// Custom adds a custom validation
func (v *Validator) Custom(field string, valid bool, message string) *Validator {
	if !valid {
		v.AddError(field, message)
	}
	return v
}

// If adds conditional validation
func (v *Validator) If(condition bool, fn func(v *Validator)) *Validator {
	if condition {
		fn(v)
	}
	return v
}

// Validate runs validation and returns error if any
func Validate(fn func(v *Validator)) error {
	v := New()
	fn(v)
	if v.HasErrors() {
		return v.Error()
	}
	return nil
}
