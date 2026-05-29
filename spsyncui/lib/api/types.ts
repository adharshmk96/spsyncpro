/**
 * TypeScript mirrors of the spsyncapi DTOs. Field names match the API's JSON
 * tags exactly (snake_case) so responses can be used without remapping.
 */

export type ApiErrorBody = { error: string };

export type SuccessBody = { success: boolean };

export type AuthTokenBody = { token: string };

export type Pagination = {
  page: number;
  limit: number;
  total: number;
};

/* ----------------------------------- Auth ---------------------------------- */

export type Member = {
  id: string;
  email: string;
  created_at: string;
};

export type MeResponse = { user: Member };

/* ------------------------------ Organizations ------------------------------ */

export type Organization = {
  id: string;
  name: string;
  tenant_id: string;
  client_id: string;
  created_at: string;
  updated_at: string;
};

/** Request body for create/update. `tenant_secret` is write-only. */
export type OrganizationInput = {
  name: string;
  tenant_id: string;
  client_id: string;
  tenant_secret?: string;
};

export type OrganizationResponse = { organization: Organization };
export type OrganizationsResponse = { organizations: Organization[] };

/* ------------------------------ Bucket Stores ------------------------------ */

export type BucketType = "s3" | "azure";

export type BucketStore = {
  id: string;
  bucket_name: string;
  bucket_type: BucketType;
  created_at: string;
  updated_at: string;
};

export type S3Config = {
  server: string;
  access_key: string;
  secret_key: string;
};

export type AzureConfig = {
  connection_string: string;
};

/** Request body for create/update. `config` is write-only. */
export type BucketStoreInput = {
  bucket_name: string;
  bucket_type: BucketType;
  config?: S3Config | AzureConfig;
};

export type BucketStoreResponse = { bucket_store: BucketStore };
export type BucketStoresResponse = { bucket_stores: BucketStore[] };

/* ------------------------------- Backup Jobs ------------------------------- */

export type Schedule = {
  type: "one_time" | "recurring";
  interval?: number | null;
  cron?: string | null;
  one_time?: string | null;
};

/** Schedule payload for create/update backup jobs. */
export type ScheduleInput = {
  type?: "one_time" | "recurring";
  interval?: number | null;
  one_time?: string | null;
};

/** Filters as returned by the API (document_libraries is comma-separated). */
export type BackupFiltersRead = {
  document_libraries?: string;
  min_file_size?: number | null;
  max_file_size?: number | null;
  created_after?: string | null;
  updated_after?: string | null;
  created_before?: string | null;
  updated_before?: string | null;
};

/** Filters as sent to the API (document_libraries is an array). */
export type BackupFiltersInput = {
  document_libraries?: string[];
  min_file_size?: number | null;
  max_file_size?: number | null;
  created_after?: string | null;
  updated_after?: string | null;
  created_before?: string | null;
  updated_before?: string | null;
};

export type BackupJobConfigRead = {
  organization: string;
  bucket_store: string;
  share_point_site: string;
  filters: BackupFiltersRead;
};

export type BackupJobConfigInput = {
  organization: string;
  bucket_store: string;
  share_point_site: string;
  filters: BackupFiltersInput;
};

export type BackupJob = {
  id: string;
  last_run?: string | null;
  next_run?: string | null;
  start_at?: string | null;
  end_at?: string | null;
  active: boolean;
  schedule: Schedule;
  job_config: BackupJobConfigRead;
  created_at: string;
  updated_at: string;
};

export type BackupJobInput = {
  start_at?: string | null;
  end_at?: string | null;
  active: boolean;
  schedule: ScheduleInput;
  job_config: BackupJobConfigInput;
};

export type BackupJobResponse = { backup_job: BackupJob };
export type BackupJobsResponse = { backup_jobs: BackupJob[] };

/* ------------------------------ Restore Jobs ------------------------------- */

export type RestoreJobConfig = {
  organization: string;
  bucket_store: string;
  share_point_site: string;
};

export type RestoreJob = {
  id: string;
  start_at?: string | null;
  last_run?: string | null;
  active: boolean;
  job_config: RestoreJobConfig;
  created_at: string;
  updated_at: string;
};

export type RestoreJobInput = {
  start_at?: string | null;
  active?: boolean;
  job_config: RestoreJobConfig;
};

export type RestoreJobResponse = { restore_job: RestoreJob };
export type RestoreJobsResponse = { restore_jobs: RestoreJob[] };

/* --------------------------------- Runs ------------------------------------ */

/** Backup and restore runs share the same shape. */
export type RunDetails = {
  id: string;
  job_id: string;
  start_at?: string | null;
  end_at?: string | null;
};

export type FileTransfer = {
  file_path: string;
  start_at?: string | null;
  end_at?: string | null;
};

export type BackupRunsResponse = {
  backup_runs: RunDetails[];
  pagination: Pagination;
};

export type BackupRunResponse = {
  backup_run: RunDetails;
  file_transfers: FileTransfer[];
  pagination: Pagination;
};

export type BackupRunStartResponse = { backup_run: RunDetails };

export type RestoreRunsResponse = {
  restore_runs: RunDetails[];
  pagination: Pagination;
};

export type RestoreRunResponse = {
  restore_run: RunDetails;
  file_transfers: FileTransfer[];
  pagination: Pagination;
};

export type RestoreRunStartResponse = { restore_run: RunDetails };

/** Run status is derived client-side from start_at / end_at (no API field). */
export type DerivedRunStatus = "pending" | "running" | "completed";
