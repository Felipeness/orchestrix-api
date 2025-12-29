package temporal

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/sdk/client"

	"github.com/orchestrix/orchestrix-api/internal/core/domain"
	"github.com/orchestrix/orchestrix-api/internal/core/port"
)

// WorkflowExecutor implements port.WorkflowExecutor using Temporal
type WorkflowExecutor struct {
	client    client.Client
	taskQueue string
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor(c client.Client) *WorkflowExecutor {
	taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = "orchestrix-queue"
	}

	return &WorkflowExecutor{
		client:    c,
		taskQueue: taskQueue,
	}
}

// Execute starts a workflow execution in Temporal
func (e *WorkflowExecutor) Execute(ctx context.Context, workflow *domain.Workflow, input map[string]interface{}) (*port.ExecuteResult, error) {
	if !workflow.CanExecute() {
		return nil, domain.ErrWorkflowCannotExecute
	}

	workflowID := fmt.Sprintf("workflow-%s-%d", workflow.ID.String(), workflow.Version)

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: e.taskQueue,
	}

	// Parse workflow definition to get the workflow type
	def, err := workflow.ParseDefinition()
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow definition: %w", err)
	}

	// Start the workflow with the definition and input
	run, err := e.client.ExecuteWorkflow(ctx, options, "DynamicWorkflow", def, input)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	return &port.ExecuteResult{
		TemporalWorkflowID: run.GetID(),
		TemporalRunID:      run.GetRunID(),
	}, nil
}

// Cancel requests cancellation of a running workflow
func (e *WorkflowExecutor) Cancel(ctx context.Context, temporalWorkflowID string) error {
	return e.client.CancelWorkflow(ctx, temporalWorkflowID, "")
}

// GetStatus gets the current status of a workflow execution
func (e *WorkflowExecutor) GetStatus(ctx context.Context, temporalWorkflowID string) (string, error) {
	run := e.client.GetWorkflow(ctx, temporalWorkflowID, "")

	resp, err := e.client.DescribeWorkflowExecution(ctx, temporalWorkflowID, "")
	if err != nil {
		return "", fmt.Errorf("failed to describe workflow: %w", err)
	}

	_ = run // Keeping reference for potential future use

	return resp.WorkflowExecutionInfo.Status.String(), nil
}
