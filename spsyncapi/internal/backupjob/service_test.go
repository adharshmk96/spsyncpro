package backupjob_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/backupjob"
	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/organization"
	"spsyncapi/internal/storage"

	"gorm.io/gorm"
)

const (
	testMemberA = "member-test-a"
	testMemberB = "member-test-b"
)

type testEnv struct {
	svc      *backupjob.Service
	orgSvc   *organization.Service
	bucketID string
	orgID    string
	db       *gorm.DB
	logger   *slog.Logger
}

func newTestBackupJobEnv(t *testing.T) testEnv {
	t.Helper()

	db, err := storage.Open("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	enc, err := crypto.NewSecretEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	orgRepo := storage.NewOrganizationRepository(db)
	bucketRepo := storage.NewBucketStoreRepository(db)

	orgSvc, err := organization.NewService(organization.ServiceConfig{
		Repo:      orgRepo,
		Encryptor: enc,
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("org service: %v", err)
	}

	bucketSvc, err := bucketstore.NewService(bucketstore.ServiceConfig{
		Repo:      bucketRepo,
		Encryptor: enc,
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("bucket service: %v", err)
	}

	org, err := orgSvc.Create(organization.CreateInput{
		MemberID:     testMemberA,
		Name:         "Test Org",
		TenantID:     "tenant-1",
		ClientID:     "client-1",
		TenantSecret: "secret-1",
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	config, err := json.Marshal(bucketstore.S3Config{
		Server: "https://s3.example.com", AccessKey: "k", SecretKey: "s",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	bucket, err := bucketSvc.Create(bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "backup-bucket",
		BucketType: "s3",
		Config:     config,
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	svc, err := backupjob.NewService(backupjob.ServiceConfig{
		Repo:       storage.NewBackupJobRepository(db),
		OrgRepo:    orgRepo,
		BucketRepo: bucketRepo,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("backup job service: %v", err)
	}

	return testEnv{
		svc:      svc,
		orgSvc:   orgSvc,
		orgID:    org.ID,
		bucketID: bucket.ID,
		db:       db,
		logger:   logger,
	}
}

func TestCreateGetListDelete(t *testing.T) {
	env := newTestBackupJobEnv(t)
	oneTime := time.Now().UTC().Add(2 * time.Hour)
	startAt := time.Now().UTC().Add(-1 * time.Hour)
	endAt := time.Now().UTC().Add(10 * time.Hour)

	created, err := env.svc.Create(backupjob.CreateInput{
		MemberID: testMemberA,
		StartAt:  startAtPtr(startAt),
		EndAt:    startAtPtr(endAt),
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			OneTime: startAtPtr(oneTime),
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "https://tenant.sharepoint.com/sites/demo",
			Filters: backupjob.FilterInput{
				DocumentLibrariesCSV: "Docs,Invoices",
			},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := env.svc.Get(testMemberA, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.JobConfig.OrganizationID != env.orgID {
		t.Fatalf("organization: got %q", got.JobConfig.OrganizationID)
	}

	list, err := env.svc.List(testMemberA)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len: got %d", len(list))
	}

	if err := env.svc.Delete(testMemberA, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := env.svc.Get(testMemberA, created.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestCreateScheduleValidation(t *testing.T) {
	env := newTestBackupJobEnv(t)
	interval := int64(60)
	cron := "*/5 * * * *"
	oneTime := time.Now().UTC().Add(1 * time.Hour)

	_, err := env.svc.Create(backupjob.CreateInput{
		MemberID: testMemberA,
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: &interval,
			Cron:            &cron,
			OneTime:         &oneTime,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "site",
		},
	})
	if err == nil {
		t.Fatal("expected invalid schedule error")
	}
}

func TestUpdateReplacesSchedule(t *testing.T) {
	env := newTestBackupJobEnv(t)
	interval := int64(3600)

	created, err := env.svc.Create(backupjob.CreateInput{
		MemberID: testMemberA,
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: &interval,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "site",
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cron := "0 */2 * * *"
	updated, err := env.svc.Update(testMemberA, backupjob.UpdateInput{
		ID:     created.ID,
		Active: true,
		Schedule: backupjob.ScheduleInput{
			Cron: &cron,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
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

func TestMemberIsolation(t *testing.T) {
	env := newTestBackupJobEnv(t)
	interval := int64(3600)

	created, err := env.svc.Create(backupjob.CreateInput{
		MemberID: testMemberA,
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: &interval,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "site",
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := env.svc.Get(testMemberB, created.ID); !errors.Is(err, backupjob.ErrBackupJobNotFound) {
		t.Fatalf("expected not found for other member, got: %v", err)
	}

	list, err := env.svc.List(testMemberB)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("other member list should be empty, got %d", len(list))
	}
}

func TestCreateImmediateOneTimeStartsRun(t *testing.T) {
	env := newTestBackupJobEnv(t)
	starter := &mockRunStarter{}
	syncer := &mockScheduleSyncer{}

	svc, err := backupjob.NewService(backupjob.ServiceConfig{
		Repo:           storage.NewBackupJobRepository(env.db),
		OrgRepo:        storage.NewOrganizationRepository(env.db),
		BucketRepo:     storage.NewBucketStoreRepository(env.db),
		ScheduleSyncer: syncer,
		RunStarter:     starter,
		Logger:         env.logger,
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}

	created, err := svc.Create(backupjob.CreateInput{
		MemberID: testMemberA,
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			Type: "one_time",
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "https://tenant.sharepoint.com/sites/demo",
			Filters: backupjob.FilterInput{
				DocumentLibrariesCSV: "Docs",
			},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(starter.jobIDs) != 1 || starter.jobIDs[0] != created.ID {
		t.Fatalf("expected run starter called for job %s, got %v", created.ID, starter.jobIDs)
	}
	if len(syncer.synced) != 0 {
		t.Fatalf("expected schedule sync skipped, got %v", syncer.synced)
	}
}

func TestCreateScheduledOneTimeStartsRunAt(t *testing.T) {
	env := newTestBackupJobEnv(t)
	starter := &mockRunStarter{}
	syncer := &mockScheduleSyncer{}

	svc, err := backupjob.NewService(backupjob.ServiceConfig{
		Repo:           storage.NewBackupJobRepository(env.db),
		OrgRepo:        storage.NewOrganizationRepository(env.db),
		BucketRepo:     storage.NewBucketStoreRepository(env.db),
		ScheduleSyncer: syncer,
		RunStarter:     starter,
		Logger:         env.logger,
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}

	oneTime := time.Now().UTC().Add(2 * time.Hour)
	created, err := svc.Create(backupjob.CreateInput{
		MemberID: testMemberA,
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			Type:    "one_time",
			OneTime: startAtPtr(oneTime),
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "https://tenant.sharepoint.com/sites/demo",
			Filters: backupjob.FilterInput{
				DocumentLibrariesCSV: "Docs",
			},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(starter.jobIDs) != 0 {
		t.Fatalf("expected no immediate runs, got %v", starter.jobIDs)
	}
	if len(starter.scheduled) != 1 || starter.scheduled[0].jobID != created.ID {
		t.Fatalf("expected scheduled run for job %s, got %v", created.ID, starter.scheduled)
	}
	if len(syncer.synced) != 0 {
		t.Fatalf("expected schedule sync skipped, got %v", syncer.synced)
	}
}

func TestCreate_InvalidCrossMemberReferences(t *testing.T) {
	env := newTestBackupJobEnv(t)
	interval := int64(3600)

	_, err := env.svc.Create(backupjob.CreateInput{
		MemberID: testMemberB,
		Active:   true,
		Schedule: backupjob.ScheduleInput{
			IntervalSeconds: &interval,
		},
		JobConfig: backupjob.JobConfigInput{
			OrganizationID: env.orgID,
			BucketStoreID:  env.bucketID,
			SharePointSite: "site",
		},
	})
	if !errors.Is(err, backupjob.ErrInvalidOrganizationID) {
		t.Fatalf("expected invalid organization error, got: %v", err)
	}
}

type mockRunStarter struct {
	jobIDs    []string
	scheduled []scheduledBackupRun
}

type scheduledBackupRun struct {
	jobID string
	at    time.Time
}

func (m *mockRunStarter) StartRun(_ context.Context, _, jobID string) error {
	m.jobIDs = append(m.jobIDs, jobID)
	return nil
}

func (m *mockRunStarter) StartRunAt(_ context.Context, _, jobID string, at time.Time) error {
	m.scheduled = append(m.scheduled, scheduledBackupRun{jobID: jobID, at: at})
	return nil
}

type mockScheduleSyncer struct {
	synced []string
}

func (m *mockScheduleSyncer) SyncJob(_ context.Context, job *storage.BackupJob) error {
	m.synced = append(m.synced, job.ID)
	return nil
}

func (m *mockScheduleSyncer) DeleteJobSchedule(context.Context, string) error {
	return nil
}

func startAtPtr(v time.Time) *time.Time {
	return &v
}
