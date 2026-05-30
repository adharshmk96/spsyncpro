package temporal_test

import (
	"testing"

	"spsyncapi/internal/temporal"
)

func TestBackupScheduleID(t *testing.T) {
	got := temporal.BackupScheduleID("job-1")
	want := "backup-job-job-1"
	if got != want {
		t.Fatalf("BackupScheduleID() = %q, want %q", got, want)
	}
}

func TestWorkflowIDs(t *testing.T) {
	if got := temporal.BackupWorkflowID("run-1"); got != "backup-run-run-1" {
		t.Fatalf("BackupWorkflowID() = %q", got)
	}
	if got := temporal.RestoreWorkflowID("run-2"); got != "restore-run-run-2" {
		t.Fatalf("RestoreWorkflowID() = %q", got)
	}
}
