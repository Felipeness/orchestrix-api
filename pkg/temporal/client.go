package temporal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"go.temporal.io/sdk/client"
)

// WorkflowRun is an alias for the Temporal SDK WorkflowRun type
type WorkflowRun = client.WorkflowRun

var (
	temporalClient client.Client
	once           sync.Once
	initErr        error
)

// GetClient returns the singleton Temporal client
func GetClient() (client.Client, error) {
	once.Do(func() {
		host := os.Getenv("TEMPORAL_HOST")
		if host == "" {
			host = "localhost:7233"
		}

		var err error
		temporalClient, err = client.Dial(client.Options{
			HostPort: host,
		})
		if err != nil {
			initErr = fmt.Errorf("failed to create temporal client: %w", err)
			slog.Error("temporal client init failed", "error", err)
			return
		}

		slog.Info("temporal client connected", "host", host)
	})

	if initErr != nil {
		return nil, initErr
	}
	return temporalClient, nil
}

// Close closes the Temporal client
func Close() {
	if temporalClient != nil {
		temporalClient.Close()
	}
}

// StartWorkflowOptions holds options for starting a workflow
type StartWorkflowOptions struct {
	ID        string
	TaskQueue string
}

// GetTaskQueue returns the default task queue or the provided one
func GetTaskQueue() string {
	queue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if queue == "" {
		queue = "orchestrix-queue"
	}
	return queue
}

// ExecuteWorkflow starts a workflow and returns the run
func ExecuteWorkflow(ctx context.Context, workflowID string, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	options := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: GetTaskQueue(),
	}

	return c.ExecuteWorkflow(ctx, options, workflow, args...)
}

// GetWorkflowHistory gets workflow execution details
func GetWorkflowHistory(ctx context.Context, workflowID, runID string) (client.WorkflowRun, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	return c.GetWorkflow(ctx, workflowID, runID), nil
}

// TerminateWorkflow terminates a running workflow
func TerminateWorkflow(ctx context.Context, workflowID, runID, reason string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	return c.TerminateWorkflow(ctx, workflowID, runID, reason)
}

// CancelWorkflow requests cancellation of a workflow
func CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	return c.CancelWorkflow(ctx, workflowID, runID)
}
