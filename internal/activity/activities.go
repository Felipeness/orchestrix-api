package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Activities holds all activity implementations
type Activities struct {
	HTTPClient *http.Client
}

// NewActivities creates a new Activities instance
func NewActivities() *Activities {
	return &Activities{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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

// HTTPInput is the input for the HTTP activity
type HTTPInput struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         interface{}       `json:"body,omitempty"`
	Timeout      int               `json:"timeout_seconds,omitempty"`
	SuccessCodes []int             `json:"success_codes,omitempty"`
}

// HTTPResult is the result of the HTTP activity
type HTTPResult struct {
	StatusCode int               `json:"status_code"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
	Success    bool              `json:"success"`
	Error      string            `json:"error,omitempty"`
}

// HTTP performs an HTTP request
func (a *Activities) HTTP(ctx context.Context, input HTTPInput) (*HTTPResult, error) {
	slog.Info("HTTP activity started", "url", input.URL, "method", input.Method)

	if input.Method == "" {
		input.Method = "GET"
	}
	if len(input.SuccessCodes) == 0 {
		input.SuccessCodes = []int{200, 201, 202, 204}
	}

	var bodyReader io.Reader
	if input.Body != nil {
		bodyBytes, err := json.Marshal(input.Body)
		if err != nil {
			return &HTTPResult{Success: false, Error: err.Error()}, nil
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, bodyReader)
	if err != nil {
		return &HTTPResult{Success: false, Error: err.Error()}, nil
	}

	for k, v := range input.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && input.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := a.HTTPClient
	if input.Timeout > 0 {
		client = &http.Client{Timeout: time.Duration(input.Timeout) * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &HTTPResult{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	success := false
	for _, code := range input.SuccessCodes {
		if resp.StatusCode == code {
			success = true
			break
		}
	}

	slog.Info("HTTP activity completed", "url", input.URL, "status", resp.StatusCode, "success", success)

	return &HTTPResult{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Headers:    headers,
		Success:    success,
	}, nil
}

// DelayInput is the input for the Delay activity
type DelayInput struct {
	Duration string `json:"duration"` // e.g., "5s", "1m"
}

// DelayResult is the result of the Delay activity
type DelayResult struct {
	Delayed bool `json:"delayed"`
}

// Delay pauses execution for a specified duration
func (a *Activities) Delay(ctx context.Context, input DelayInput) (*DelayResult, error) {
	duration, err := time.ParseDuration(input.Duration)
	if err != nil {
		duration = time.Second
	}

	slog.Info("Delay activity", "duration", duration)
	time.Sleep(duration)

	return &DelayResult{Delayed: true}, nil
}

// LogInput is the input for the Log activity
type LogInput struct {
	Level   string                 `json:"level"` // info, warn, error
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// LogResult is the result of the Log activity
type LogResult struct {
	Logged bool `json:"logged"`
}

// Log writes a log entry
func (a *Activities) Log(ctx context.Context, input LogInput) (*LogResult, error) {
	attrs := []any{"message", input.Message}
	for k, v := range input.Data {
		attrs = append(attrs, k, v)
	}

	switch input.Level {
	case "error":
		slog.Error("Workflow log", attrs...)
	case "warn":
		slog.Warn("Workflow log", attrs...)
	default:
		slog.Info("Workflow log", attrs...)
	}

	return &LogResult{Logged: true}, nil
}

// WebhookInput is the input for the Webhook activity
type WebhookInput struct {
	URL     string                 `json:"url"`
	Payload map[string]interface{} `json:"payload"`
}

// WebhookResult is the result of the Webhook activity
type WebhookResult struct {
	StatusCode int    `json:"status_code"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// Webhook sends a webhook notification
func (a *Activities) Webhook(ctx context.Context, input WebhookInput) (*WebhookResult, error) {
	slog.Info("Webhook activity", "url", input.URL)

	payload, _ := json.Marshal(input.Payload)

	req, err := http.NewRequestWithContext(ctx, "POST", input.URL, bytes.NewReader(payload))
	if err != nil {
		return &WebhookResult{Success: false, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return &WebhookResult{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	return &WebhookResult{
		StatusCode: resp.StatusCode,
		Success:    success,
	}, nil
}
