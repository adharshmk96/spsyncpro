package temporal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/storage"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// ReconcileDeps holds storage and Temporal dependencies for reconciliation.
type ReconcileDeps struct {
	BackupJobRepo   *storage.BackupJobRepository
	BackupRunRepo   *storage.BackupRunRepository
	RestoreJobRepo  *storage.RestoreJobRepository
	RestoreRunRepo  *storage.RestoreRunRepository
	Scheduler       *ScheduleOrchestrator
	Executor        *RunExecutor
	TemporalClient  client.Client
	Logger          *slog.Logger
}

// ReconcileOnStartup syncs schedules and resumes incomplete runs (alias for Reconcile).
func ReconcileOnStartup(ctx context.Context, deps ReconcileDeps) error {
	return Reconcile(ctx, deps)
}

// Reconcile rebuilds Temporal schedules and workflows from application database state.
func Reconcile(ctx context.Context, deps ReconcileDeps) error {
	backupJobs, err := deps.BackupJobRepo.ListAllActive()
	if err != nil {
		return fmt.Errorf("reconcile: list backup jobs: %w", err)
	}
	for i := range backupJobs {
		if err := reconcileBackupJob(ctx, deps, &backupJobs[i]); err != nil {
			deps.Logger.Error("reconcile: backup job", "job_id", backupJobs[i].ID, "error", err)
		}
	}

	restoreJobs, err := deps.RestoreJobRepo.ListAllActive()
	if err != nil {
		return fmt.Errorf("reconcile: list restore jobs: %w", err)
	}
	for i := range restoreJobs {
		if err := reconcileRestoreJob(ctx, deps, &restoreJobs[i]); err != nil {
			deps.Logger.Error("reconcile: restore job", "job_id", restoreJobs[i].ID, "error", err)
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

	deps.Logger.Info("reconciliation complete",
		"backup_jobs", len(backupJobs),
		"restore_jobs", len(restoreJobs),
		"incomplete_backup_runs", len(backupRuns),
		"incomplete_restore_runs", len(restoreRuns),
	)
	return nil
}

func reconcileBackupJob(ctx context.Context, deps ReconcileDeps, job *storage.BackupJob) error {
	if !job.Active {
		return deps.Scheduler.DeleteJobSchedule(ctx, job.ID)
	}

	if backupjob.UsesRunStarterSchedule(job) {
		if err := deps.Scheduler.DeleteJobSchedule(ctx, job.ID); err != nil {
			return fmt.Errorf("clear schedule: %w", err)
		}
		return reconcileRunStarterBackupJob(ctx, deps, job)
	}

	if err := deps.Scheduler.SyncJob(ctx, job); err != nil {
		return fmt.Errorf("sync schedule: %w", err)
	}
	return nil
}

func reconcileRunStarterBackupJob(ctx context.Context, deps ReconcileDeps, job *storage.BackupJob) error {
	existing, err := deps.BackupRunRepo.FindIncompleteByJobID(job.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return deps.Executor.ResumeBackupRunIfNeeded(ctx, RunWorkflowInput{
			RunID:    existing.ID,
			JobID:    job.ID,
			MemberID: job.MemberID,
			Kind:     RunKindBackup,
			Resume:   true,
		})
	}

	at, ok := pendingBackupRunAt(job)
	if !ok {
		return nil
	}

	run := &storage.BackupRun{
		ID:        uuid.NewString(),
		JobID:     job.ID,
		MemberID:  job.MemberID,
		CreatedAt: time.Now().UTC(),
	}
	if err := deps.BackupRunRepo.Create(run); err != nil {
		return fmt.Errorf("create pending backup run: %w", err)
	}

	in := RunWorkflowInput{
		RunID:    run.ID,
		JobID:    job.ID,
		MemberID: job.MemberID,
		Kind:     RunKindBackup,
		Resume:   true,
	}
	if at.After(time.Now().UTC()) {
		return deps.Executor.StartBackupRunAt(ctx, in, at)
	}
	return deps.Executor.StartBackupRun(ctx, in)
}

func pendingBackupRunAt(job *storage.BackupJob) (time.Time, bool) {
	now := time.Now().UTC()
	if job.ScheduleOneTime != nil {
		t := job.ScheduleOneTime.UTC()
		if job.LastRun == nil {
			return t, true
		}
		return time.Time{}, false
	}
	if job.NextRun != nil && job.LastRun == nil {
		return job.NextRun.UTC(), true
	}
	if job.NextRun != nil && job.NextRun.After(now) {
		return job.NextRun.UTC(), true
	}
	return time.Time{}, false
}

func reconcileRestoreJob(ctx context.Context, deps ReconcileDeps, job *storage.RestoreJob) error {
	if !job.Active {
		return nil
	}

	existing, err := deps.RestoreRunRepo.FindIncompleteByJobID(job.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return deps.Executor.ResumeRestoreRunIfNeeded(ctx, RunWorkflowInput{
			RunID:    existing.ID,
			JobID:    job.ID,
			MemberID: job.MemberID,
			Kind:     RunKindRestore,
			Resume:   true,
		})
	}

	if job.LastRun != nil {
		return nil
	}

	run := &storage.RestoreRun{
		ID:        uuid.NewString(),
		JobID:     job.ID,
		MemberID:  job.MemberID,
		CreatedAt: time.Now().UTC(),
	}
	if err := deps.RestoreRunRepo.Create(run); err != nil {
		return fmt.Errorf("create pending restore run: %w", err)
	}

	in := RunWorkflowInput{
		RunID:    run.ID,
		JobID:    job.ID,
		MemberID: job.MemberID,
		Kind:     RunKindRestore,
		Resume:   true,
	}

	if job.StartAt != nil {
		at := job.StartAt.UTC()
		if at.After(time.Now().UTC()) {
			return deps.Executor.StartRestoreRunAt(ctx, in, at)
		}
	}
	return deps.Executor.StartRestoreRun(ctx, in)
}

// TemporalHealthy reports whether the Temporal cluster accepts client requests.
func TemporalHealthy(ctx context.Context, c client.Client) bool {
	if c == nil {
		return false
	}
	_, err := c.CheckHealth(ctx, &client.CheckHealthRequest{})
	return err == nil
}
