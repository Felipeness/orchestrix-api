package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/orchestrix/orchestrix-api/internal/activity"
)

// ProcessWorkflowInput defines the input for the process workflow
type ProcessWorkflowInput struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

// ProcessWorkflowOutput defines the output of the process workflow
type ProcessWorkflowOutput struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Result    string `json:"result"`
	Duration  int64  `json:"duration_ms"`
	Timestamp int64  `json:"timestamp"`
}

// ProcessWorkflow is the main workflow for processing tasks
func ProcessWorkflow(ctx workflow.Context, input ProcessWorkflowInput) (*ProcessWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("ProcessWorkflow started", "id", input.ID, "name", input.Name)

	startTime := workflow.Now(ctx)

	// Activity options with retry policy
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Validate input
	var validateResult activity.ValidateResult
	err := workflow.ExecuteActivity(ctx, "Validate", activity.ValidateInput{
		ID:   input.ID,
		Name: input.Name,
	}).Get(ctx, &validateResult)
	if err != nil {
		logger.Error("validation failed", "error", err)
		return nil, err
	}

	// Step 2: Process
	var processResult activity.ProcessResult
	err = workflow.ExecuteActivity(ctx, "Process", activity.ProcessInput{
		ID:     input.ID,
		Params: input.Params,
	}).Get(ctx, &processResult)
	if err != nil {
		logger.Error("processing failed", "error", err)
		return nil, err
	}

	// Step 3: Notify (fire and forget - we don't fail the workflow if notification fails)
	notifyCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
	_ = workflow.ExecuteActivity(notifyCtx, "Notify", activity.NotifyInput{
		ID:      input.ID,
		Status:  "completed",
		Message: processResult.Message,
	})

	endTime := workflow.Now(ctx)
	duration := endTime.Sub(startTime).Milliseconds()

	output := &ProcessWorkflowOutput{
		ID:        input.ID,
		Status:    "completed",
		Result:    processResult.Message,
		Duration:  duration,
		Timestamp: endTime.Unix(),
	}

	logger.Info("ProcessWorkflow completed", "id", input.ID, "duration_ms", duration)
	return output, nil
}
