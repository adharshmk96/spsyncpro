export type BackupJobType = "onetime" | "recurring";

export type BackupJobLogStatus = "running" | "success" | "failed" | "cancelled";

export type BackupBucketType = "s3" | "azure_blob";

export type BackupJobFilter = {
  documentLibraryList: string[];
  minFileSize: number | null;
  maxFileSize: number | null;
  createdBefore: string | null;
  createdAfter: string | null;
  modifiedBefore: string | null;
  modifiedAfter: string | null;
};

export type S3BucketConfig = {
  region: string;
  bucketName: string;
  accessKeyId: string;
  secretAccessKey: string;
};

export type AzureBlobBucketConfig = {
  connectionString: string;
  containerName: string;
};

export type BackupConfig =
  | { bucketType: "s3"; bucketConfig: S3BucketConfig }
  | { bucketType: "azure_blob"; bucketConfig: AzureBlobBucketConfig };

export type BackupJob = {
  id: string;
  sharepointSiteUrl: string;
  filter: BackupJobFilter;
  jobType: BackupJobType;
  lastRunAt: string | null;
  nextRunAt: string | null;
  createdAt: string;
  updatedAt: string;
  backupConfig: BackupConfig;
};

export type BackupJobLogTransferStatus =
  | "pending"
  | "in_progress"
  | "completed"
  | "failed"
  | "skipped";

export type BackupJobLog = {
  id: string;
  jobId: string;
  createdAt: string;
  endedAt: string | null;
  status: BackupJobLogStatus;
};

export type BackupJobLogFileEntry = {
  id: string;
  logId: string;
  filename: string;
  size: number;
  transferStatus: BackupJobLogTransferStatus;
  startAt: string | null;
  endAt: string | null;
};
