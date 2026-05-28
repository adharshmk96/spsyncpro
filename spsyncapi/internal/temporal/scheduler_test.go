package temporal

import (
	"testing"
	"time"

	"spsyncapi/internal/storage"
)

func TestBuildScheduleSpecInterval(t *testing.T) {
	interval := int64(300)
	job := &storage.BackupJob{
		ID:                      "j1",
		ScheduleIntervalSeconds: &interval,
	}
	spec, oneTime, err := buildScheduleSpec(job)
	if err != nil {
		t.Fatalf("buildScheduleSpec: %v", err)
	}
	if oneTime {
		t.Fatal("expected oneTime false")
	}
	if len(spec.Intervals) != 1 || spec.Intervals[0].Every != 300*time.Second {
		t.Fatalf("intervals: %+v", spec.Intervals)
	}
}

func TestBuildScheduleSpecCron(t *testing.T) {
	cron := "0 0 * * *"
	job := &storage.BackupJob{
		ID:           "j1",
		ScheduleCron: &cron,
	}
	spec, _, err := buildScheduleSpec(job)
	if err != nil {
		t.Fatalf("buildScheduleSpec: %v", err)
	}
	if len(spec.CronExpressions) != 1 || spec.CronExpressions[0] != cron {
		t.Fatalf("cron: %+v", spec.CronExpressions)
	}
}

func TestBuildScheduleSpecOneTime(t *testing.T) {
	when := time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC)
	job := &storage.BackupJob{
		ID:              "j1",
		ScheduleOneTime: &when,
	}
	spec, oneTime, err := buildScheduleSpec(job)
	if err != nil {
		t.Fatalf("buildScheduleSpec: %v", err)
	}
	if !oneTime {
		t.Fatal("expected oneTime true")
	}
	if !spec.StartAt.Equal(when) {
		t.Fatalf("start_at = %v", spec.StartAt)
	}
	if len(spec.Calendars) != 1 {
		t.Fatalf("calendars: %+v", spec.Calendars)
	}
}
