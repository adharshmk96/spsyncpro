package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"

	"spsyncworker/internal/repository"
	"spsyncworker/internal/workflows"
)

type Reconciler struct {
	logger    *slog.Logger
	repo      *repository.JobsRepository
	tClient   client.Client
	taskQueue string
}

const debugLogPath = "/home/adharsh/dev/spsyncpro/spsyncui/.cursor/debug-efd826.log"

type debugEntry struct {
	SessionID    string         `json:"sessionId"`
	RunID        string         `json:"runId"`
	HypothesisID string         `json:"hypothesisId"`
	Location     string         `json:"location"`
	Message      string         `json:"message"`
	Data         map[string]any `json:"data"`
	Timestamp    int64          `json:"timestamp"`
}

func writeDebugLog(runID, hypothesisID, location, message string, data map[string]any) {
	entry := debugEntry{
		SessionID:    "efd826",
		RunID:        runID,
		HypothesisID: hypothesisID,
		Location:     location,
		Message:      message,
		Data:         data,
		Timestamp:    time.Now().UnixMilli(),
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return
	}

	file, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	_, _ = file.Write(append(bytes, '\n'))
}

func NewReconciler(
	logger *slog.Logger,
	repo *repository.JobsRepository,
	tClient client.Client,
	taskQueue string,
) *Reconciler {
	return &Reconciler{
		logger:    logger,
		repo:      repo,
		tClient:   tClient,
		taskQueue: taskQueue,
	}
}

func (r *Reconciler) RunForever(ctx context.Context, interval time.Duration) {
	r.logger.Info("Starting job reconciliation loop", "intervalSec", interval.Seconds())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.reconcileOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Stopping reconciliation loop")
			return
		case <-ticker.C:
			r.reconcileOnce(ctx)
		}
	}
}

func (r *Reconciler) reconcileOnce(ctx context.Context) {
	jobs, err := r.repo.ListSchedulableActiveJobs(ctx)
	if err != nil {
		r.logger.Error("Failed to list active jobs for reconciliation", "error", err)
		return
	}
	runID := "before-fix"
	// #region agent log
	writeDebugLog(runID, "H2", "internal/bootstrap/reconcile.go:92", "loaded schedulable active jobs", map[string]any{
		"jobCount": len(jobs),
	})
	// #endregion

	for _, job := range jobs {
		// #region agent log
		writeDebugLog(runID, "H1", "internal/bootstrap/reconcile.go:100", "considering job for workflow start", map[string]any{
			"jobId":         job.ID,
			"runMode":       job.RunMode,
			"status":        job.Status,
			"hasNextRunAt":  job.NextRunAt != nil,
			"hasLastRunAt":  job.LastRunAt != nil,
			"nextRunAtUnix": func() int64 {
				if job.NextRunAt == nil {
					return 0
				}
				return job.NextRunAt.Unix()
			}(),
			"lastRunAtUnix": func() int64 {
				if job.LastRunAt == nil {
					return 0
				}
				return job.LastRunAt.Unix()
			}(),
		})
		// #endregion
		startOpts := client.StartWorkflowOptions{
			ID:        "job/" + job.ID,
			TaskQueue: r.taskQueue,
		}

		recurrence := ""
		if job.Recurrence != nil {
			recurrence = *job.Recurrence
		}

		input := workflows.JobWorkflowInput{
			JobID:             job.ID,
			RunMode:           job.RunMode,
			Recurrence:        recurrence,
			StorageConfigJSON: string(job.StorageConfig),
			FilterConfigJSON:  string(job.FilterConfig),
		}

		_, err := r.tClient.ExecuteWorkflow(ctx, startOpts, workflows.JobExecutionWorkflow, input)
		if err != nil {
			var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
			if errors.As(err, &alreadyStarted) {
				// #region agent log
				writeDebugLog(runID, "H3", "internal/bootstrap/reconcile.go:131", "workflow already running for job", map[string]any{
					"jobId": job.ID,
				})
				// #endregion
				if job.ManualRunPending() {
					if sigErr := r.tClient.SignalWorkflow(ctx, startOpts.ID, "", workflows.RunNowSignalName, nil); sigErr != nil {
						r.logger.Error("Failed to signal run-now for job", "jobID", job.ID, "error", sigErr)
						continue
					}
					if clearErr := r.repo.ClearManualRunAt(ctx, job.ID); clearErr != nil {
						r.logger.Error("Failed to clear manualRunAt after signal", "jobID", job.ID, "error", clearErr)
					}
					r.logger.Info("Signaled run-now for active job workflow", "jobID", job.ID)
				}
				continue
			}
			r.logger.Error("Failed to start job workflow", "jobID", job.ID, "error", err)
			continue
		}
		// #region agent log
		writeDebugLog(runID, "H3", "internal/bootstrap/reconcile.go:141", "workflow started for job", map[string]any{
			"jobId": job.ID,
		})
		// #endregion

		if job.ManualRunPending() {
			if clearErr := r.repo.ClearManualRunAt(ctx, job.ID); clearErr != nil {
				r.logger.Error("Failed to clear manualRunAt after workflow start", "jobID", job.ID, "error", clearErr)
			}
		}

		r.logger.Info("Started workflow for active job", "jobID", job.ID)
	}
}
