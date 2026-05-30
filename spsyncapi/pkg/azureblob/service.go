package azureblob

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
)

// Config wires dependencies for the Azure Blob client.
type Config struct {
	ConnectionString string
	Logger           *slog.Logger
}

// Service defines Azure Blob Storage operations used for backup and restore.
type Service interface {
	CreateContainer(containerName string) error
	FetchBlobs(containerName string, filter []string) <-chan Blob
	ListBlobsPage(containerName, marker string, filter []string) (blobs []Blob, nextMarker string, err error)
	UploadBlob(containerName, blobName string, resp *http.Response) error
	DownloadBlobToStream(containerName, blobName string) (*blob.DownloadStreamResponse, error)
}

type service struct {
	connectionString string
	logger           *slog.Logger
}

// NewService constructs an Azure Blob client.
func NewService(cfg Config) Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &service{
		connectionString: cfg.ConnectionString,
		logger:           logger.With(slog.String("module", "azureblob")),
	}
}

func parseConnectionString(connectionString string) (accountName, accountKey string) {
	for _, part := range strings.Split(connectionString, ";") {
		if strings.HasPrefix(part, "AccountName=") {
			accountName = strings.TrimPrefix(part, "AccountName=")
		} else if strings.HasPrefix(part, "AccountKey=") {
			accountKey = strings.TrimPrefix(part, "AccountKey=")
		}
	}
	return accountName, accountKey
}

func generateBlobSASURL(connectionString, containerName, blobName string) (string, error) {
	expiryTime := time.Now().Add(time.Hour)
	permissions := sas.BlobPermissions{Read: true}

	accountName, accountKey := parseConnectionString(connectionString)
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return "", err
	}

	signatureValues := sas.BlobSignatureValues{
		ContainerName: containerName,
		BlobName:      blobName,
		ExpiryTime:    expiryTime,
		Permissions:   permissions.String(),
	}

	queryParams, err := signatureValues.SignWithSharedKey(credential)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s",
		accountName, containerName, blobName, queryParams.Encode()), nil
}

func (s *service) ListBlobsPage(containerName, marker string, filter []string) ([]Blob, string, error) {
	serviceClient, err := azblob.NewClientFromConnectionString(s.connectionString, nil)
	if err != nil {
		return nil, "", err
	}

	opts := &azblob.ListBlobsFlatOptions{}
	if marker != "" {
		opts.Marker = &marker
	}

	pager := serviceClient.NewListBlobsFlatPager(containerName, opts)
	if !pager.More() {
		return nil, "", nil
	}

	page, err := pager.NextPage(context.Background())
	if err != nil {
		return nil, "", err
	}

	var blobs []Blob
	for _, item := range page.Segment.BlobItems {
		if item.Name == nil {
			continue
		}
		itemName := *item.Name
		if !blobMatchesFilter(itemName, filter) {
			continue
		}

		downloadURL, err := generateBlobSASURL(s.connectionString, containerName, itemName)
		if err != nil {
			s.logger.Error("failed to generate blob SAS URL", slog.String("error", err.Error()))
			continue
		}

		var size int64
		if item.Properties != nil && item.Properties.ContentLength != nil {
			size = *item.Properties.ContentLength
		}
		blobs = append(blobs, Blob{FullPath: itemName, DownloadURL: downloadURL, Size: size})
	}

	nextMarker := ""
	if page.NextMarker != nil {
		nextMarker = *page.NextMarker
	}
	return blobs, nextMarker, nil
}

func blobMatchesFilter(itemName string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if strings.Contains(itemName, f) {
			return true
		}
	}
	return false
}

func (s *service) FetchBlobs(containerName string, filter []string) <-chan Blob {
	blobChan := make(chan Blob)

	go func() {
		defer close(blobChan)

		marker := ""
		for {
			blobs, next, err := s.ListBlobsPage(containerName, marker, filter)
			if err != nil {
				s.logger.Error("failed to list blobs page", slog.String("error", err.Error()))
				return
			}
			for _, b := range blobs {
				blobChan <- b
			}
			if next == "" {
				return
			}
			marker = next
		}
	}()

	return blobChan
}

func (s *service) CreateContainer(containerName string) error {
	client, err := azblob.NewClientFromConnectionString(s.connectionString, nil)
	if err != nil {
		return err
	}

	_, err = client.CreateContainer(context.Background(), containerName, nil)
	if err != nil {
		if strings.Contains(err.Error(), "ContainerAlreadyExists") {
			s.logger.Info("container already exists", slog.String("container_name", containerName))
			return nil
		}
		s.logger.Error("failed to create container", slog.String("error", err.Error()))
		return err
	}

	s.logger.Info("container created", slog.String("container_name", containerName))
	time.Sleep(2 * time.Second)
	return nil
}

func (s *service) UploadBlob(containerName, blobName string, resp *http.Response) error {
	client, err := azblob.NewClientFromConnectionString(s.connectionString, nil)
	if err != nil {
		return err
	}

	uploadOptions := &blockblob.UploadStreamOptions{
		BlockSize: 4 * 1024 * 1024,
	}

	stream, err := client.UploadStream(context.Background(), containerName, blobName, resp.Body, uploadOptions)
	if err != nil {
		s.logger.Error("failed to upload blob", slog.String("error", err.Error()))
		return err
	}

	requestID := ""
	if stream.RequestID != nil {
		requestID = *stream.RequestID
	}
	s.logger.Info("uploaded blob",
		slog.String("blob_name", blobName),
		slog.String("container_name", containerName),
		slog.String("request_id", requestID),
	)
	return nil
}

func (s *service) DownloadBlobToStream(containerName, blobName string) (*blob.DownloadStreamResponse, error) {
	client, err := azblob.NewClientFromConnectionString(s.connectionString, nil)
	if err != nil {
		return nil, err
	}

	stream, err := client.DownloadStream(context.Background(), containerName, blobName, nil)
	if err != nil {
		return nil, err
	}
	return &stream, nil
}
