package restorerun_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/restorerun"
	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

const testMemberA = "member-restore-run-a"

func newTestRestoreRunService(t *testing.T) (*restorerun.Service, *storage.RestoreJobRepository, *storage.RestoreRunRepository) {
	t.Helper()

	db, err := storage.Open("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewRestoreJobRepository(db)
	runRepo := storage.NewRestoreRunRepository(db)

	svc, err := restorerun.NewService(restorerun.ServiceConfig{
		RunRepo: runRepo,
		JobRepo: jobRepo,
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("restore run service: %v", err)
	}
	return svc, jobRepo, runRepo
}

func seedRestoreJob(t *testing.T, jobRepo *storage.RestoreJobRepository) string {
	t.Helper()
	now := time.Now().UTC()
	jobID := uuid.NewString()
	job := &storage.RestoreJob{
		ID:             jobID,
		MemberID:       testMemberA,
		Active:         true,
		OrganizationID: "org-1",
		BucketStoreID:  "bucket-1",
		SharePointSite: "https://example.sharepoint.com/sites/demo",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := jobRepo.Create(job); err != nil {
		t.Fatalf("create restore job: %v", err)
	}
	return jobID
}

func seedRestoreRun(t *testing.T, runRepo *storage.RestoreRunRepository, jobID string) string {
	t.Helper()
	now := time.Now().UTC()
	start := now.Add(-30 * time.Minute)
	end := now
	runID := uuid.NewString()
	run := &storage.RestoreRun{
		ID:        runID,
		JobID:     jobID,
		MemberID:  testMemberA,
		StartAt:   &start,
		EndAt:     &end,
		CreatedAt: now,
	}
	if err := runRepo.Create(run); err != nil {
		t.Fatalf("create restore run: %v", err)
	}
	return runID
}

func TestListAndGetRestoreRuns(t *testing.T) {
	svc, jobRepo, runRepo := newTestRestoreRunService(t)
	jobID := seedRestoreJob(t, jobRepo)
	runID := seedRestoreRun(t, runRepo, jobID)

	now := time.Now().UTC()
	ft := &storage.RestoreRunFileTransfer{
		ID:        uuid.NewString(),
		RunID:     runID,
		FilePath:  "/restored/doc.pdf",
		StartAt:   &now,
		EndAt:     &now,
		CreatedAt: now,
	}
	if err := runRepo.CreateFileTransfer(ft); err != nil {
		t.Fatalf("create file transfer: %v", err)
	}

	list, err := svc.List(testMemberA, nil, 1, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if list.Total != 1 || len(list.Runs) != 1 {
		t.Fatalf("list: total=%d len=%d", list.Total, len(list.Runs))
	}

	got, err := svc.Get(testMemberA, runID, 1, 20)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Run.JobID != jobID {
		t.Fatalf("job id: got %q", got.Run.JobID)
	}
	if got.FilesTotal != 1 || len(got.FileTransfers) != 1 {
		t.Fatalf("files: total=%d len=%d", got.FilesTotal, len(got.FileTransfers))
	}
	if got.FileTransfers[0].FilePath != "/restored/doc.pdf" {
		t.Fatalf("file path: got %q", got.FileTransfers[0].FilePath)
	}
}

func TestGetRestoreRunNotFound(t *testing.T) {
	svc, _, _ := newTestRestoreRunService(t)
	_, err := svc.Get(testMemberA, "missing-run", 1, 20)
	if err == nil {
		t.Fatal("expected not found")
	}
}
