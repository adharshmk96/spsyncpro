package bucketstore_test

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/storage"
)

func newTestBucketStoreService(t *testing.T) *bucketstore.Service {
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

func TestCreate_Get_List_Delete_S3(t *testing.T) {
	svc := newTestBucketStoreService(t)

	config, err := json.Marshal(bucketstore.S3Config{
		Server:    "https://s3.example.com",
		AccessKey: "key",
		SecretKey: "secret",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	created, err := svc.Create(bucketstore.CreateInput{
		BucketName: "backup-primary",
		BucketType: "s3",
		Config:     config,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.BucketName != "backup-primary" || got.BucketType != "s3" {
		t.Fatalf("unexpected details: %+v", got)
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

func TestCreate_Azure(t *testing.T) {
	svc := newTestBucketStoreService(t)

	config, err := json.Marshal(bucketstore.AzureConfig{
		ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test",
	})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	created, err := svc.Create(bucketstore.CreateInput{
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

	config, _ := json.Marshal(bucketstore.S3Config{
		Server: "https://s3.example.com", AccessKey: "k", SecretKey: "s",
	})

	in := bucketstore.CreateInput{
		BucketName: "dup-bucket",
		BucketType: "s3",
		Config:     config,
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

func TestUpdate_ConfigOptional(t *testing.T) {
	svc := newTestBucketStoreService(t)

	config, _ := json.Marshal(bucketstore.S3Config{
		Server: "https://s3.example.com", AccessKey: "k", SecretKey: "s",
	})

	created, err := svc.Create(bucketstore.CreateInput{
		BucketName: "store-1",
		BucketType: "s3",
		Config:     config,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(bucketstore.UpdateInput{
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
		BucketName: "bad-config",
		BucketType: "s3",
		Config:     config,
	})
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}
