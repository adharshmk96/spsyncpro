package backuprun_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/backuprun"
	"spsyncapi/internal/storage"

	"github.com/google/uuid"
)

const testMemberA = "member-backup-run-a"

func newTestBackupRunService(t *testing.T) (*backuprun.Service, *storage.BackupJobRepository, *storage.BackupRunRepository) {
	t.Helper()

	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)

	svc, err := backuprun.NewService(backuprun.ServiceConfig{
		RunRepo: runRepo,
		JobRepo: jobRepo,
		Logger:  logger,
	})
	if err != nil {
		t.Fatalf("backup run service: %v", err)
	}
	return svc, jobRepo, runRepo
}

func seedBackupJob(t *testing.T, jobRepo *storage.BackupJobRepository) string {
	t.Helper()
	now := time.Now().UTC()
	jobID := uuid.NewString()
	job := &storage.BackupJob{
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
		t.Fatalf("create backup job: %v", err)
	}
	return jobID
}

func seedBackupRun(t *testing.T, runRepo *storage.BackupRunRepository, jobID string) string {
	t.Helper()
	now := time.Now().UTC()
	start := now.Add(-1 * time.Hour)
	end := now
	runID := uuid.NewString()
	run := &storage.BackupRun{
		ID:        runID,
		JobID:     jobID,
		MemberID:  testMemberA,
		StartAt:   &start,
		EndAt:     &end,
		CreatedAt: now,
	}
	if err := runRepo.Create(run); err != nil {
		t.Fatalf("create backup run: %v", err)
	}
	return runID
}

func TestListAndGetBackupRuns(t *testing.T) {
	svc, jobRepo, runRepo := newTestBackupRunService(t)
	jobID := seedBackupJob(t, jobRepo)
	runID := seedBackupRun(t, runRepo, jobID)

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		start := now.Add(time.Duration(i) * time.Minute)
		ft := &storage.BackupRunFileTransfer{
			ID:        uuid.NewString(),
			RunID:     runID,
			FilePath:  "/docs/file-" + string(rune('a'+i)) + ".txt",
			StartAt:   &start,
			EndAt:     &start,
			CreatedAt: now,
		}
		if err := runRepo.CreateFileTransfer(ft); err != nil {
			t.Fatalf("create file transfer: %v", err)
		}
	}

	list, err := svc.List(testMemberA, nil, 1, 20)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if list.Total != 1 || len(list.Runs) != 1 {
		t.Fatalf("list: total=%d len=%d", list.Total, len(list.Runs))
	}
	if list.Runs[0].JobID != jobID {
		t.Fatalf("job id: got %q", list.Runs[0].JobID)
	}

	filter := jobID
	filtered, err := svc.List(testMemberA, &filter, 1, 20)
	if err != nil {
		t.Fatalf("list with filter: %v", err)
	}
	if filtered.Total != 1 {
		t.Fatalf("filtered total: got %d", filtered.Total)
	}

	unknownJob := "unknown-job"
	_, err = svc.List(testMemberA, &unknownJob, 1, 20)
	if err == nil {
		t.Fatal("expected job not found for filter")
	}

	got, err := svc.Get(testMemberA, runID, 1, 2)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Run.ID != runID {
		t.Fatalf("run id: got %q", got.Run.ID)
	}
	if got.FilesTotal != 3 {
		t.Fatalf("files total: got %d", got.FilesTotal)
	}
	if len(got.FileTransfers) != 2 {
		t.Fatalf("file transfers page: got %d", len(got.FileTransfers))
	}

	page2, err := svc.Get(testMemberA, runID, 2, 2)
	if err != nil {
		t.Fatalf("get page 2: %v", err)
	}
	if len(page2.FileTransfers) != 1 {
		t.Fatalf("page 2 files: got %d", len(page2.FileTransfers))
	}
}

func TestGetBackupRunNotFound(t *testing.T) {
	svc, _, _ := newTestBackupRunService(t)
	_, err := svc.Get(testMemberA, "missing-run", 1, 20)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestGetBackupRunInactiveJobHidden(t *testing.T) {
	svc, jobRepo, runRepo := newTestBackupRunService(t)
	jobID := seedBackupJob(t, jobRepo)
	runID := seedBackupRun(t, runRepo, jobID)

	if err := jobRepo.MarkInactive(jobID, testMemberA); err != nil {
		t.Fatalf("mark inactive: %v", err)
	}

	_, err := svc.Get(testMemberA, runID, 1, 20)
	if err == nil {
		t.Fatal("expected not found when job inactive")
	}
}
