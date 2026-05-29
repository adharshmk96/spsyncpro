package temporal

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupTransferFixtures(t *testing.T, db *gorm.DB, memberID string) (orgID, bucketID string, enc *crypto.SecretEncryptor) {
	t.Helper()

	enc, err := crypto.NewSecretEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}

	now := time.Now().UTC()
	orgRepo := storage.NewOrganizationRepository(db)

	tenantSecretEnc, err := enc.Encrypt("secret-1")
	if err != nil {
		t.Fatalf("encrypt tenant secret: %v", err)
	}
	orgID = uuid.NewString()
	if err := orgRepo.Create(&storage.Organization{
		ID:                    orgID,
		MemberID:              memberID,
		Name:                  "Test Org",
		TenantID:              "tenant-1",
		ClientID:              "client-1",
		TenantSecretEncrypted: tenantSecretEnc,
		Active:                true,
		CreatedAt:             now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("create org: %v", err)
	}

	config, err := json.Marshal(bucketstore.S3Config{
		Server:    "https://s3.example.com",
		AccessKey: "k",
		SecretKey: "s",
	})
	if err != nil {
		t.Fatalf("marshal bucket config: %v", err)
	}
	configEnc, err := enc.Encrypt(string(config))
	if err != nil {
		t.Fatalf("encrypt bucket config: %v", err)
	}

	bucketRepo := storage.NewBucketStoreRepository(db)
	bucketID = uuid.NewString()
	if err := bucketRepo.Create(&storage.BucketStore{
		ID:              bucketID,
		MemberID:        memberID,
		BucketName:      "backup-bucket",
		BucketType:      storage.BucketTypeS3,
		ConfigEncrypted: configEnc,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create bucket store: %v", err)
	}

	return orgID, bucketID, enc
}

func runTransferChain(ctx context.Context, acts *Activities, in FinalizeRunInput) error {
	meta, err := acts.FetchFileMetadata(ctx, FetchFileMetadataInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	})
	if err != nil {
		return err
	}

	for _, path := range meta.Paths {
		if err := acts.TransferSingleFile(ctx, TransferSingleFileInput{
			RunID:    in.RunID,
			JobID:    in.JobID,
			MemberID: in.MemberID,
			Kind:     in.Kind,
			FilePath: path,
		}); err != nil {
			return err
		}
	}

	return acts.FinalizeRun(ctx, in)
}

func TestBackupTransferChainIdempotent(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)

	memberID := "member-1"
	orgID, bucketID, enc := setupTransferFixtures(t, db, memberID)

	now := time.Now().UTC()
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.BackupJob{
		ID:             jobID,
		MemberID:       memberID,
		Active:         true,
		OrganizationID: orgID,
		BucketStoreID:  bucketID,
		SharePointSite: "https://example.com",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	runID := uuid.NewString()
	if err := runRepo.Create(&storage.BackupRun{
		ID:        runID,
		JobID:     jobID,
		MemberID:  memberID,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	acts := &Activities{
		BackupRunRepo:      runRepo,
		BackupJobRepo:      jobRepo,
		OrgRepo:            storage.NewOrganizationRepository(db),
		BucketStoreRepo:    storage.NewBucketStoreRepository(db),
		Encryptor:          enc,
		Logger:             logger,
		MetadataFetchDelay: time.Millisecond,
		TransferDelay:      time.Millisecond,
	}

	in := FinalizeRunInput{
		RunID:    runID,
		JobID:    jobID,
		MemberID: memberID,
		Kind:     RunKindBackup,
	}
	if err := runTransferChain(context.Background(), acts, in); err != nil {
		t.Fatalf("first transfer chain: %v", err)
	}

	run, err := runRepo.FindByID(runID, memberID)
	if err != nil {
		t.Fatalf("find run: %v", err)
	}
	if run.EndAt == nil {
		t.Fatal("expected run to be complete")
	}

	_, total, err := runRepo.ListFileTransfers(runID, 0, 200)
	if err != nil {
		t.Fatalf("list transfers: %v", err)
	}
	if total != metadataFileCount {
		t.Fatalf("file count = %d, want %d", total, metadataFileCount)
	}

	// Second invocation should be a no-op for completed run.
	if err := runTransferChain(context.Background(), acts, in); err != nil {
		t.Fatalf("second transfer chain: %v", err)
	}
}

func TestRestoreTransferChainSetsJobLastRun(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewRestoreJobRepository(db)
	runRepo := storage.NewRestoreRunRepository(db)

	memberID := "member-1"
	orgID, bucketID, enc := setupTransferFixtures(t, db, memberID)

	now := time.Now().UTC()
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.RestoreJob{
		ID:             jobID,
		MemberID:       memberID,
		StartAt:        &now,
		Active:         true,
		OrganizationID: orgID,
		BucketStoreID:  bucketID,
		SharePointSite: "https://example.com",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	runID := uuid.NewString()
	if err := runRepo.Create(&storage.RestoreRun{
		ID:        runID,
		JobID:     jobID,
		MemberID:  memberID,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	acts := &Activities{
		RestoreRunRepo:     runRepo,
		RestoreJobRepo:     jobRepo,
		OrgRepo:            storage.NewOrganizationRepository(db),
		BucketStoreRepo:    storage.NewBucketStoreRepository(db),
		Encryptor:          enc,
		Logger:             logger,
		MetadataFetchDelay: time.Millisecond,
		TransferDelay:      time.Millisecond,
	}

	in := FinalizeRunInput{
		RunID:    runID,
		JobID:    jobID,
		MemberID: memberID,
		Kind:     RunKindRestore,
	}
	if err := runTransferChain(context.Background(), acts, in); err != nil {
		t.Fatalf("transfer chain: %v", err)
	}

	job, err := jobRepo.FindActiveByID(jobID, memberID)
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if job.LastRun == nil {
		t.Fatal("expected restore job last_run to be set when work starts")
	}
}

func TestTransferSingleFileLogsConnectionContext(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)

	memberID := "member-1"
	orgID, bucketID, enc := setupTransferFixtures(t, db, memberID)

	now := time.Now().UTC()
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.BackupJob{
		ID:             jobID,
		MemberID:       memberID,
		Active:         true,
		OrganizationID: orgID,
		BucketStoreID:  bucketID,
		SharePointSite: "https://example.com",
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	runID := uuid.NewString()
	if err := runRepo.Create(&storage.BackupRun{
		ID:        runID,
		JobID:     jobID,
		MemberID:  memberID,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	acts := &Activities{
		BackupRunRepo:   runRepo,
		BackupJobRepo:   jobRepo,
		OrgRepo:         storage.NewOrganizationRepository(db),
		BucketStoreRepo: storage.NewBucketStoreRepository(db),
		Encryptor:       enc,
		Logger:          logger,
		TransferDelay:   time.Millisecond,
	}

	path := DummyFilePath(jobID, 1)
	if err := acts.TransferSingleFile(context.Background(), TransferSingleFileInput{
		RunID:    runID,
		JobID:    jobID,
		MemberID: memberID,
		Kind:     RunKindBackup,
		FilePath: path,
	}); err != nil {
		t.Fatalf("transfer single file: %v", err)
	}
}
