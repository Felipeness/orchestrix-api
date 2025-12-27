package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/orchestrix/orchestrix-api/internal/activity"
	"github.com/orchestrix/orchestrix-api/internal/workflow"
)

const (
	defaultTaskQueue = "orchestrix-queue"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Temporal client
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		slog.Error("failed to create temporal client", "error", err)
		os.Exit(1)
	}
	defer c.Close()

	// Task queue
	taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = defaultTaskQueue
	}

	// Worker
	w := worker.New(c, taskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflow.ProcessWorkflow)

	// Register activities
	activities := &activity.Activities{}
	w.RegisterActivity(activities)

	// Start worker
	go func() {
		slog.Info("starting temporal worker", "taskQueue", taskQueue)
		if err := w.Run(worker.InterruptCh()); err != nil {
			slog.Error("worker error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker...")
	w.Stop()
	slog.Info("worker exited")
}
