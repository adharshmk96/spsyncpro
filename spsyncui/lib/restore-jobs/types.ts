export type RestoreJobLogStatus = "running" | "success" | "failed" | "cancelled";

export type RestoreBucketType = "s3" | "azure_blob";

export type RestoreJobFilter = {
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

/** Source bucket to restore files from (bucket → SharePoint). */
export type RestoreConfig =
  | { bucketType: "s3"; bucketConfig: S3BucketConfig }
  | { bucketType: "azure_blob"; bucketConfig: AzureBlobBucketConfig };

export type RestoreRunMode = "immediate" | "scheduled";

export type RestoreJob = {
  id: string;
  sharepointSiteUrl: string;
  filter: RestoreJobFilter;
  runMode: RestoreRunMode;
  scheduledAt: string | null;
  lastRunAt: string | null;
  createdAt: string;
  updatedAt: string;
  restoreConfig: RestoreConfig;
};

export type RestoreJobLogTransferStatus =
  | "pending"
  | "in_progress"
  | "completed"
  | "failed"
  | "skipped";

export type RestoreJobLog = {
  id: string;
  jobId: string;
  createdAt: string;
  endedAt: string | null;
  status: RestoreJobLogStatus;
};

export type RestoreJobLogFileEntry = {
  id: string;
  logId: string;
  filename: string;
  size: number;
  transferStatus: RestoreJobLogTransferStatus;
  startAt: string | null;
  endAt: string | null;
};
