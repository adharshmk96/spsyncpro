package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	dailySimulationInterval     = 30 * time.Second
	weeklySimulationInterval    = 45 * time.Second
	monthlySimulationInterval   = 60 * time.Second
	defaultImmediateInterval    = 20 * time.Second

	// RunNowSignalName is sent by the reconciler when the UI requests an immediate extra run
	// while a long-running workflow is already executing.
	RunNowSignalName = "run-now"
)

type JobWorkflowInput struct {
	JobID             string
	RunMode           string
	Recurrence        string
	StorageConfigJSON string
	FilterConfigJSON  string
}

type JobActivityInput struct {
	JobID             string
	RunMode           string
	Recurrence        string
	StorageConfigJSON string
	FilterConfigJSON  string
}

func JobExecutionWorkflow(ctx workflow.Context, input JobWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting job execution workflow", "jobID", input.JobID, "runMode", input.RunMode)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    15 * time.Second,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)
	runNowCh := workflow.GetSignalChannel(ctx, RunNowSignalName)

	activityInput := JobActivityInput{
		JobID:             input.JobID,
		RunMode:           input.RunMode,
		Recurrence:        input.Recurrence,
		StorageConfigJSON: input.StorageConfigJSON,
		FilterConfigJSON:  input.FilterConfigJSON,
	}

	if err := workflow.ExecuteActivity(ctx, "LogJobConfigActivity", activityInput).Get(ctx, nil); err != nil {
		return fmt.Errorf("log config activity failed: %w", err)
	}

	// One-time jobs run exactly once; recurring/immediate jobs keep simulating.
	for {
		if err := workflow.ExecuteActivity(ctx, "SimulateRunActivity", activityInput).Get(ctx, nil); err != nil {
			return fmt.Errorf("simulate run activity failed: %w", err)
		}

		if input.RunMode == "ONE_TIME_AT" {
			logger.Info("One-time job simulation complete", "jobID", input.JobID)
			return nil
		}

		interval := recurrenceInterval(input.Recurrence)
		timerCtx, cancelTimer := workflow.WithCancel(ctx)
		timer := workflow.NewTimer(timerCtx, interval)

		selector := workflow.NewSelector(ctx)
		selector.AddFuture(timer, func(f workflow.Future) {
			_ = f.Get(ctx, nil)
		})
		selector.AddReceive(runNowCh, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, nil)
			cancelTimer()
		})
		selector.Select(ctx)
	}
}

func recurrenceInterval(recurrence string) time.Duration {
	switch recurrence {
	case "DAILY":
		return dailySimulationInterval
	case "WEEKLY":
		return weeklySimulationInterval
	case "MONTHLY":
		return monthlySimulationInterval
	default:
		return defaultImmediateInterval
	}
}
