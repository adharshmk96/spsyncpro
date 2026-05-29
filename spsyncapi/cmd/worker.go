package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"spsyncapi/internal/config"
	"spsyncapi/internal/storage"
	"spsyncapi/internal/temporal"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the Temporal worker",
	Long:  "Connect to Temporal, reconcile DB state, and poll for backup/restore workflow tasks.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.SlogLevel(),
		}))

		db, err := storage.Open(cfg.DB)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}

		temporalClient, err := temporal.NewClient(cfg.Temporal)
		if err != nil {
			return fmt.Errorf("temporal client: %w", err)
		}
		defer temporalClient.Close()

		backupJobRepo := storage.NewBackupJobRepository(db)
		backupRunRepo := storage.NewBackupRunRepository(db)
		restoreJobRepo := storage.NewRestoreJobRepository(db)
		restoreRunRepo := storage.NewRestoreRunRepository(db)

		scheduler := temporal.NewScheduleOrchestrator(temporalClient, cfg.Temporal, logger)
		executor := temporal.NewRunExecutor(temporalClient, cfg.Temporal)

		reconcileDeps := temporal.ReconcileDeps{
			BackupJobRepo:  backupJobRepo,
			BackupRunRepo:  backupRunRepo,
			RestoreJobRepo: restoreJobRepo,
			RestoreRunRepo: restoreRunRepo,
			Scheduler:      scheduler,
			Executor:       executor,
			TemporalClient: temporalClient,
			Logger:         logger,
		}

		reconcileCtx, reconcileCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		if err := temporal.ReconcileOnStartup(reconcileCtx, reconcileDeps); err != nil {
			reconcileCancel()
			return fmt.Errorf("startup reconciliation: %w", err)
		}
		reconcileCancel()

		acts := &temporal.Activities{
			BackupRunRepo:  backupRunRepo,
			RestoreRunRepo: restoreRunRepo,
			BackupJobRepo:  backupJobRepo,
			RestoreJobRepo: restoreJobRepo,
			Logger:         logger,
		}
		w := temporal.NewWorker(temporalClient, cfg.Temporal, acts, logger)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		if cfg.Temporal.ReconcileInterval > 0 {
			go temporal.RunReconcileLoop(ctx, reconcileDeps, cfg.Temporal.ReconcileInterval, logger)
			logger.Info("temporal reconcile loop enabled", "interval", cfg.Temporal.ReconcileInterval)
		}

		logger.Info("starting temporal worker", "task_queue", cfg.Temporal.TaskQueue)
		if err := w.Run(ctx); err != nil {
			return fmt.Errorf("run temporal worker: %w", err)
		}
		logger.Info("temporal worker stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
