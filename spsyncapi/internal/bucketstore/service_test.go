package bucketstore_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/storage"
)

const (
	testMemberA = "member-test-a"
	testMemberB = "member-test-b"
)

func newTestBucketStoreService(t *testing.T) *bucketstore.Service {
	t.Helper()

	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	enc, err := crypto.NewSecretEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("encryptor: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	svc, err := bucketstore.NewService(bucketstore.ServiceConfig{
		Repo:      storage.NewBucketStoreRepository(db),
		Encryptor: enc,
		Logger:    logger,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func s3Config(t *testing.T) json.RawMessage {
	t.Helper()
	config, err := json.Marshal(bucketstore.S3Config{
		Server:    "https://s3.example.com",
		AccessKey: "key",
		SecretKey: "secret",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	return config
}

func TestCreate_Get_List_Delete_S3(t *testing.T) {
	svc := newTestBucketStoreService(t)

	created, err := svc.Create(bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "backup-primary",
		BucketType: "s3",
		Config:     s3Config(t),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.Get(testMemberA, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.BucketName != "backup-primary" || got.BucketType != "s3" {
		t.Fatalf("unexpected details: %+v", got)
	}

	list, err := svc.List(testMemberA)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len: got %d", len(list))
	}

	if err := svc.Delete(testMemberA, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := svc.Get(testMemberA, created.ID); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestCreate_Azure(t *testing.T) {
	svc := newTestBucketStoreService(t)

	config, err := json.Marshal(bucketstore.AzureConfig{
		ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	created, err := svc.Create(bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "azure-backup",
		BucketType: "azure_blob",
		Config:     config,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.BucketType != "azure" {
		t.Fatalf("bucket type: got %q", created.BucketType)
	}
}

func TestCreate_DuplicateBucketName(t *testing.T) {
	svc := newTestBucketStoreService(t)

	in := bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "dup-bucket",
		BucketType: "s3",
		Config:     s3Config(t),
	}
	if _, err := svc.Create(in); err != nil {
		t.Fatalf("first create: %v", err)
	}

	in.Config, _ = json.Marshal(bucketstore.S3Config{
		Server: "https://other.example.com", AccessKey: "k2", SecretKey: "s2",
	})
	if _, err := svc.Create(in); err == nil {
		t.Fatal("expected duplicate bucket name error")
	}
}

func TestDuplicateBucketName_DifferentMembers(t *testing.T) {
	svc := newTestBucketStoreService(t)

	base := bucketstore.CreateInput{
		BucketName: "shared-bucket",
		BucketType: "s3",
		Config:     s3Config(t),
	}

	base.MemberID = testMemberA
	if _, err := svc.Create(base); err != nil {
		t.Fatalf("member A create: %v", err)
	}

	base.MemberID = testMemberB
	if _, err := svc.Create(base); err != nil {
		t.Fatalf("member B create with same bucket name: %v", err)
	}
}

func TestMemberIsolation(t *testing.T) {
	svc := newTestBucketStoreService(t)

	created, err := svc.Create(bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "private-bucket",
		BucketType: "s3",
		Config:     s3Config(t),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := svc.Get(testMemberB, created.ID); !errors.Is(err, bucketstore.ErrBucketStoreNotFound) {
		t.Fatalf("expected not found for other member, got: %v", err)
	}

	list, err := svc.List(testMemberB)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("other member list should be empty, got %d", len(list))
	}
}

func TestUpdate_ConfigOptional(t *testing.T) {
	svc := newTestBucketStoreService(t)

	created, err := svc.Create(bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "store-1",
		BucketType: "s3",
		Config:     s3Config(t),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(testMemberA, bucketstore.UpdateInput{
		ID:         created.ID,
		BucketName: "store-1-renamed",
		BucketType: "s3",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.BucketName != "store-1-renamed" {
		t.Fatalf("name: got %q", updated.BucketName)
	}
}

func TestCreate_InvalidConfig(t *testing.T) {
	svc := newTestBucketStoreService(t)

	config, _ := json.Marshal(map[string]string{"server": "only-server"})
	_, err := svc.Create(bucketstore.CreateInput{
		MemberID:   testMemberA,
		BucketName: "bad-config",
		BucketType: "s3",
		Config:     config,
	})
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}
