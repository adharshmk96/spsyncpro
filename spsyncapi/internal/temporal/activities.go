package temporal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"spsyncapi/internal/storage"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const defaultTransferDelay = 5 * time.Second

const dummyFileCount = 5

// Activities holds dependencies for Temporal activity implementations.
type Activities struct {
	BackupRunRepo  *storage.BackupRunRepository
	RestoreRunRepo *storage.RestoreRunRepository
	BackupJobRepo  *storage.BackupJobRepository
	RestoreJobRepo *storage.RestoreJobRepository
	Logger         *slog.Logger
	TransferDelay  time.Duration
}

func (a *Activities) transferDelay() time.Duration {
	if a.TransferDelay > 0 {
		return a.TransferDelay
	}
	return defaultTransferDelay
}

// CreateBackupRun inserts a backup run row for a scheduled execution.
func (a *Activities) CreateBackupRun(ctx context.Context, in ScheduledBackupInput) (string, error) {
	if _, err := a.BackupJobRepo.FindActiveByID(in.JobID, in.MemberID); err != nil {
		return "", fmt.Errorf("create backup run activity: validate job: %w", err)
	}

	now := time.Now().UTC()
	run := &storage.BackupRun{
		ID:        uuid.NewString(),
		JobID:     in.JobID,
		MemberID:  in.MemberID,
		CreatedAt: now,
	}
	if err := a.BackupRunRepo.Create(run); err != nil {
		return "", fmt.Errorf("create backup run activity: %w", err)
	}
	a.Logger.Info("backup run created", "run_id", run.ID, "job_id", in.JobID)
	return run.ID, nil
}

// CreateRestoreRun inserts a restore run row.
func (a *Activities) CreateRestoreRun(ctx context.Context, in ScheduledBackupInput) (string, error) {
	if _, err := a.RestoreJobRepo.FindActiveByID(in.JobID, in.MemberID); err != nil {
		return "", fmt.Errorf("create restore run activity: validate job: %w", err)
	}

	now := time.Now().UTC()
	run := &storage.RestoreRun{
		ID:        uuid.NewString(),
		JobID:     in.JobID,
		MemberID:  in.MemberID,
		CreatedAt: now,
	}
	if err := a.RestoreRunRepo.Create(run); err != nil {
		return "", fmt.Errorf("create restore run activity: %w", err)
	}
	a.Logger.Info("restore run created", "run_id", run.ID, "job_id", in.JobID)
	return run.ID, nil
}

// TransferFiles performs dummy file transfers (5 files, 5s each by default).
func (a *Activities) TransferFiles(ctx context.Context, in TransferFilesInput) error {
	switch in.Kind {
	case RunKindBackup:
		return a.transferBackupFiles(ctx, in)
	case RunKindRestore:
		return a.transferRestoreFiles(ctx, in)
	default:
		return fmt.Errorf("transfer files activity: unknown kind %q", in.Kind)
	}
}

func (a *Activities) transferBackupFiles(ctx context.Context, in TransferFilesInput) error {
	run, err := a.BackupRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return fmt.Errorf("transfer backup files: find run: %w", err)
	}
	if run.EndAt != nil {
		return nil
	}

	now := time.Now().UTC()
	if run.StartAt == nil {
		run.StartAt = &now
		if err := a.BackupRunRepo.Update(run); err != nil {
			return fmt.Errorf("transfer backup files: set start_at: %w", err)
		}
	}

	delay := a.transferDelay()
	for i := 1; i <= dummyFileCount; i++ {
		if err := ctx.Err(); err != nil {
			return a.finalizeBackupRun(ctx, run, true)
		}
		recordHeartbeat(ctx, i)

		path := DummyFilePath(in.JobID, i)
		existing, err := a.BackupRunRepo.FindFileTransferByRunAndPath(in.RunID, path)
		if err != nil {
			return fmt.Errorf("transfer backup files: find existing transfer: %w", err)
		}
		if existing == nil {
			start := time.Now().UTC()
			if err := sleepOrCancel(ctx, delay); err != nil {
				return a.finalizeBackupRun(ctx, run, true)
			}
			end := time.Now().UTC()
			ft := &storage.BackupRunFileTransfer{
				ID:        uuid.NewString(),
				RunID:     in.RunID,
				FilePath:  path,
				StartAt:   &start,
				EndAt:     &end,
				CreatedAt: start,
			}
			if err := a.BackupRunRepo.CreateFileTransfer(ft); err != nil {
				return fmt.Errorf("transfer backup files: create transfer: %w", err)
			}
			a.Logger.Info("backup file transferred", "run_id", in.RunID, "path", path)
		}
	}

	return a.finalizeBackupRun(ctx, run, false)
}

func (a *Activities) transferRestoreFiles(ctx context.Context, in TransferFilesInput) error {
	run, err := a.RestoreRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return fmt.Errorf("transfer restore files: find run: %w", err)
	}
	if run.EndAt != nil {
		return nil
	}

	now := time.Now().UTC()
	if run.StartAt == nil {
		run.StartAt = &now
		if err := a.RestoreRunRepo.Update(run); err != nil {
			return fmt.Errorf("transfer restore files: set start_at: %w", err)
		}
	}

	delay := a.transferDelay()
	for i := 1; i <= dummyFileCount; i++ {
		if err := ctx.Err(); err != nil {
			return a.finalizeRestoreRun(ctx, run, true)
		}
		recordHeartbeat(ctx, i)

		path := DummyFilePath(in.JobID, i)
		existing, err := a.RestoreRunRepo.FindFileTransferByRunAndPath(in.RunID, path)
		if err != nil {
			return fmt.Errorf("transfer restore files: find existing transfer: %w", err)
		}
		if existing == nil {
			start := time.Now().UTC()
			if err := sleepOrCancel(ctx, delay); err != nil {
				return a.finalizeRestoreRun(ctx, run, true)
			}
			end := time.Now().UTC()
			ft := &storage.RestoreRunFileTransfer{
				ID:        uuid.NewString(),
				RunID:     in.RunID,
				FilePath:  path,
				StartAt:   &start,
				EndAt:     &end,
				CreatedAt: start,
			}
			if err := a.RestoreRunRepo.CreateFileTransfer(ft); err != nil {
				return fmt.Errorf("transfer restore files: create transfer: %w", err)
			}
			a.Logger.Info("restore file transferred", "run_id", in.RunID, "path", path)
		}
	}

	return a.finalizeRestoreRun(ctx, run, false)
}

func (a *Activities) finalizeBackupRun(ctx context.Context, run *storage.BackupRun, cancelled bool) error {
	if err := ctx.Err(); err != nil && cancelled {
		now := time.Now().UTC()
		run.EndAt = &now
		if updateErr := a.BackupRunRepo.Update(run); updateErr != nil {
			return fmt.Errorf("finalize backup run: %w", updateErr)
		}
		return temporal.NewCanceledError("backup run stopped")
	}
	if cancelled {
		now := time.Now().UTC()
		run.EndAt = &now
		if err := a.BackupRunRepo.Update(run); err != nil {
			return fmt.Errorf("finalize backup run: %w", err)
		}
		return temporal.NewCanceledError("backup run stopped")
	}
	now := time.Now().UTC()
	run.EndAt = &now
	if err := a.BackupRunRepo.Update(run); err != nil {
		return fmt.Errorf("finalize backup run: %w", err)
	}
	return nil
}

func (a *Activities) finalizeRestoreRun(ctx context.Context, run *storage.RestoreRun, cancelled bool) error {
	if err := ctx.Err(); err != nil && cancelled {
		now := time.Now().UTC()
		run.EndAt = &now
		if updateErr := a.RestoreRunRepo.Update(run); updateErr != nil {
			return fmt.Errorf("finalize restore run: %w", updateErr)
		}
		return temporal.NewCanceledError("restore run stopped")
	}
	if cancelled {
		now := time.Now().UTC()
		run.EndAt = &now
		if err := a.RestoreRunRepo.Update(run); err != nil {
			return fmt.Errorf("finalize restore run: %w", err)
		}
		return temporal.NewCanceledError("restore run stopped")
	}
	now := time.Now().UTC()
	run.EndAt = &now
	if err := a.RestoreRunRepo.Update(run); err != nil {
		return fmt.Errorf("finalize restore run: %w", err)
	}
	return nil
}

func recordHeartbeat(ctx context.Context, details interface{}) {
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, details)
	}
}

func sleepOrCancel(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Register wires activity methods on the worker.
func (a *Activities) Register(w interface {
	RegisterActivityWithOptions(fn interface{}, opts activity.RegisterOptions)
}) {
	w.RegisterActivityWithOptions(a.CreateBackupRun, activity.RegisterOptions{Name: "CreateBackupRun"})
	w.RegisterActivityWithOptions(a.CreateRestoreRun, activity.RegisterOptions{Name: "CreateRestoreRun"})
	w.RegisterActivityWithOptions(a.TransferFiles, activity.RegisterOptions{Name: transferActivityName})
}

// ErrRunAlreadyComplete is returned when starting a workflow for a finished run.
var ErrRunAlreadyComplete = errors.New("run already complete")
