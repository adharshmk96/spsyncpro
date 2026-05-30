package temporal

import (
	"context"
	"fmt"
	"time"

	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

const listPendingFileLogsBatchSize = 100

// SyncFileMetadataPage syncs one Graph or Azure page of file metadata into the run file log.
func (a *Activities) SyncFileMetadataPage(ctx context.Context, in SyncFileMetadataPageInput) (SyncFileMetadataPageOutput, error) {
	switch in.Kind {
	case RunKindBackup:
		return a.syncBackupFileMetadataPage(ctx, in)
	case RunKindRestore:
		return a.syncRestoreFileMetadataPage(ctx, in)
	default:
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync file metadata page: unknown kind %q", in.Kind)
	}
}

func (a *Activities) syncBackupFileMetadataPage(ctx context.Context, in SyncFileMetadataPageInput) (SyncFileMetadataPageOutput, error) {
	run, err := a.BackupRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: find run: %w", err)
	}
	if run.EndAt != nil {
		return SyncFileMetadataPageOutput{Complete: true}, nil
	}
	if run.MetadataSyncStatus == storage.MetadataSyncComplete {
		return SyncFileMetadataPageOutput{Complete: true}, nil
	}

	if err := a.ensureBackupRunMetadataStarted(run, in); err != nil {
		return SyncFileMetadataPageOutput{}, err
	}

	jc, err := a.loadJobContext(in.JobID, in.MemberID, RunKindBackup)
	if err != nil {
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: %w", err)
	}

	graph := a.buildGraphClient(jc)
	azure := a.buildAzureClient(jc)

	if _, err := graph.GetAccessToken(); err != nil {
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: graph token: %w", err)
	}

	cp := parseBackupCheckpoint(run.MetadataSyncCheckpoint)
	if cp.SiteID == "" {
		siteID, err := graph.GetSiteId(jc.sharePointSite)
		if err != nil {
			return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: site id: %w", err)
		}
		cp.SiteID = siteID
		if err := azure.CreateContainer(jc.containerName); err != nil {
			return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: create container: %w", err)
		}
	}

	synced := 0

	if cp.Phase == checkpointPhaseDrives {
		drives, nextURL, err := graph.ListDrivesPage(cp.SiteID, cp.DrivesPageURL)
		if err != nil {
			a.failBackupMetadataSync(in.RunID, cp, err)
			return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: list drives: %w", err)
		}
		for _, drive := range drives {
			if !driveAllowed(drive.Name, jc.documentLibs) || driveCompleted(&cp, drive.ID) {
				continue
			}
			cp.PendingDrives = append(cp.PendingDrives, driveRef{ID: drive.ID, Name: drive.Name})
		}
		recordHeartbeat(ctx, map[string]string{"phase": "drives", "next": nextURL})
		if nextURL != "" {
			cp.DrivesPageURL = nextURL
			if err := a.saveBackupMetadataCheckpoint(in.RunID, cp); err != nil {
				return SyncFileMetadataPageOutput{}, err
			}
			return SyncFileMetadataPageOutput{FilesSyncedThisPage: synced}, nil
		}
		cp.Phase = checkpointPhaseItems
		cp.DrivesPageURL = ""
	}

	if cp.CurrentDrive == nil && len(cp.PendingDrives) > 0 {
		cp.CurrentDrive = &cp.PendingDrives[0]
		cp.PendingDrives = cp.PendingDrives[1:]
		cp.Crawl = nil
	}

	if cp.CurrentDrive == nil {
		return a.completeBackupMetadataSync(in.RunID, cp, synced)
	}

	recordHeartbeat(ctx, map[string]string{
		"phase":      "items",
		"drive":      cp.CurrentDrive.Name,
		"page_url":   cp.CrawlPageURL(),
	})

	items, nextCrawl, crawlDone, err := graph.ListDriveItemsPage(cp.CurrentDrive.ID, cp.Crawl)
	if err != nil {
		a.failBackupMetadataSync(in.RunID, cp, err)
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: list items: %w", err)
	}
	cp.Crawl = nextCrawl

	now := time.Now().UTC()
	for _, item := range items {
		if shouldSkipFile(item, jc.backupJob) {
			continue
		}
		ft := &storage.BackupRunFileTransfer{
			ID:               uuid.NewString(),
			RunID:            in.RunID,
			FilePath:         backupBlobPath(cp.CurrentDrive.Name, item.FilePath),
			Status:           storage.FileLogStatusPending,
			DriveID:          cp.CurrentDrive.ID,
			DriveName:        cp.CurrentDrive.Name,
			DriveItemID:      item.ID,
			Size:             item.Size,
			MetadataSyncedAt: &now,
			CreatedAt:        now,
		}
		if err := a.BackupRunRepo.UpsertFileLog(ft); err != nil {
			return SyncFileMetadataPageOutput{}, fmt.Errorf("sync backup metadata page: upsert file: %w", err)
		}
		synced++
	}

	if crawlDone {
		cp.CompletedDriveIDs = append(cp.CompletedDriveIDs, cp.CurrentDrive.ID)
		cp.CurrentDrive = nil
		cp.Crawl = nil
	}

	if cp.CurrentDrive == nil && len(cp.PendingDrives) == 0 {
		return a.completeBackupMetadataSync(in.RunID, cp, synced)
	}

	if err := a.saveBackupMetadataCheckpoint(in.RunID, cp); err != nil {
		return SyncFileMetadataPageOutput{}, err
	}
	return SyncFileMetadataPageOutput{FilesSyncedThisPage: synced}, nil
}

func (a *Activities) syncRestoreFileMetadataPage(ctx context.Context, in SyncFileMetadataPageInput) (SyncFileMetadataPageOutput, error) {
	run, err := a.RestoreRunRepo.FindByID(in.RunID, in.MemberID)
	if err != nil {
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync restore metadata page: find run: %w", err)
	}
	if run.EndAt != nil {
		return SyncFileMetadataPageOutput{Complete: true}, nil
	}
	if run.MetadataSyncStatus == storage.MetadataSyncComplete {
		return SyncFileMetadataPageOutput{Complete: true}, nil
	}

	if err := a.ensureRestoreRunMetadataStarted(run, in); err != nil {
		return SyncFileMetadataPageOutput{}, err
	}

	jc, err := a.loadJobContext(in.JobID, in.MemberID, RunKindRestore)
	if err != nil {
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync restore metadata page: %w", err)
	}

	azure := a.buildAzureClient(jc)
	cp := parseRestoreCheckpoint(run.MetadataSyncCheckpoint)

	blobs, nextMarker, err := azure.ListBlobsPage(jc.containerName, cp.Marker, nil)
	if err != nil {
		a.failRestoreMetadataSync(in.RunID, cp, err)
		return SyncFileMetadataPageOutput{}, fmt.Errorf("sync restore metadata page: list blobs: %w", err)
	}

	recordHeartbeat(ctx, map[string]string{"marker": nextMarker})

	now := time.Now().UTC()
	synced := 0
	for _, blob := range blobs {
		ft := &storage.RestoreRunFileTransfer{
			ID:               uuid.NewString(),
			RunID:            in.RunID,
			FilePath:         blob.FullPath,
			Status:           storage.FileLogStatusPending,
			Size:             blob.Size,
			MetadataSyncedAt: &now,
			CreatedAt:        now,
		}
		if err := a.RestoreRunRepo.UpsertFileLog(ft); err != nil {
			return SyncFileMetadataPageOutput{}, fmt.Errorf("sync restore metadata page: upsert file: %w", err)
		}
		synced++
	}

	if nextMarker == "" {
		if err := a.RestoreRunRepo.UpdateMetadataSyncState(in.RunID, storage.MetadataSyncComplete, "", ""); err != nil {
			return SyncFileMetadataPageOutput{}, err
		}
		a.Logger.Info("restore file metadata sync complete", "run_id", in.RunID)
		return SyncFileMetadataPageOutput{Complete: true, FilesSyncedThisPage: synced}, nil
	}

	cp.Marker = nextMarker
	if err := a.RestoreRunRepo.UpdateMetadataSyncState(in.RunID, storage.MetadataSyncInProgress, encodeRestoreCheckpoint(cp), ""); err != nil {
		return SyncFileMetadataPageOutput{}, err
	}
	return SyncFileMetadataPageOutput{FilesSyncedThisPage: synced}, nil
}

func (a *Activities) ensureBackupRunMetadataStarted(run *storage.BackupRun, in SyncFileMetadataPageInput) error {
	if run.StartAt != nil && run.MetadataSyncStatus != storage.MetadataSyncNotStarted && run.MetadataSyncStatus != "" {
		return nil
	}
	now := time.Now().UTC()
	if run.StartAt == nil {
		run.StartAt = &now
		if err := a.BackupRunRepo.Update(run); err != nil {
			return fmt.Errorf("sync backup metadata page: set start_at: %w", err)
		}
		if err := a.markBackupRunStarted(in.JobID, in.MemberID, now); err != nil {
			return err
		}
	}
	run.MetadataSyncStatus = storage.MetadataSyncInProgress
	return a.BackupRunRepo.UpdateMetadataSyncState(in.RunID, storage.MetadataSyncInProgress, run.MetadataSyncCheckpoint, "")
}

func (a *Activities) ensureRestoreRunMetadataStarted(run *storage.RestoreRun, in SyncFileMetadataPageInput) error {
	if run.StartAt != nil && run.MetadataSyncStatus != storage.MetadataSyncNotStarted && run.MetadataSyncStatus != "" {
		return nil
	}
	now := time.Now().UTC()
	if run.StartAt == nil {
		run.StartAt = &now
		if err := a.RestoreRunRepo.Update(run); err != nil {
			return fmt.Errorf("sync restore metadata page: set start_at: %w", err)
		}
		if err := a.markRestoreRunStarted(in.JobID, in.MemberID, now); err != nil {
			return err
		}
	}
	return a.RestoreRunRepo.UpdateMetadataSyncState(in.RunID, storage.MetadataSyncInProgress, run.MetadataSyncCheckpoint, "")
}

func (a *Activities) saveBackupMetadataCheckpoint(runID string, cp backupMetadataCheckpoint) error {
	return a.BackupRunRepo.UpdateMetadataSyncState(runID, storage.MetadataSyncInProgress, encodeBackupCheckpoint(cp), "")
}

func (a *Activities) completeBackupMetadataSync(runID string, cp backupMetadataCheckpoint, synced int) (SyncFileMetadataPageOutput, error) {
	if err := a.BackupRunRepo.UpdateMetadataSyncState(runID, storage.MetadataSyncComplete, "", ""); err != nil {
		return SyncFileMetadataPageOutput{}, err
	}
	a.Logger.Info("backup file metadata sync complete", "run_id", runID)
	return SyncFileMetadataPageOutput{Complete: true, FilesSyncedThisPage: synced}, nil
}

func (a *Activities) failBackupMetadataSync(runID string, cp backupMetadataCheckpoint, err error) {
	_ = a.BackupRunRepo.UpdateMetadataSyncState(runID, storage.MetadataSyncFailed, encodeBackupCheckpoint(cp), err.Error())
}

func (a *Activities) failRestoreMetadataSync(runID string, cp restoreMetadataCheckpoint, err error) {
	_ = a.RestoreRunRepo.UpdateMetadataSyncState(runID, storage.MetadataSyncFailed, encodeRestoreCheckpoint(cp), err.Error())
}

func (cp *backupMetadataCheckpoint) CrawlPageURL() string {
	if cp.Crawl == nil {
		return ""
	}
	return cp.Crawl.PageURL
}

// ListPendingFileLogs returns a batch of files awaiting transfer for a run.
func (a *Activities) ListPendingFileLogs(ctx context.Context, in ListPendingFileLogsInput) (ListPendingFileLogsOutput, error) {
	statuses := []string{storage.FileLogStatusPending, storage.FileLogStatusFailure}
	switch in.Kind {
	case RunKindBackup:
		logs, err := a.BackupRunRepo.ListFileLogsByStatus(in.RunID, statuses, 0, in.Limit)
		if err != nil {
			return ListPendingFileLogsOutput{}, fmt.Errorf("list pending backup file logs: %w", err)
		}
		return ListPendingFileLogsOutput{Files: backupFileLogsToDescriptors(logs)}, nil
	case RunKindRestore:
		logs, err := a.RestoreRunRepo.ListFileLogsByStatus(in.RunID, statuses, 0, in.Limit)
		if err != nil {
			return ListPendingFileLogsOutput{}, fmt.Errorf("list pending restore file logs: %w", err)
		}
		return ListPendingFileLogsOutput{Files: restoreFileLogsToDescriptors(logs)}, nil
	default:
		return ListPendingFileLogsOutput{}, fmt.Errorf("list pending file logs: unknown kind %q", in.Kind)
	}
}

func backupFileLogsToDescriptors(logs []storage.BackupRunFileTransfer) []FileDescriptor {
	out := make([]FileDescriptor, 0, len(logs))
	for i := range logs {
		out = append(out, FileDescriptor{
			Path:        logs[i].FilePath,
			DriveID:     logs[i].DriveID,
			DriveItemID: logs[i].DriveItemID,
			Size:        logs[i].Size,
		})
	}
	return out
}

func restoreFileLogsToDescriptors(logs []storage.RestoreRunFileTransfer) []FileDescriptor {
	out := make([]FileDescriptor, 0, len(logs))
	for i := range logs {
		out = append(out, FileDescriptor{
			Path: logs[i].FilePath,
			Size: logs[i].Size,
		})
	}
	return out
}
