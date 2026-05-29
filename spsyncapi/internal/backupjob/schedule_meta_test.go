package backupjob_test

import (
	"testing"
	"time"

	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/storage"
)

func TestScheduleTypeRecurringInterval(t *testing.T) {
	interval := int64(3600)
	job := &storage.BackupJob{ScheduleIntervalSeconds: &interval}
	if got := backupjob.ScheduleType(job); got != backupjob.ScheduleTypeRecurring {
		t.Fatalf("ScheduleType = %q, want recurring", got)
	}
}

func TestRecordRunStartedRecurringSetsNextRun(t *testing.T) {
	interval := int64(3600)
	job := &storage.BackupJob{ScheduleIntervalSeconds: &interval}
	runAt := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	backupjob.RecordRunStarted(job, runAt, runAt)

	if job.LastRun == nil || !job.LastRun.Equal(runAt) {
		t.Fatalf("last_run = %v, want %v", job.LastRun, runAt)
	}
	wantNext := runAt.Add(time.Hour)
	if job.NextRun == nil || !job.NextRun.Equal(wantNext) {
		t.Fatalf("next_run = %v, want %v", job.NextRun, wantNext)
	}
}

func TestRecordRunStartedOneTimeClearsNextRun(t *testing.T) {
	future := time.Now().UTC().Add(2 * time.Hour)
	job := &storage.BackupJob{ScheduleOneTime: &future}
	runAt := time.Now().UTC()
	backupjob.RecordRunStarted(job, runAt, runAt)

	if job.NextRun != nil {
		t.Fatalf("next_run = %v, want nil for one-time after run", job.NextRun)
	}
}

func TestApplyScheduleMetadataFutureOneTime(t *testing.T) {
	future := time.Now().UTC().Add(2 * time.Hour)
	job := &storage.BackupJob{ScheduleOneTime: &future}
	backupjob.ApplyScheduleMetadataOnSave(job, time.Now().UTC())

	if job.NextRun == nil || !job.NextRun.Equal(future) {
		t.Fatalf("next_run = %v, want %v", job.NextRun, future)
	}
}
