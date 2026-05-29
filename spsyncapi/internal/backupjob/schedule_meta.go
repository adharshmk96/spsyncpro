package backupjob

import (
	"strings"
	"time"

	"spsyncapi/internal/storage"

	cronlib "github.com/robfig/cron/v3"
)

const (
	ScheduleTypeOneTime   = "one_time"
	ScheduleTypeRecurring = "recurring"
)

// ScheduleType derives the schedule type stored on a backup job row.
func ScheduleType(job *storage.BackupJob) string {
	if job == nil {
		return ScheduleTypeOneTime
	}
	if job.ScheduleIntervalSeconds != nil && *job.ScheduleIntervalSeconds > 0 {
		return ScheduleTypeRecurring
	}
	if job.ScheduleCron != nil && strings.TrimSpace(*job.ScheduleCron) != "" {
		return ScheduleTypeRecurring
	}
	return ScheduleTypeOneTime
}

// ApplyScheduleMetadataOnSave sets next_run for jobs that have not started yet.
// last_run is preserved when already set.
func ApplyScheduleMetadataOnSave(job *storage.BackupJob, now time.Time) {
	if job == nil {
		return
	}
	now = now.UTC()
	if job.LastRun != nil {
		job.NextRun = computeNextAfterRun(job, *job.LastRun)
	} else {
		job.NextRun = computeNextBeforeFirstRun(job, now)
	}
}

// RecordRunStarted sets last_run to the run start time and updates next_run for recurring jobs.
func RecordRunStarted(job *storage.BackupJob, runAt, now time.Time) {
	if job == nil {
		return
	}
	runAt = runAt.UTC()
	now = now.UTC()
	job.LastRun = &runAt
	job.NextRun = computeNextAfterRun(job, runAt)
	job.UpdatedAt = now
}

// RecordPendingRun sets next_run for a future one-shot execution without changing last_run.
func RecordPendingRun(job *storage.BackupJob, at, now time.Time) {
	if job == nil {
		return
	}
	at = at.UTC()
	now = now.UTC()
	job.NextRun = &at
	job.UpdatedAt = now
}

func computeNextBeforeFirstRun(job *storage.BackupJob, now time.Time) *time.Time {
	if ScheduleType(job) == ScheduleTypeOneTime {
		if job.ScheduleOneTime != nil {
			t := job.ScheduleOneTime.UTC()
			if t.After(now) {
				return &t
			}
			return nil
		}
		return nil
	}

	if job.StartAt != nil {
		start := job.StartAt.UTC()
		if start.After(now) {
			return &start
		}
	}
	return computeNextAfterRun(job, now)
}

func computeNextAfterRun(job *storage.BackupJob, lastRun time.Time) *time.Time {
	if ScheduleType(job) != ScheduleTypeRecurring {
		return nil
	}
	lastRun = lastRun.UTC()

	if job.ScheduleIntervalSeconds != nil && *job.ScheduleIntervalSeconds > 0 {
		next := lastRun.Add(time.Duration(*job.ScheduleIntervalSeconds) * time.Second)
		return &next
	}
	if job.ScheduleCron != nil {
		expr := strings.TrimSpace(*job.ScheduleCron)
		if expr == "" {
			return nil
		}
		schedule, err := cronlib.ParseStandard(expr)
		if err != nil {
			return nil
		}
		next := schedule.Next(lastRun)
		return &next
	}
	return nil
}
