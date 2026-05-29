package restorejob_test

import (
	"testing"
	"time"

	"spsyncapi/internal/restorejob"
	"spsyncapi/internal/storage"
)

func TestApplyScheduleOnSaveImmediate(t *testing.T) {
	now := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	job := &storage.RestoreJob{}
	restorejob.ApplyScheduleOnSave(job, nil, now)

	if job.StartAt == nil || !job.StartAt.Equal(now) {
		t.Fatalf("start_at = %v, want %v", job.StartAt, now)
	}
}

func TestApplyScheduleOnSaveFuture(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(2 * time.Hour)
	job := &storage.RestoreJob{}
	restorejob.ApplyScheduleOnSave(job, &future, now)

	if job.StartAt == nil || !job.StartAt.Equal(future.UTC()) {
		t.Fatalf("start_at = %v, want %v", job.StartAt, future.UTC())
	}
}

func TestRecordRunStartedSetsLastRun(t *testing.T) {
	runAt := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	job := &storage.RestoreJob{}
	restorejob.RecordRunStarted(job, runAt, runAt)

	if job.LastRun == nil || !job.LastRun.Equal(runAt) {
		t.Fatalf("last_run = %v, want %v", job.LastRun, runAt)
	}
}
