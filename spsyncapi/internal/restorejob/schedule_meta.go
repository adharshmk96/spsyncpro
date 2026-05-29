package restorejob

import (
	"time"

	"spsyncapi/internal/storage"
)

// ApplyScheduleOnSave sets start_at to now for immediate runs or to the requested future time.
func ApplyScheduleOnSave(job *storage.RestoreJob, requestedStartAt *time.Time, now time.Time) {
	if job == nil {
		return
	}
	now = now.UTC()
	if requestedStartAt != nil {
		t := requestedStartAt.UTC()
		job.StartAt = &t
		return
	}
	job.StartAt = &now
}

// RecordRunStarted sets last_run when restore work begins.
func RecordRunStarted(job *storage.RestoreJob, runAt, now time.Time) {
	if job == nil {
		return
	}
	runAt = runAt.UTC()
	now = now.UTC()
	job.LastRun = &runAt
	job.UpdatedAt = now
}
