package temporal

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

func TestTransferFilesBackupIdempotent(t *testing.T) {
	db, err := storage.Open("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)

	memberID := "member-1"
	now := time.Now().UTC()
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.BackupJob{
		ID:             jobID,
		MemberID:       memberID,
		Active:         true,
		OrganizationID: "org",
		BucketStoreID:  "bucket",
		SharePointSite: "https://example.com",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	runID := uuid.NewString()
	if err := runRepo.Create(&storage.BackupRun{
		ID:        runID,
		JobID:     jobID,
		MemberID:  memberID,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	acts := &Activities{
		BackupRunRepo: runRepo,
		BackupJobRepo: jobRepo,
		Logger:        logger,
		TransferDelay: time.Millisecond,
	}

	in := TransferFilesInput{
		RunID:    runID,
		JobID:    jobID,
		MemberID: memberID,
		Kind:     RunKindBackup,
	}
	if err := acts.TransferFiles(context.Background(), in); err != nil {
		t.Fatalf("first transfer: %v", err)
	}

	run, err := runRepo.FindByID(runID, memberID)
	if err != nil {
		t.Fatalf("find run: %v", err)
	}
	if run.EndAt == nil {
		t.Fatal("expected run to be complete")
	}

	_, total, err := runRepo.ListFileTransfers(runID, 0, 100)
	if err != nil {
		t.Fatalf("list transfers: %v", err)
	}
	if total != dummyFileCount {
		t.Fatalf("file count = %d, want %d", total, dummyFileCount)
	}

	// Second invocation should be a no-op for completed run.
	if err := acts.TransferFiles(context.Background(), in); err != nil {
		t.Fatalf("second transfer: %v", err)
	}
}
