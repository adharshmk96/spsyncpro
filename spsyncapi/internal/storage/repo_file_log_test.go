package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBackupRunFileLogUpsertAndListByStatus(t *testing.T) {
	db, err := OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	repo := NewBackupRunRepository(db)
	now := time.Now().UTC()
	runID := uuid.NewString()

	if err := repo.Create(&BackupRun{
		ID:        runID,
		JobID:     uuid.NewString(),
		MemberID:  "member-1",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	ft := &BackupRunFileTransfer{
		ID:        uuid.NewString(),
		RunID:     runID,
		FilePath:  "Documents/a.txt",
		Status:    FileLogStatusPending,
		DriveName: "Documents",
		Size:      100,
		CreatedAt: now,
	}
	if err := repo.UpsertFileLog(ft); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := repo.UpsertFileLog(ft); err != nil {
		t.Fatalf("duplicate upsert: %v", err)
	}

	pending, err := repo.ListFileLogsByStatus(runID, []string{FileLogStatusPending}, 0, 10)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending count = %d, want 1", len(pending))
	}

	ft.Status = FileLogStatusSuccess
	end := now.Add(time.Minute)
	ft.EndAt = &end
	if err := repo.UpdateFileTransfer(ft); err != nil {
		t.Fatalf("update: %v", err)
	}

	pending, err = repo.ListFileLogsByStatus(runID, []string{FileLogStatusPending, FileLogStatusFailure}, 0, 10)
	if err != nil {
		t.Fatalf("list pending after success: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after success = %d, want 0", len(pending))
	}

	if err := repo.UpdateMetadataSyncState(runID, MetadataSyncComplete, `{"site_id":"s1"}`, ""); err != nil {
		t.Fatalf("update metadata sync: %v", err)
	}
	run, err := repo.FindByID(runID, "member-1")
	if err != nil {
		t.Fatalf("find run: %v", err)
	}
	if run.MetadataSyncStatus != MetadataSyncComplete {
		t.Fatalf("metadata status = %q, want complete", run.MetadataSyncStatus)
	}
}
