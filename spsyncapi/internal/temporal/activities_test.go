package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/crypto"
	"spsyncapi/internal/storage"
	"spsyncapi/pkg/azureblob"
	"spsyncapi/pkg/graphapi"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockGraphService struct {
	tokenCalls     int
	siteID         string
	drives         []graphapi.Drive
	itemsByDrive   map[string][]graphapi.DriveItem
	itemsPageDone  map[string]bool
	drivesPageDone bool
	downloadBody   string
	uploadWhole    string
	uploadChunk    struct {
		path string
		size int64
	}
}

func (m *mockGraphService) GetAccessToken() (string, error) {
	m.tokenCalls++
	return "token", nil
}

func (m *mockGraphService) ValidateToken(string) (bool, error) { return true, nil }
func (m *mockGraphService) FetchFromGraphApi(string) (int, []byte, error) {
	return http.StatusOK, nil, nil
}

func (m *mockGraphService) GetSiteId(string) (string, error) {
	return m.siteID, nil
}

func (m *mockGraphService) GetDriveId(string, driveName string) (string, error) {
	for _, d := range m.drives {
		if d.Name == driveName {
			return d.ID, nil
		}
	}
	return "", graphapi.ErrDriveNotFound
}

func (m *mockGraphService) GetDriveList(string) <-chan graphapi.Drive {
	ch := make(chan graphapi.Drive, len(m.drives))
	for _, d := range m.drives {
		ch <- d
	}
	close(ch)
	return ch
}

func (m *mockGraphService) ListDrivesPage(_ string, pageURL string) ([]graphapi.Drive, string, error) {
	if pageURL != "" || m.drivesPageDone {
		return nil, "", nil
	}
	m.drivesPageDone = true
	return m.drives, "", nil
}

func (m *mockGraphService) ListDriveItemsPage(driveID string, state *graphapi.DriveCrawlState) ([]graphapi.DriveItem, *graphapi.DriveCrawlState, bool, error) {
	if m.itemsPageDone == nil {
		m.itemsPageDone = make(map[string]bool)
	}
	if m.itemsPageDone[driveID] {
		return nil, state, true, nil
	}
	m.itemsPageDone[driveID] = true
	items := m.itemsByDrive[driveID]
	return items, &graphapi.DriveCrawlState{}, true, nil
}

func (m *mockGraphService) GetDriveItems(driveID string) <-chan graphapi.DriveItem {
	ch := make(chan graphapi.DriveItem, len(m.itemsByDrive[driveID]))
	for _, item := range m.itemsByDrive[driveID] {
		ch <- item
	}
	close(ch)
	return ch
}

func (m *mockGraphService) GetDriveItemDownload(string, string) (*http.Response, error) {
	body := m.downloadBody
	if body == "" {
		body = "file-content"
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}, nil
}

func (m *mockGraphService) CreateDocumentLibrary(string, string) (string, error) {
	return "new-list-id", nil
}

func (m *mockGraphService) UploadDriveItemWhole(_ string, itemPath string, _ io.Reader) error {
	m.uploadWhole = itemPath
	return nil
}

func (m *mockGraphService) UploadDriveItemChunked(_, itemPath string, totalBytes int64, _ io.Reader) error {
	m.uploadChunk.path = itemPath
	m.uploadChunk.size = totalBytes
	return nil
}

type mockAzureService struct {
	containerCreated bool
	blobs            []azureblob.Blob
	blobsPageDone    bool
	uploaded         []string
	downloadBody     []byte
}

func (m *mockAzureService) CreateContainer(string) error {
	m.containerCreated = true
	return nil
}

func (m *mockAzureService) FetchBlobs(string, []string) <-chan azureblob.Blob {
	ch := make(chan azureblob.Blob, len(m.blobs))
	for _, b := range m.blobs {
		ch <- b
	}
	close(ch)
	return ch
}

func (m *mockAzureService) ListBlobsPage(_ string, marker string, _ []string) ([]azureblob.Blob, string, error) {
	if marker != "" || m.blobsPageDone {
		return nil, "", nil
	}
	m.blobsPageDone = true
	return m.blobs, "", nil
}

func (m *mockAzureService) UploadBlob(_, blobName string, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	m.uploaded = append(m.uploaded, blobName+":"+string(body))
	return nil
}

func (m *mockAzureService) DownloadBlobToStream(string, string) (*blob.DownloadStreamResponse, error) {
	body := m.downloadBody
	if body == nil {
		body = []byte("restore-content")
	}
	length := int64(len(body))
	return &blob.DownloadStreamResponse{
		DownloadResponse: blob.DownloadResponse{
			Body:          io.NopCloser(bytes.NewReader(body)),
			ContentLength: &length,
		},
	}, nil
}

func setupAzureTransferFixtures(t *testing.T, db *gorm.DB, memberID string) (orgID, bucketID string, enc *crypto.SecretEncryptor) {
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

	config, err := json.Marshal(bucketstore.AzureConfig{
		ConnectionString: "DefaultEndpointsProtocol=https;AccountName=test;AccountKey=key;EndpointSuffix=core.windows.net",
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
		BucketName:      "backup-container",
		BucketType:      storage.BucketTypeAzure,
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
	syncIn := SyncFileMetadataPageInput{
		RunID:    in.RunID,
		JobID:    in.JobID,
		MemberID: in.MemberID,
		Kind:     in.Kind,
	}
	for {
		out, err := acts.SyncFileMetadataPage(ctx, syncIn)
		if err != nil {
			return err
		}
		if out.Complete {
			break
		}
	}

	for {
		pending, err := acts.ListPendingFileLogs(ctx, ListPendingFileLogsInput{
			RunID:    in.RunID,
			JobID:    in.JobID,
			MemberID: in.MemberID,
			Kind:     in.Kind,
			Offset:   0,
			Limit:    listPendingFileLogsBatchSize,
		})
		if err != nil {
			return err
		}
		if len(pending.Files) == 0 {
			break
		}
		for _, file := range pending.Files {
			if err := acts.TransferSingleFile(ctx, TransferSingleFileInput{
				RunID:    in.RunID,
				JobID:    in.JobID,
				MemberID: in.MemberID,
				Kind:     in.Kind,
				File:     file,
			}); err != nil {
				return err
			}
		}
	}

	return acts.FinalizeRun(ctx, in)
}

func TestBackupTransferChainWithMocks(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)

	memberID := "member-1"
	orgID, bucketID, enc := setupAzureTransferFixtures(t, db, memberID)

	graphMock := &mockGraphService{
		siteID: "site-1",
		drives: []graphapi.Drive{{ID: "drive-1", Name: "Documents"}},
		itemsByDrive: map[string][]graphapi.DriveItem{
			"drive-1": {
				{ID: "item-1", Name: "file.txt", FilePath: "/file.txt", Size: 12},
				{ID: "item-2", Name: "skip.txt", FilePath: "/skip.txt", Size: 2},
			},
		},
	}
	azureMock := &mockAzureService{}

	now := time.Now().UTC()
	minSize := int64(10)
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.BackupJob{
		ID:                jobID,
		MemberID:          memberID,
		Active:            true,
		OrganizationID:    orgID,
		BucketStoreID:     bucketID,
		SharePointSite:    "https://tenant.sharepoint.com/sites/demo",
		FilterMinFileSize: &minSize,
		CreatedAt:         now,
		UpdatedAt:         now,
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
		BackupRunRepo: runRepo,
		BackupJobRepo: jobRepo,
		OrgRepo:       storage.NewOrganizationRepository(db),
		BucketStoreRepo: storage.NewBucketStoreRepository(db),
		Encryptor:     enc,
		Logger:        logger,
		GraphServiceBuilder: func(jobContext) graphapi.Service { return graphMock },
		AzureServiceBuilder: func(jobContext) azureblob.Service { return azureMock },
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
	if total != 1 {
		t.Fatalf("file count = %d, want 1 (size filter should skip one file)", total)
	}
	if !azureMock.containerCreated {
		t.Fatal("expected azure container to be created")
	}
	if len(azureMock.uploaded) != 1 {
		t.Fatalf("uploaded = %#v, want 1 upload", azureMock.uploaded)
	}

	if err := runTransferChain(context.Background(), acts, in); err != nil {
		t.Fatalf("second transfer chain: %v", err)
	}
}

func TestRestoreTransferChainWithMocks(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewRestoreJobRepository(db)
	runRepo := storage.NewRestoreRunRepository(db)

	memberID := "member-1"
	orgID, bucketID, enc := setupAzureTransferFixtures(t, db, memberID)

	graphMock := &mockGraphService{
		siteID: "site-1",
		drives: []graphapi.Drive{{ID: "drive-1", Name: "Documents"}},
	}
	azureMock := &mockAzureService{
		blobs: []azureblob.Blob{
			{FullPath: "Documents/folder/file.txt", Size: 13},
		},
		downloadBody: []byte("hello-restore"),
	}

	now := time.Now().UTC()
	jobID := uuid.NewString()
	if err := jobRepo.Create(&storage.RestoreJob{
		ID:             jobID,
		MemberID:       memberID,
		StartAt:        &now,
		Active:         true,
		OrganizationID: orgID,
		BucketStoreID:  bucketID,
		SharePointSite: "https://tenant.sharepoint.com/sites/demo",
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
		RestoreRunRepo:  runRepo,
		RestoreJobRepo:  jobRepo,
		OrgRepo:         storage.NewOrganizationRepository(db),
		BucketStoreRepo: storage.NewBucketStoreRepository(db),
		Encryptor:       enc,
		Logger:          logger,
		GraphServiceBuilder: func(jobContext) graphapi.Service { return graphMock },
		AzureServiceBuilder: func(jobContext) azureblob.Service { return azureMock },
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

	if graphMock.uploadWhole != "folder/file.txt" {
		t.Fatalf("upload path = %q, want folder/file.txt", graphMock.uploadWhole)
	}
}

func TestTransferSingleBackupFileIdempotent(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	jobRepo := storage.NewBackupJobRepository(db)
	runRepo := storage.NewBackupRunRepository(db)

	memberID := "member-1"
	orgID, bucketID, enc := setupAzureTransferFixtures(t, db, memberID)

	graphMock := &mockGraphService{}
	azureMock := &mockAzureService{}

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

	if err := runRepo.UpdateMetadataSyncState(runID, storage.MetadataSyncComplete, "", ""); err != nil {
		t.Fatalf("mark metadata complete: %v", err)
	}
	syncedAt := now
	if err := runRepo.UpsertFileLog(&storage.BackupRunFileTransfer{
		ID:               uuid.NewString(),
		RunID:            runID,
		FilePath:         "Documents/file.txt",
		Status:           storage.FileLogStatusPending,
		DriveID:          "drive-1",
		DriveItemID:      "item-1",
		Size:             12,
		MetadataSyncedAt: &syncedAt,
		CreatedAt:        now,
	}); err != nil {
		t.Fatalf("seed file log: %v", err)
	}

	acts := &Activities{
		BackupRunRepo:   runRepo,
		BackupJobRepo:   jobRepo,
		OrgRepo:         storage.NewOrganizationRepository(db),
		BucketStoreRepo: storage.NewBucketStoreRepository(db),
		Encryptor:       enc,
		Logger:          logger,
		GraphServiceBuilder: func(jobContext) graphapi.Service { return graphMock },
		AzureServiceBuilder: func(jobContext) azureblob.Service { return azureMock },
	}

	file := FileDescriptor{
		Path:        "Documents/file.txt",
		DriveID:     "drive-1",
		DriveItemID: "item-1",
		Size:        12,
	}
	in := TransferSingleFileInput{
		RunID:    runID,
		JobID:    jobID,
		MemberID: memberID,
		Kind:     RunKindBackup,
		File:     file,
	}

	if err := acts.TransferSingleFile(context.Background(), in); err != nil {
		t.Fatalf("first transfer: %v", err)
	}
	if err := acts.TransferSingleFile(context.Background(), in); err != nil {
		t.Fatalf("second transfer: %v", err)
	}

	ft, err := runRepo.FindFileTransferByRunAndPath(runID, file.Path)
	if err != nil {
		t.Fatalf("find file log: %v", err)
	}
	if ft == nil || ft.Status != storage.FileLogStatusSuccess {
		t.Fatalf("status = %q, want success", ft.Status)
	}
}

func TestLoadJobContextRejectsS3(t *testing.T) {
	db, err := storage.OpenSQLite("file::memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	memberID := "member-1"
	orgID, _, enc := setupAzureTransferFixtures(t, db, memberID)

	s3Config, err := json.Marshal(bucketstore.S3Config{
		Server:    "https://s3.example.com",
		AccessKey: "k",
		SecretKey: "s",
	})
	if err != nil {
		t.Fatalf("marshal s3 config: %v", err)
	}
	configEnc, err := enc.Encrypt(string(s3Config))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	bucketID := uuid.NewString()
	now := time.Now().UTC()
	if err := storage.NewBucketStoreRepository(db).Create(&storage.BucketStore{
		ID:              bucketID,
		MemberID:        memberID,
		BucketName:      "s3-bucket",
		BucketType:      storage.BucketTypeS3,
		ConfigEncrypted: configEnc,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("create s3 bucket: %v", err)
	}

	jobID := uuid.NewString()
	if err := storage.NewBackupJobRepository(db).Create(&storage.BackupJob{
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

	acts := &Activities{
		BackupJobRepo:   storage.NewBackupJobRepository(db),
		OrgRepo:         storage.NewOrganizationRepository(db),
		BucketStoreRepo: storage.NewBucketStoreRepository(db),
		Encryptor:       enc,
		Logger:          logger,
	}

	_, err = acts.loadJobContext(jobID, memberID, RunKindBackup)
	if err == nil {
		t.Fatal("expected error for s3 bucket type")
	}
}
