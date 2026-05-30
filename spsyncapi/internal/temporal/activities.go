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
	"spsyncapi/pkg/azureblob"
	"spsyncapi/pkg/graphapi"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

const (
	syncFileMetadataPageActivityName = "SyncFileMetadataPage"
	listPendingFileLogsActivityName  = "ListPendingFileLogs"
	transferSingleFileActivityName   = "TransferSingleFile"
	finalizeRunActivityName          = "FinalizeRun"
)

// GraphServiceBuilder constructs a Graph client for a job. Used in tests.
type GraphServiceBuilder func(jc jobContext) graphapi.Service

// AzureServiceBuilder constructs an Azure Blob client for a job. Used in tests.
type AzureServiceBuilder func(jc jobContext) azureblob.Service

// Activities holds dependencies for Temporal activity implementations.
type Activities struct {
	BackupRunRepo       *storage.BackupRunRepository
	RestoreRunRepo      *storage.RestoreRunRepository
	BackupJobRepo       *storage.BackupJobRepository
	RestoreJobRepo      *storage.RestoreJobRepository
	OrgRepo             *storage.OrganizationRepository
	BucketStoreRepo     *storage.BucketStoreRepository
	Encryptor           *crypto.SecretEncryptor
	Logger              *slog.Logger
	GraphServiceBuilder GraphServiceBuilder
	AzureServiceBuilder AzureServiceBuilder
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

// TransferSingleFile transfers one file between SharePoint and Azure Blob.
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

	ft, err := a.BackupRunRepo.FindFileTransferByRunAndPath(in.RunID, in.File.Path)
	if err != nil {
		return fmt.Errorf("transfer backup file: find file log: %w", err)
	}
	if ft == nil {
		return fmt.Errorf("transfer backup file: file log not found for path %q", in.File.Path)
	}
	if ft.Status == storage.FileLogStatusSuccess {
		return nil
	}

	jc, err := a.loadJobContext(in.JobID, in.MemberID, RunKindBackup)
	if err != nil {
		return fmt.Errorf("transfer backup file: %w", err)
	}

	graph := a.buildGraphClient(jc)
	azure := a.buildAzureClient(jc)

	start := time.Now().UTC()
	ft.Status = storage.FileLogStatusPending
	ft.StartAt = &start
	ft.ErrorMessage = ""
	if err := a.BackupRunRepo.UpdateFileTransfer(ft); err != nil {
		return fmt.Errorf("transfer backup file: mark started: %w", err)
	}

	recordHeartbeat(ctx, in.File.Path)

	if _, err := graph.GetAccessToken(); err != nil {
		return a.failBackupFileTransfer(ft, fmt.Errorf("transfer backup file: graph token: %w", err))
	}

	resp, err := graph.GetDriveItemDownload(in.File.DriveID, in.File.DriveItemID)
	if err != nil {
		return a.failBackupFileTransfer(ft, fmt.Errorf("transfer backup file: download from sharepoint: %w", err))
	}
	defer resp.Body.Close()

	if err := azure.UploadBlob(jc.containerName, in.File.Path, resp); err != nil {
		return a.failBackupFileTransfer(ft, fmt.Errorf("transfer backup file: upload to azure: %w", err))
	}

	end := time.Now().UTC()
	ft.Status = storage.FileLogStatusSuccess
	ft.EndAt = &end
	if err := a.BackupRunRepo.UpdateFileTransfer(ft); err != nil {
		return fmt.Errorf("transfer backup file: update transfer: %w", err)
	}

	a.Logger.Info("backup file transferred",
		"run_id", in.RunID,
		"path", in.File.Path,
		"size", in.File.Size,
	)
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

	ft, err := a.RestoreRunRepo.FindFileTransferByRunAndPath(in.RunID, in.File.Path)
	if err != nil {
		return fmt.Errorf("transfer restore file: find file log: %w", err)
	}
	if ft == nil {
		return fmt.Errorf("transfer restore file: file log not found for path %q", in.File.Path)
	}
	if ft.Status == storage.FileLogStatusSuccess {
		return nil
	}

	jc, err := a.loadJobContext(in.JobID, in.MemberID, RunKindRestore)
	if err != nil {
		return fmt.Errorf("transfer restore file: %w", err)
	}

	graph := a.buildGraphClient(jc)
	azure := a.buildAzureClient(jc)

	start := time.Now().UTC()
	ft.Status = storage.FileLogStatusPending
	ft.StartAt = &start
	ft.ErrorMessage = ""
	if err := a.RestoreRunRepo.UpdateFileTransfer(ft); err != nil {
		return fmt.Errorf("transfer restore file: mark started: %w", err)
	}

	recordHeartbeat(ctx, in.File.Path)

	documentLibrary, libraryPath := splitRestorePath(in.File.Path)

	if _, err := graph.GetAccessToken(); err != nil {
		return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: graph token: %w", err))
	}

	siteID, err := graph.GetSiteId(jc.sharePointSite)
	if err != nil {
		return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: site id: %w", err))
	}

	driveID, err := graph.GetDriveId(siteID, documentLibrary)
	if errors.Is(err, graphapi.ErrDriveNotFound) {
		if _, createErr := graph.CreateDocumentLibrary(siteID, documentLibrary); createErr != nil {
			return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: create document library: %w", createErr))
		}
		driveID, err = graph.GetDriveId(siteID, documentLibrary)
	}
	if err != nil {
		return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: drive id: %w", err))
	}

	download, err := azure.DownloadBlobToStream(jc.containerName, in.File.Path)
	if err != nil {
		return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: download from azure: %w", err))
	}
	defer download.Body.Close()

	var contentLength int64
	if download.ContentLength != nil {
		contentLength = *download.ContentLength
	}

	if contentLength > 0 && contentLength < chunkUploadThreshold {
		if err := graph.UploadDriveItemWhole(driveID, libraryPath, download.Body); err != nil {
			return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: upload whole: %w", err))
		}
	} else {
		if contentLength == 0 {
			contentLength = in.File.Size
		}
		if err := graph.UploadDriveItemChunked(driveID, libraryPath, contentLength, download.Body); err != nil {
			return a.failRestoreFileTransfer(ft, fmt.Errorf("transfer restore file: upload chunked: %w", err))
		}
	}

	end := time.Now().UTC()
	ft.Status = storage.FileLogStatusSuccess
	ft.EndAt = &end
	if err := a.RestoreRunRepo.UpdateFileTransfer(ft); err != nil {
		return fmt.Errorf("transfer restore file: update transfer: %w", err)
	}

	a.Logger.Info("restore file transferred",
		"run_id", in.RunID,
		"path", in.File.Path,
		"size", contentLength,
	)
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
		return a.finalizeBackupRun(run)
	case RunKindRestore:
		run, err := a.RestoreRunRepo.FindByID(in.RunID, in.MemberID)
		if err != nil {
			return fmt.Errorf("finalize restore run: find run: %w", err)
		}
		if run.EndAt != nil {
			return nil
		}
		return a.finalizeRestoreRun(run)
	default:
		return fmt.Errorf("finalize run activity: unknown kind %q", in.Kind)
	}
}

func (a *Activities) finalizeBackupRun(run *storage.BackupRun) error {
	now := time.Now().UTC()
	run.EndAt = &now
	if err := a.BackupRunRepo.Update(run); err != nil {
		return fmt.Errorf("finalize backup run: %w", err)
	}
	return nil
}

func (a *Activities) finalizeRestoreRun(run *storage.RestoreRun) error {
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

// Register wires activity methods on the worker.
func (a *Activities) Register(w interface {
	RegisterActivityWithOptions(fn interface{}, opts activity.RegisterOptions)
}) {
	w.RegisterActivityWithOptions(a.CreateBackupRun, activity.RegisterOptions{Name: "CreateBackupRun"})
	w.RegisterActivityWithOptions(a.CreateRestoreRun, activity.RegisterOptions{Name: "CreateRestoreRun"})
	w.RegisterActivityWithOptions(a.SyncFileMetadataPage, activity.RegisterOptions{Name: syncFileMetadataPageActivityName})
	w.RegisterActivityWithOptions(a.ListPendingFileLogs, activity.RegisterOptions{Name: listPendingFileLogsActivityName})
	w.RegisterActivityWithOptions(a.TransferSingleFile, activity.RegisterOptions{Name: transferSingleFileActivityName})
	w.RegisterActivityWithOptions(a.FinalizeRun, activity.RegisterOptions{Name: finalizeRunActivityName})
}

// ErrRunAlreadyComplete is returned when starting a workflow for a finished run.
var ErrRunAlreadyComplete = errors.New("run already complete")
