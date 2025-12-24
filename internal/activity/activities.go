package activity

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Activities holds all activity implementations
type Activities struct {
	// Add dependencies here (db, clients, etc.)
}

// ValidateInput is the input for the Validate activity
type ValidateInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ValidateResult is the result of the Validate activity
type ValidateResult struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// Validate validates the input
func (a *Activities) Validate(ctx context.Context, input ValidateInput) (*ValidateResult, error) {
	slog.Info("Validate activity started", "id", input.ID)

	if input.ID == "" {
		return &ValidateResult{
			Valid:   false,
			Message: "ID is required",
		}, nil
	}

	if input.Name == "" {
		return &ValidateResult{
			Valid:   false,
			Message: "Name is required",
		}, nil
	}

	return &ValidateResult{
		Valid:   true,
		Message: "Validation passed",
	}, nil
}

// ProcessInput is the input for the Process activity
type ProcessInput struct {
	ID     string            `json:"id"`
	Params map[string]string `json:"params"`
}

// ProcessResult is the result of the Process activity
type ProcessResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Process executes the main processing logic
func (a *Activities) Process(ctx context.Context, input ProcessInput) (*ProcessResult, error) {
	slog.Info("Process activity started", "id", input.ID)

	// Simulate some processing work
	time.Sleep(100 * time.Millisecond)

	return &ProcessResult{
		Success: true,
		Message: fmt.Sprintf("Successfully processed %s", input.ID),
	}, nil
}

// NotifyInput is the input for the Notify activity
type NotifyInput struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NotifyResult is the result of the Notify activity
type NotifyResult struct {
	Sent bool `json:"sent"`
}

// Notify sends a notification
func (a *Activities) Notify(ctx context.Context, input NotifyInput) (*NotifyResult, error) {
	slog.Info("Notify activity", "id", input.ID, "status", input.Status, "message", input.Message)

	// Here you would send a notification (email, slack, webhook, etc.)

	return &NotifyResult{
		Sent: true,
	}, nil
}
