package temporal

import (
	"testing"
	"time"

	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/storage"
)

func TestUsesRunStarterScheduleViaBackupJob(t *testing.T) {
	interval := int64(60)
	job := &storage.BackupJob{ScheduleIntervalSeconds: &interval}
	if backupjob.UsesRunStarterSchedule(job) {
		t.Fatal("interval job should not use run starter")
	}
}

func TestPendingBackupRunAtOneTime(t *testing.T) {
	when := time.Now().UTC().Add(2 * time.Hour)
	job := &storage.BackupJob{ScheduleOneTime: &when}
	at, ok := pendingBackupRunAt(job)
	if !ok {
		t.Fatal("expected pending one_time")
	}
	if !at.Equal(when) {
		t.Fatalf("at = %v, want %v", at, when)
	}
}

func TestPendingBackupRunAtSkipsAfterLastRun(t *testing.T) {
	when := time.Now().UTC().Add(-time.Hour)
	last := when.Add(-time.Minute)
	job := &storage.BackupJob{
		ScheduleOneTime: &when,
		LastRun:         &last,
	}
	if _, ok := pendingBackupRunAt(job); ok {
		t.Fatal("expected no pending after last_run")
	}
}

func TestPendingBackupRunAtNextRun(t *testing.T) {
	next := time.Now().UTC().Add(time.Hour)
	job := &storage.BackupJob{NextRun: &next}
	at, ok := pendingBackupRunAt(job)
	if !ok {
		t.Fatal("expected pending next_run")
	}
	if !at.Equal(next) {
		t.Fatalf("at = %v, want %v", at, next)
	}
}
