package temporal

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"spsyncapi/internal/bucketstore"
	"spsyncapi/internal/storage"
	"spsyncapi/pkg/azureblob"
	"spsyncapi/pkg/graphapi"
)

const chunkUploadThreshold = 4 * 1024 * 1024

var errUnsupportedBucketType = errors.New("bucket type s3 not supported for file transfer")

type jobContext struct {
	sharePointSite   string
	tenantID         string
	clientID         string
	tenantSecret     string
	containerName    string
	bucketType       string
	connectionString string
	documentLibs     []string
	backupJob        *storage.BackupJob
}

func (a *Activities) loadJobContext(jobID, memberID string, kind RunKind) (jobContext, error) {
	var orgID, bucketStoreID, sharePointSite string
	var backupJob *storage.BackupJob

	switch kind {
	case RunKindBackup:
		job, err := a.BackupJobRepo.FindActiveByID(jobID, memberID)
		if err != nil {
			return jobContext{}, fmt.Errorf("load job context: backup job: %w", err)
		}
		orgID = job.OrganizationID
		bucketStoreID = job.BucketStoreID
		sharePointSite = job.SharePointSite
		jobCopy := *job
		backupJob = &jobCopy
	case RunKindRestore:
		job, err := a.RestoreJobRepo.FindActiveByID(jobID, memberID)
		if err != nil {
			return jobContext{}, fmt.Errorf("load job context: restore job: %w", err)
		}
		orgID = job.OrganizationID
		bucketStoreID = job.BucketStoreID
		sharePointSite = job.SharePointSite
	default:
		return jobContext{}, fmt.Errorf("load job context: unknown kind %q", kind)
	}

	org, err := a.OrgRepo.FindActiveByID(orgID, memberID)
	if err != nil {
		return jobContext{}, fmt.Errorf("load job context: organization: %w", err)
	}

	tenantSecret, err := a.Encryptor.Decrypt(org.TenantSecretEncrypted)
	if err != nil {
		return jobContext{}, fmt.Errorf("load job context: decrypt tenant secret: %w", err)
	}

	bucket, err := a.BucketStoreRepo.FindActiveByID(bucketStoreID, memberID)
	if err != nil {
		return jobContext{}, fmt.Errorf("load job context: bucket store: %w", err)
	}

	bucketConfig, err := a.Encryptor.Decrypt(bucket.ConfigEncrypted)
	if err != nil {
		return jobContext{}, fmt.Errorf("load job context: decrypt bucket config: %w", err)
	}

	connectionString, err := azureConnectionString(bucket.BucketType, bucketConfig)
	if err != nil {
		return jobContext{}, fmt.Errorf("load job context: %w", err)
	}

	ctx := jobContext{
		sharePointSite:   sharePointSite,
		tenantID:         org.TenantID,
		clientID:         org.ClientID,
		tenantSecret:     tenantSecret,
		containerName:    bucket.BucketName,
		bucketType:       bucket.BucketType,
		connectionString: connectionString,
		backupJob:        backupJob,
	}
	if backupJob != nil {
		ctx.documentLibs = parseDocumentLibraries(backupJob.FilterDocumentLibraries)
	}
	return ctx, nil
}

func azureConnectionString(bucketType, configJSON string) (string, error) {
	if bucketType != storage.BucketTypeAzure {
		return "", errUnsupportedBucketType
	}
	var cfg bucketstore.AzureConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return "", fmt.Errorf("parse azure config: %w", err)
	}
	if strings.TrimSpace(cfg.ConnectionString) == "" {
		return "", fmt.Errorf("azure config requires connection_string")
	}
	return cfg.ConnectionString, nil
}

func parseDocumentLibraries(csv string) []string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func shouldSkipFile(item graphapi.DriveItem, job *storage.BackupJob) bool {
	if job == nil {
		return false
	}

	if job.FilterMinFileSize != nil && item.Size < *job.FilterMinFileSize {
		return true
	}
	if job.FilterMaxFileSize != nil && item.Size > *job.FilterMaxFileSize {
		return true
	}

	if created, err := time.Parse(time.RFC3339, item.CreatedDateTime); err == nil {
		if job.FilterCreatedBefore != nil && !created.Before(*job.FilterCreatedBefore) {
			return true
		}
		if job.FilterCreatedAfter != nil && !created.After(*job.FilterCreatedAfter) {
			return true
		}
	}

	if modified, err := time.Parse(time.RFC3339, item.LastModifiedDateTime); err == nil {
		if job.FilterUpdatedBefore != nil && !modified.Before(*job.FilterUpdatedBefore) {
			return true
		}
		if job.FilterUpdatedAfter != nil && !modified.After(*job.FilterUpdatedAfter) {
			return true
		}
	}

	return false
}

func driveAllowed(driveName string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, driveName)
}

func backupBlobPath(driveName, filePath string) string {
	return driveName + "/" + strings.TrimPrefix(filePath, "/")
}

func splitRestorePath(blobPath string) (documentLibrary, libraryPath string) {
	parts := strings.SplitN(blobPath, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func (a *Activities) buildGraphClient(jc jobContext) graphapi.Service {
	if a.GraphServiceBuilder != nil {
		return a.GraphServiceBuilder(jc)
	}
	return graphapi.NewService(graphapi.Config{
		Org: graphapi.OrgConfig{
			TenantID:     jc.tenantID,
			ClientID:     jc.clientID,
			ClientSecret: jc.tenantSecret,
		},
		Logger: a.Logger,
	})
}

func (a *Activities) buildAzureClient(jc jobContext) azureblob.Service {
	if a.AzureServiceBuilder != nil {
		return a.AzureServiceBuilder(jc)
	}
	return azureblob.NewService(azureblob.Config{
		ConnectionString: jc.connectionString,
		Logger:           a.Logger,
	})
}

func effectiveMaxConcurrentTransfers(value int) int {
	if value > 0 {
		return value
	}
	return defaultMaxConcurrentTransfers
}
