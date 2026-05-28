package temporal

import (
	"context"
	"fmt"
	"log/slog"

	"spsyncapi/internal/config"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Worker runs the Temporal worker process.
type Worker struct {
	worker worker.Worker
	logger *slog.Logger
}

// NewWorker registers workflows and activities on a Temporal worker.
func NewWorker(c client.Client, cfg config.TemporalConfig, acts *Activities, logger *slog.Logger) *Worker {
	w := worker.New(c, cfg.TaskQueue, worker.Options{})
	w.RegisterWorkflow(BackupRunWorkflow)
	w.RegisterWorkflow(RestoreRunWorkflow)
	w.RegisterWorkflow(ScheduledBackupWorkflow)
	acts.Register(w)
	return &Worker{worker: w, logger: logger}
}

// Run blocks until ctx is cancelled, then stops the worker gracefully.
func (w *Worker) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- w.worker.Run(worker.InterruptCh())
	}()

	select {
	case <-ctx.Done():
		w.logger.Info("stopping temporal worker")
		w.worker.Stop()
		<-errCh
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("temporal worker: %w", err)
		}
		return nil
	}
}
