package temporal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/restorejob"
	"spsyncapi/internal/storage"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const (
	defaultMetadataFetchDelay = 1 * time.Minute
	defaultTransferDelay      = 30 * time.Second
	metadataFileCount         = 100
)

const (
	fetchFileMetadataActivityName  = "FetchFileMetadata"
	transferSingleFileActivityName = "TransferSingleFile"
	finalizeRunActivityName        = "FinalizeRun"
)

// Activities holds dependencies for Temporal activity implementations.
type Activities struct {
	BackupRunRepo      *storage.BackupRunRepository
	RestoreRunRepo     *storage.RestoreRunRepository
	BackupJobRepo      *storage.BackupJobRepository
	RestoreJobRepo     *storage.RestoreJobRepository
	OrgRepo            *storage.OrganizationRepository
	BucketStoreRepo    *storage.BucketStoreRepository
	Encryptor          *crypto.SecretEncryptor
	Logger             *slog.Logger
	MetadataFetchDelay time.Duration
	TransferDelay      time.Duration
}

func (a *Activities) metadataFetchDelay() time.Duration {
	if a.MetadataFetchDelay > 0 {
		return a.MetadataFetchDelay
	}
	return defaultMetadataFetchDelay
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
	if err := a.markBackupRunStarted(in.JobID, in.MemberID, now); err != nil {
		return "", err
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

// FetchFileMetadata simulates discovering files to transfer and returns sample paths.
func (a *Activities) FetchFileMetadata(ctx context.Context, in FetchFileMetadataInput) (FetchFileMetadataOutput, error) {
	switch in.Kind {
	case RunKindBackup:
		return a.fetchBackupFileMetadata(ctx, in)
	case RunKindRestore:
		return a.fetchRestoreFileMetadata(ctx, in)
	default:
		return FetchFileMetadataOutput{}, fmt.Errorf("fetch file metadata activity: unknown kind %q", in.Kind)
	}
}

func (a *Activities) fetchBackupFileMetadata(ctx context.Context, in FetchFileMetadataInput) (FetchFileMetadataOutput, error) {
	run, err := a.BackupRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return FetchFileMetadataOutput{}, fmt.Errorf("fetch backup file metadata: find run: %w", err)
	}
	if run.EndAt != nil {
		return FetchFileMetadataOutput{Paths: sampleFilePaths(in.JobID)}, nil
	}

	if run.StartAt == nil {
		now := time.Now().UTC()
		run.StartAt = &now
		if err := a.BackupRunRepo.Update(run); err != nil {
			return FetchFileMetadataOutput{}, fmt.Errorf("fetch backup file metadata: set start_at: %w", err)
		}
		if err := a.markBackupRunStarted(in.JobID, in.MemberID, now); err != nil {
			return FetchFileMetadataOutput{}, err
		}
		recordHeartbeat(ctx, "fetching metadata")
		if err := sleepOrCancel(ctx, a.metadataFetchDelay()); err != nil {
			return FetchFileMetadataOutput{}, err
		}
	}

	a.Logger.Info("backup file metadata fetched", "run_id", in.RunID, "file_count", metadataFileCount)
	return FetchFileMetadataOutput{Paths: sampleFilePaths(in.JobID)}, nil
}

func (a *Activities) fetchRestoreFileMetadata(ctx context.Context, in FetchFileMetadataInput) (FetchFileMetadataOutput, error) {
	run, err := a.RestoreRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return FetchFileMetadataOutput{}, fmt.Errorf("fetch restore file metadata: find run: %w", err)
	}
	if run.EndAt != nil {
		return FetchFileMetadataOutput{Paths: sampleFilePaths(in.JobID)}, nil
	}

	if run.StartAt == nil {
		now := time.Now().UTC()
		run.StartAt = &now
		if err := a.RestoreRunRepo.Update(run); err != nil {
			return FetchFileMetadataOutput{}, fmt.Errorf("fetch restore file metadata: set start_at: %w", err)
		}
		if err := a.markRestoreRunStarted(in.JobID, in.MemberID, now); err != nil {
			return FetchFileMetadataOutput{}, err
		}
		recordHeartbeat(ctx, "fetching metadata")
		if err := sleepOrCancel(ctx, a.metadataFetchDelay()); err != nil {
			return FetchFileMetadataOutput{}, err
		}
	}

	a.Logger.Info("restore file metadata fetched", "run_id", in.RunID, "file_count", metadataFileCount)
	return FetchFileMetadataOutput{Paths: sampleFilePaths(in.JobID)}, nil
}

// TransferSingleFile simulates transferring one file.
func (a *Activities) TransferSingleFile(ctx context.Context, in TransferSingleFileInput) error {
	switch in.Kind {
	case RunKindBackup:
		return a.transferSingleBackupFile(ctx, in)
	case RunKindRestore:
		return a.transferSingleRestoreFile(ctx, in)
	default:
		return fmt.Errorf("transfer single file activity: unknown kind %q", in.Kind)
	}
}

func (a *Activities) transferSingleBackupFile(ctx context.Context, in TransferSingleFileInput) error {
	run, err := a.BackupRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return fmt.Errorf("transfer backup file: find run: %w", err)
	}
	if run.EndAt != nil {
		return nil
	}

	if err := a.logTransferContext(in.JobID, in.MemberID, in.Kind, in.FilePath); err != nil {
		return err
	}

	existing, err := a.BackupRunRepo.FindFileTransferByRunAndPath(in.RunID, in.FilePath)
	if err != nil {
		return fmt.Errorf("transfer backup file: find existing transfer: %w", err)
	}
	if existing != nil {
		return nil
	}

	recordHeartbeat(ctx, in.FilePath)
	start := time.Now().UTC()
	if err := sleepOrCancel(ctx, a.transferDelay()); err != nil {
		return err
	}
	end := time.Now().UTC()

	ft := &storage.BackupRunFileTransfer{
		ID:        uuid.NewString(),
		RunID:     in.RunID,
		FilePath:  in.FilePath,
		StartAt:   &start,
		EndAt:     &end,
		CreatedAt: start,
	}
	if err := a.BackupRunRepo.CreateFileTransfer(ft); err != nil {
		return fmt.Errorf("transfer backup file: create transfer: %w", err)
	}
	a.Logger.Info("backup file transferred", "run_id", in.RunID, "path", in.FilePath)
	return nil
}

func (a *Activities) transferSingleRestoreFile(ctx context.Context, in TransferSingleFileInput) error {
	run, err := a.RestoreRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return fmt.Errorf("transfer restore file: find run: %w", err)
	}
	if run.EndAt != nil {
		return nil
	}

	if err := a.logTransferContext(in.JobID, in.MemberID, in.Kind, in.FilePath); err != nil {
		return err
	}

	existing, err := a.RestoreRunRepo.FindFileTransferByRunAndPath(in.RunID, in.FilePath)
	if err != nil {
		return fmt.Errorf("transfer restore file: find existing transfer: %w", err)
	}
	if existing != nil {
		return nil
	}

	recordHeartbeat(ctx, in.FilePath)
	start := time.Now().UTC()
	if err := sleepOrCancel(ctx, a.transferDelay()); err != nil {
		return err
	}
	end := time.Now().UTC()

	ft := &storage.RestoreRunFileTransfer{
		ID:        uuid.NewString(),
		RunID:     in.RunID,
		FilePath:  in.FilePath,
		StartAt:   &start,
		EndAt:     &end,
		CreatedAt: start,
	}
	if err := a.RestoreRunRepo.CreateFileTransfer(ft); err != nil {
		return fmt.Errorf("transfer restore file: create transfer: %w", err)
	}
	a.Logger.Info("restore file transferred", "run_id", in.RunID, "path", in.FilePath)
	return nil
}

// FinalizeRun marks a run as complete.
func (a *Activities) FinalizeRun(ctx context.Context, in FinalizeRunInput) error {
	switch in.Kind {
	case RunKindBackup:
		run, err := a.BackupRunRepo.FindByID(in.RunID, in.MemberID)
		if err != nil {
			return fmt.Errorf("finalize backup run: find run: %w", err)
		}
		if run.EndAt != nil {
			return nil
		}
		return a.finalizeBackupRun(ctx, run, false)
	case RunKindRestore:
		run, err := a.RestoreRunRepo.FindByID(in.RunID, in.MemberID)
		if err != nil {
			return fmt.Errorf("finalize restore run: find run: %w", err)
		}
		if run.EndAt != nil {
			return nil
		}
		return a.finalizeRestoreRun(ctx, run, false)
	default:
		return fmt.Errorf("finalize run activity: unknown kind %q", in.Kind)
	}
}

type transferContext struct {
	organizationID string
	tenantID       string
	tenantSecret   string
	bucketName     string
	bucketType     string
	bucketConfig   string
}

func (a *Activities) loadTransferContext(jobID, memberID string, kind RunKind) (transferContext, error) {
	var orgID, bucketStoreID string

	switch kind {
	case RunKindBackup:
		job, err := a.BackupJobRepo.FindActiveByID(jobID, memberID)
		if err != nil {
			return transferContext{}, fmt.Errorf("load transfer context: backup job: %w", err)
		}
		orgID = job.OrganizationID
		bucketStoreID = job.BucketStoreID
	case RunKindRestore:
		job, err := a.RestoreJobRepo.FindActiveByID(jobID, memberID)
		if err != nil {
			return transferContext{}, fmt.Errorf("load transfer context: restore job: %w", err)
		}
		orgID = job.OrganizationID
		bucketStoreID = job.BucketStoreID
	default:
		return transferContext{}, fmt.Errorf("load transfer context: unknown kind %q", kind)
	}

	org, err := a.OrgRepo.FindActiveByID(orgID, memberID)
	if err != nil {
		return transferContext{}, fmt.Errorf("load transfer context: organization: %w", err)
	}

	tenantSecret, err := a.Encryptor.Decrypt(org.TenantSecretEncrypted)
	if err != nil {
		return transferContext{}, fmt.Errorf("load transfer context: decrypt tenant secret: %w", err)
	}

	bucket, err := a.BucketStoreRepo.FindActiveByID(bucketStoreID, memberID)
	if err != nil {
		return transferContext{}, fmt.Errorf("load transfer context: bucket store: %w", err)
	}

	bucketConfig, err := a.Encryptor.Decrypt(bucket.ConfigEncrypted)
	if err != nil {
		return transferContext{}, fmt.Errorf("load transfer context: decrypt bucket config: %w", err)
	}

	return transferContext{
		organizationID: org.ID,
		tenantID:       org.TenantID,
		tenantSecret:   tenantSecret,
		bucketName:     bucket.BucketName,
		bucketType:     bucket.BucketType,
		bucketConfig:   bucketConfig,
	}, nil
}

// logTransferContext logs org/tenant/bucket connection details for simulation.
// Production transfers should log IDs only, not decrypted secrets.
func (a *Activities) logTransferContext(jobID, memberID string, kind RunKind, filePath string) error {
	tc, err := a.loadTransferContext(jobID, memberID, kind)
	if err != nil {
		return err
	}

	a.Logger.Info("simulated file transfer context",
		"file_path", filePath,
		"organization_id", tc.organizationID,
		"tenant_id", tc.tenantID,
		"tenant_secret", tc.tenantSecret,
		"bucket_name", tc.bucketName,
		"bucket_type", tc.bucketType,
		"bucket_config", tc.bucketConfig,
	)
	return nil
}

func sampleFilePaths(jobID string) []string {
	paths := make([]string, metadataFileCount)
	for i := 1; i <= metadataFileCount; i++ {
		paths[i-1] = DummyFilePath(jobID, i)
	}
	return paths
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

func (a *Activities) markBackupRunStarted(jobID, memberID string, runAt time.Time) error {
	job, err := a.BackupJobRepo.FindActiveByID(jobID, memberID)
	if err != nil {
		return fmt.Errorf("mark backup run started: load job: %w", err)
	}
	backupjob.RecordRunStarted(job, runAt, time.Now().UTC())
	if err := a.BackupJobRepo.Update(job); err != nil {
		return fmt.Errorf("mark backup run started: update job: %w", err)
	}
	return nil
}

func (a *Activities) markRestoreRunStarted(jobID, memberID string, runAt time.Time) error {
	job, err := a.RestoreJobRepo.FindActiveByID(jobID, memberID)
	if err != nil {
		return fmt.Errorf("mark restore run started: load job: %w", err)
	}
	restorejob.RecordRunStarted(job, runAt, time.Now().UTC())
	if err := a.RestoreJobRepo.Update(job); err != nil {
		return fmt.Errorf("mark restore run started: update job: %w", err)
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
	w.RegisterActivityWithOptions(a.FetchFileMetadata, activity.RegisterOptions{Name: fetchFileMetadataActivityName})
	w.RegisterActivityWithOptions(a.TransferSingleFile, activity.RegisterOptions{Name: transferSingleFileActivityName})
	w.RegisterActivityWithOptions(a.FinalizeRun, activity.RegisterOptions{Name: finalizeRunActivityName})
}

// ErrRunAlreadyComplete is returned when starting a workflow for a finished run.
var ErrRunAlreadyComplete = errors.New("run already complete")
