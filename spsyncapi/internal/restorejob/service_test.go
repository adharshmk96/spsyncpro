package restorejob_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/organization"
	"spsyncapi/internal/restorejob"
	"spsyncapi/internal/storage"
)

const testMemberA = "member-restore-job-a"

type mockRestoreRunStarter struct {
	immediate []string
	scheduled []scheduledRun
}

type scheduledRun struct {
	jobID string
	at    time.Time
}

func (m *mockRestoreRunStarter) StartRun(_ context.Context, _, jobID string) error {
	m.immediate = append(m.immediate, jobID)
	return nil
}

func (m *mockRestoreRunStarter) StartRunAt(_ context.Context, _, jobID string, at time.Time) error {
	m.scheduled = append(m.scheduled, scheduledRun{jobID: jobID, at: at})
	return nil
}

func newTestRestoreJobService(t *testing.T, starter restorejob.RunStarter) (*restorejob.Service, string, string) {
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
		BucketName: "restore-bucket",
		BucketType: "s3",
		Config:     config,
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	svc, err := restorejob.NewService(restorejob.ServiceConfig{
		Repo:       storage.NewRestoreJobRepository(db),
		OrgRepo:    orgRepo,
		BucketRepo: bucketRepo,
		RunStarter: starter,
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("restore job service: %v", err)
	}

	return svc, org.ID, bucket.ID
}

func TestCreateImmediateRestoreStartsRun(t *testing.T) {
	starter := &mockRestoreRunStarter{}
	svc, orgID, bucketID := newTestRestoreJobService(t, starter)

	created, err := svc.Create(restorejob.CreateInput{
		MemberID: testMemberA,
		Active:   true,
		JobConfig: restorejob.JobConfigInput{
			OrganizationID: orgID,
			BucketStoreID:  bucketID,
			SharePointSite: "https://tenant.sharepoint.com/sites/demo",
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(starter.immediate) != 1 || starter.immediate[0] != created.ID {
		t.Fatalf("expected immediate run for %s, got %v", created.ID, starter.immediate)
	}
	if len(starter.scheduled) != 0 {
		t.Fatalf("expected no scheduled runs, got %v", starter.scheduled)
	}

	got, err := svc.Get(testMemberA, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.StartAt == nil {
		t.Fatal("expected start_at to be set for immediate restore")
	}
	if got.LastRun != nil {
		t.Fatalf("last_run = %v, want nil before work starts with mock starter", got.LastRun)
	}
}

func TestCreateScheduledRestoreStartsRunAt(t *testing.T) {
	starter := &mockRestoreRunStarter{}
	svc, orgID, bucketID := newTestRestoreJobService(t, starter)

	startAt := time.Now().UTC().Add(2 * time.Hour)
	created, err := svc.Create(restorejob.CreateInput{
		MemberID: testMemberA,
		StartAt:  &startAt,
		Active:   true,
		JobConfig: restorejob.JobConfigInput{
			OrganizationID: orgID,
			BucketStoreID:  bucketID,
			SharePointSite: "https://tenant.sharepoint.com/sites/demo",
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(starter.immediate) != 0 {
		t.Fatalf("expected no immediate runs, got %v", starter.immediate)
	}
	if len(starter.scheduled) != 1 || starter.scheduled[0].jobID != created.ID {
		t.Fatalf("expected scheduled run for %s, got %v", created.ID, starter.scheduled)
	}

	got, err := svc.Get(testMemberA, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.StartAt == nil || !got.StartAt.Equal(startAt) {
		t.Fatalf("start_at = %v, want %v", got.StartAt, startAt)
	}
	if got.LastRun != nil {
		t.Fatalf("last_run = %v, want nil before scheduled work starts", got.LastRun)
	}
}

func TestCreateRejectsPastStartAt(t *testing.T) {
	starter := &mockRestoreRunStarter{}
	svc, orgID, bucketID := newTestRestoreJobService(t, starter)

	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err := svc.Create(restorejob.CreateInput{
		MemberID: testMemberA,
		StartAt:  &past,
		Active:   true,
		JobConfig: restorejob.JobConfigInput{
			OrganizationID: orgID,
			BucketStoreID:  bucketID,
			SharePointSite: "site",
		},
	})
	if !errors.Is(err, restorejob.ErrInvalidStartAtPast) {
		t.Fatalf("expected past start_at error, got: %v", err)
	}
}
