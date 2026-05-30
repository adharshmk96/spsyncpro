package temporal

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/storage"
	"spsyncapi/pkg/azureblob"
	"spsyncapi/pkg/graphapi"

	"github.com/google/uuid"
)

func TestBackupMetadataSyncResumesFromCheckpoint(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	memberID := "member-1"
	orgID, bucketID, enc := setupAzureTransferFixtures(t, db, memberID)

	graphMock := &mockGraphService{
		siteID: "site-1",
		drives: []graphapi.Drive{{ID: "drive-1", Name: "Documents"}},
		itemsByDrive: map[string][]graphapi.DriveItem{
			"drive-1": {{ID: "item-1", Name: "file.txt", FilePath: "/file.txt", Size: 12}},
		},
	}
	azureMock := &mockAzureService{}

	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)
	now := time.Now().UTC()
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.BackupJob{
		ID:             jobID,
		MemberID:       memberID,
		Active:         true,
		OrganizationID: orgID,
		BucketStoreID:  bucketID,
		SharePointSite: "https://tenant.sharepoint.com/sites/demo",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	runID := uuid.NewString()
	if err := runRepo.Create(&storage.BackupRun{
		ID:                 runID,
		JobID:              jobID,
		MemberID:           memberID,
		MetadataSyncStatus: storage.MetadataSyncInProgress,
		MetadataSyncCheckpoint: encodeBackupCheckpoint(backupMetadataCheckpoint{
			SiteID: "site-1",
			Phase:  checkpointPhaseItems,
			CurrentDrive: &driveRef{ID: "drive-1", Name: "Documents"},
		}),
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	acts := &Activities{
		BackupRunRepo:       runRepo,
		BackupJobRepo:       jobRepo,
		OrgRepo:             storage.NewOrganizationRepository(db),
		BucketStoreRepo:     storage.NewBucketStoreRepository(db),
		Encryptor:           enc,
		Logger:              logger,
		GraphServiceBuilder: func(jobContext) graphapi.Service { return graphMock },
		AzureServiceBuilder: func(jobContext) azureblob.Service { return azureMock },
	}

	syncIn := SyncFileMetadataPageInput{
		RunID: runID, JobID: jobID, MemberID: memberID, Kind: RunKindBackup,
	}
	out, err := acts.SyncFileMetadataPage(context.Background(), syncIn)
	if err != nil {
		t.Fatalf("sync page: %v", err)
	}
	if !out.Complete {
		t.Fatal("expected metadata sync to complete in one items page")
	}

	logs, err := runRepo.ListFileLogsByStatus(runID, []string{storage.FileLogStatusPending}, 0, 10)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("file log count = %d, want 1", len(logs))
	}
	if graphMock.drivesPageDone {
		t.Fatal("expected drives listing to be skipped when resuming in items phase")
	}
}
