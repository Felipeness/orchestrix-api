package workflow

import (
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/orchestrix/orchestrix-api/internal/activity"
)

// DynamicWorkflowInput defines the input for the dynamic workflow
type DynamicWorkflowInput struct {
	ExecutionID string                 `json:"execution_id"`
	WorkflowID  string                 `json:"workflow_id"`
	Name        string                 `json:"name"`
	Definition  json.RawMessage        `json:"definition"`
	Input       map[string]interface{} `json:"input,omitempty"`
}

// DynamicWorkflowOutput defines the output of the dynamic workflow
type DynamicWorkflowOutput struct {
	ExecutionID string                   `json:"execution_id"`
	Status      string                   `json:"status"`
	StepResults []StepResult             `json:"step_results"`
	Output      map[string]interface{}   `json:"output,omitempty"`
	Error       string                   `json:"error,omitempty"`
	Duration    int64                    `json:"duration_ms"`
	Timestamp   int64                    `json:"timestamp"`
}

// StepResult represents the result of a single step execution
type StepResult struct {
	StepID      string      `json:"step_id"`
	StepName    string      `json:"step_name"`
	StepType    string      `json:"step_type"`
	Success     bool        `json:"success"`
	Output      interface{} `json:"output,omitempty"`
	Error       string      `json:"error,omitempty"`
	DurationMs  int64       `json:"duration_ms"`
}

// DynamicWorkflow executes a workflow based on its definition
func DynamicWorkflow(ctx workflow.Context, input DynamicWorkflowInput) (*DynamicWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("DynamicWorkflow started",
		"execution_id", input.ExecutionID,
		"workflow_id", input.WorkflowID,
		"name", input.Name)

	startTime := workflow.Now(ctx)
	output := &DynamicWorkflowOutput{
		ExecutionID: input.ExecutionID,
		StepResults: []StepResult{},
	}

	// Parse the workflow definition
	def, err := ParseDefinition(input.Definition)
	if err != nil {
		logger.Error("failed to parse definition", "error", err)
		output.Status = "failed"
		output.Error = fmt.Sprintf("failed to parse definition: %v", err)
		return output, nil
	}

	// Create context for storing step outputs
	stepOutputs := make(map[string]interface{})
	stepOutputs["input"] = input.Input

	// Default activity options
	defaultAO := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}

	// Execute each step
	for i, step := range def.Steps {
		stepStart := workflow.Now(ctx)
		logger.Info("Executing step", "step_id", step.ID, "step_name", step.Name, "step_type", step.Type)

		result, err := executeStep(ctx, step, stepOutputs, defaultAO)

		stepDuration := workflow.Now(ctx).Sub(stepStart).Milliseconds()

		stepResult := StepResult{
			StepID:     step.ID,
			StepName:   step.Name,
			StepType:   string(step.Type),
			DurationMs: stepDuration,
		}

		if err != nil {
			stepResult.Success = false
			stepResult.Error = err.Error()
			output.StepResults = append(output.StepResults, stepResult)

			if !step.ContinueOnError {
				logger.Error("step failed", "step_id", step.ID, "error", err)
				output.Status = "failed"
				output.Error = fmt.Sprintf("step %s failed: %v", step.ID, err)
				break
			}
			logger.Warn("step failed but continuing", "step_id", step.ID, "error", err)
		} else {
			stepResult.Success = true
			stepResult.Output = result
			output.StepResults = append(output.StepResults, stepResult)

			// Store step output for use in subsequent steps
			stepOutputs[step.ID] = result
			stepOutputs[fmt.Sprintf("step_%d", i)] = result
		}
	}

	// Set final status
	if output.Status == "" {
		output.Status = "completed"
	}

	// Execute on_success or on_error steps
	if output.Status == "completed" && len(def.OnSuccess) > 0 {
		for _, step := range def.OnSuccess {
			_, _ = executeStep(ctx, step, stepOutputs, defaultAO)
		}
	} else if output.Status == "failed" && len(def.OnError) > 0 {
		for _, step := range def.OnError {
			_, _ = executeStep(ctx, step, stepOutputs, defaultAO)
		}
	}

	endTime := workflow.Now(ctx)
	output.Duration = endTime.Sub(startTime).Milliseconds()
	output.Timestamp = endTime.Unix()
	output.Output = stepOutputs

	logger.Info("DynamicWorkflow completed",
		"execution_id", input.ExecutionID,
		"status", output.Status,
		"duration_ms", output.Duration)

	return output, nil
}

// executeStep executes a single step based on its type
func executeStep(ctx workflow.Context, step StepDefinition, stepOutputs map[string]interface{}, defaultAO workflow.ActivityOptions) (interface{}, error) {
	// Apply step-specific timeout if specified
	ao := defaultAO
	if step.Timeout != "" {
		if timeout, err := time.ParseDuration(step.Timeout); err == nil {
			ao.StartToCloseTimeout = timeout
		}
	}

	// Apply step-specific retry policy if specified
	if step.RetryPolicy != nil {
		ao.RetryPolicy = &temporal.RetryPolicy{
			MaximumAttempts: int32(step.RetryPolicy.MaxAttempts),
		}
		if step.RetryPolicy.InitialInterval != "" {
			if d, err := time.ParseDuration(step.RetryPolicy.InitialInterval); err == nil {
				ao.RetryPolicy.InitialInterval = d
			}
		}
		if step.RetryPolicy.MaxInterval != "" {
			if d, err := time.ParseDuration(step.RetryPolicy.MaxInterval); err == nil {
				ao.RetryPolicy.MaximumInterval = d
			}
		}
		if step.RetryPolicy.Multiplier > 0 {
			ao.RetryPolicy.BackoffCoefficient = step.RetryPolicy.Multiplier
		}
	}

	actCtx := workflow.WithActivityOptions(ctx, ao)

	switch step.Type {
	case StepTypeHTTP:
		return executeHTTPStep(actCtx, step.Config)

	case StepTypeDelay:
		return executeDelayStep(actCtx, step.Config)

	case StepTypeLog:
		return executeLogStep(actCtx, step.Config)

	case StepTypeNotify:
		return executeNotifyStep(actCtx, step.Config, stepOutputs)

	case StepTypeValidate:
		return executeValidateStep(actCtx, step.Config, stepOutputs)

	case StepTypeProcess:
		return executeProcessStep(actCtx, step.Config, stepOutputs)

	default:
		return nil, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

func executeHTTPStep(ctx workflow.Context, config map[string]interface{}) (*activity.HTTPResult, error) {
	cfg, err := ParseHTTPConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP config: %w", err)
	}

	input := activity.HTTPInput{
		URL:          cfg.URL,
		Method:       cfg.Method,
		Headers:      cfg.Headers,
		Body:         cfg.Body,
		SuccessCodes: cfg.SuccessCodes,
	}

	var result activity.HTTPResult
	err = workflow.ExecuteActivity(ctx, "HTTP", input).Get(ctx, &result)
	return &result, err
}

func executeDelayStep(ctx workflow.Context, config map[string]interface{}) (*activity.DelayResult, error) {
	cfg, err := ParseDelayConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid delay config: %w", err)
	}

	input := activity.DelayInput{
		Duration: cfg.Duration,
	}

	var result activity.DelayResult
	err = workflow.ExecuteActivity(ctx, "Delay", input).Get(ctx, &result)
	return &result, err
}

func executeLogStep(ctx workflow.Context, config map[string]interface{}) (*activity.LogResult, error) {
	cfg, err := ParseLogConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid log config: %w", err)
	}

	input := activity.LogInput{
		Level:   cfg.Level,
		Message: cfg.Message,
	}

	var result activity.LogResult
	err = workflow.ExecuteActivity(ctx, "Log", input).Get(ctx, &result)
	return &result, err
}

func executeNotifyStep(ctx workflow.Context, config map[string]interface{}, stepOutputs map[string]interface{}) (*activity.NotifyResult, error) {
	cfg, err := ParseNotifyConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid notify config: %w", err)
	}

	// Get execution ID from step outputs
	executionID := ""
	if input, ok := stepOutputs["input"].(map[string]interface{}); ok {
		if id, ok := input["execution_id"].(string); ok {
			executionID = id
		}
	}

	input := activity.NotifyInput{
		ID:      executionID,
		Status:  "completed",
		Message: cfg.Message,
	}

	var result activity.NotifyResult
	err = workflow.ExecuteActivity(ctx, "Notify", input).Get(ctx, &result)
	return &result, err
}

func executeValidateStep(ctx workflow.Context, config map[string]interface{}, stepOutputs map[string]interface{}) (*activity.ValidateResult, error) {
	input := activity.ValidateInput{}

	if id, ok := config["id"].(string); ok {
		input.ID = id
	} else if inputMap, ok := stepOutputs["input"].(map[string]interface{}); ok {
		if id, ok := inputMap["id"].(string); ok {
			input.ID = id
		}
	}

	if name, ok := config["name"].(string); ok {
		input.Name = name
	}

	var result activity.ValidateResult
	err := workflow.ExecuteActivity(ctx, "Validate", input).Get(ctx, &result)
	return &result, err
}

func executeProcessStep(ctx workflow.Context, config map[string]interface{}, stepOutputs map[string]interface{}) (*activity.ProcessResult, error) {
	input := activity.ProcessInput{
		Params: make(map[string]string),
	}

	if id, ok := config["id"].(string); ok {
		input.ID = id
	} else if inputMap, ok := stepOutputs["input"].(map[string]interface{}); ok {
		if id, ok := inputMap["id"].(string); ok {
			input.ID = id
		}
	}

	if params, ok := config["params"].(map[string]interface{}); ok {
		for k, v := range params {
			if s, ok := v.(string); ok {
				input.Params[k] = s
			}
		}
	}

	var result activity.ProcessResult
	err := workflow.ExecuteActivity(ctx, "Process", input).Get(ctx, &result)
	return &result, err
}
