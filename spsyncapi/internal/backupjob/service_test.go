package backupjob_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/storage"
)

func newTestBackupJobService(t *testing.T) *backupjob.Service {
	t.Helper()

	db, err := storage.Open("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	svc, err := backupjob.NewService(backupjob.ServiceConfig{
		Repo:   storage.NewBackupJobRepository(db),
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func TestCreateGetListDelete(t *testing.T) {
	svc := newTestBackupJobService(t)
	oneTime := time.Now().UTC().Add(2 * time.Hour)
	startAt := time.Now().UTC().Add(-1 * time.Hour)
	endAt := time.Now().UTC().Add(10 * time.Hour)

	created, err := svc.Create(backupjob.CreateInput{
		StartAt: startAtPtr(startAt),
		EndAt:   startAtPtr(endAt),
		Active:  true,
		Schedule: backupjob.ScheduleInput{
			OneTime: startAtPtr(oneTime),
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: "org-1",
			BucketStoreID:  "bucket-1",
			SharePointSite: "https://tenant.sharepoint.com/sites/demo",
			Filters: backupjob.FilterInput{
				DocumentLibrariesCSV: "Docs,Invoices",
			},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.JobConfig.OrganizationID != "org-1" {
		t.Fatalf("organization: got %q", got.JobConfig.OrganizationID)
	}

	list, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len: got %d", len(list))
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := svc.Get(created.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestCreateScheduleValidation(t *testing.T) {
	svc := newTestBackupJobService(t)
	interval := int64(60)
	cron := "*/5 * * * *"
	oneTime := time.Now().UTC().Add(1 * time.Hour)

	_, err := svc.Create(backupjob.CreateInput{
		Active: true,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: &interval,
			Cron:            &cron,
			OneTime:         &oneTime,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: "org-1",
			BucketStoreID:  "bucket-1",
			SharePointSite: "site",
		},
	})
	if err == nil {
		t.Fatal("expected invalid schedule error")
	}
}

func TestUpdateReplacesSchedule(t *testing.T) {
	svc := newTestBackupJobService(t)
	interval := int64(3600)

	created, err := svc.Create(backupjob.CreateInput{
		Active: true,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: &interval,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: "org-1",
			BucketStoreID:  "bucket-1",
			SharePointSite: "site",
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cron := "0 */2 * * *"
	updated, err := svc.Update(backupjob.UpdateInput{
		ID:     created.ID,
		Active: true,
		Schedule: backupjob.ScheduleInput{
			Cron: &cron,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: "org-1",
			BucketStoreID:  "bucket-1",
			SharePointSite: "site",
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Schedule.Cron == nil || *updated.Schedule.Cron != cron {
		t.Fatal("expected cron schedule after update")
	}
	if updated.Schedule.IntervalSeconds != nil {
		t.Fatal("expected interval schedule to be cleared")
	}
}

func startAtPtr(v time.Time) *time.Time {
	return &v
}
