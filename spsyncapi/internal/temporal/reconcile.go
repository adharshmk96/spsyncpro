package temporal

import (
	"context"
	"fmt"
	"log/slog"

	"spsyncapi/internal/storage"
)

// ReconcileDeps holds storage dependencies for startup reconciliation.
type ReconcileDeps struct {
	BackupJobRepo  *storage.BackupJobRepository
	BackupRunRepo  *storage.BackupRunRepository
	RestoreRunRepo *storage.RestoreRunRepository
	Scheduler      *ScheduleOrchestrator
	Executor       *RunExecutor
	Logger         *slog.Logger
}

// ReconcileOnStartup syncs schedules and resumes incomplete runs.
func ReconcileOnStartup(ctx context.Context, deps ReconcileDeps) error {
	jobs, err := deps.BackupJobRepo.ListAllActive()
	if err != nil {
		return fmt.Errorf("reconcile: list backup jobs: %w", err)
	}
	for i := range jobs {
		if err := deps.Scheduler.SyncJob(ctx, &jobs[i]); err != nil {
			deps.Logger.Error("reconcile: sync backup job schedule", "job_id", jobs[i].ID, "error", err)
		}
	}

	backupRuns, err := deps.BackupRunRepo.ListIncomplete()
	if err != nil {
		return fmt.Errorf("reconcile: list incomplete backup runs: %w", err)
	}
	for i := range backupRuns {
		run := &backupRuns[i]
		if err := deps.Executor.ResumeBackupRunIfNeeded(ctx, RunWorkflowInput{
			RunID:    run.ID,
			JobID:    run.JobID,
			MemberID: run.MemberID,
			Kind:     RunKindBackup,
			Resume:   true,
		}); err != nil {
			deps.Logger.Error("reconcile: resume backup run", "run_id", run.ID, "error", err)
		}
	}

	restoreRuns, err := deps.RestoreRunRepo.ListIncomplete()
	if err != nil {
		return fmt.Errorf("reconcile: list incomplete restore runs: %w", err)
	}
	for i := range restoreRuns {
		run := &restoreRuns[i]
		if err := deps.Executor.ResumeRestoreRunIfNeeded(ctx, RunWorkflowInput{
			RunID:    run.ID,
			JobID:    run.JobID,
			MemberID: run.MemberID,
			Kind:     RunKindRestore,
			Resume:   true,
		}); err != nil {
			deps.Logger.Error("reconcile: resume restore run", "run_id", run.ID, "error", err)
		}
	}

	deps.Logger.Info("startup reconciliation complete",
		"backup_jobs", len(jobs),
		"incomplete_backup_runs", len(backupRuns),
		"incomplete_restore_runs", len(restoreRuns),
	)
	return nil
}
