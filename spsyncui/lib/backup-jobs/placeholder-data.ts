import type { BackupJob, BackupJobLog, BackupJobLogFileEntry } from "@/lib/backup-jobs/types";

export const PLACEHOLDER_BACKUP_JOBS: BackupJob[] = [
  {
    id: "job-001",
    sharepointSiteUrl: "https://contoso.sharepoint.com/sites/legal",
    filter: {
      documentLibraryList: ["Contracts", "Policies"],
      minFileSize: 1024,
      maxFileSize: 52428800,
      createdBefore: null,
      createdAfter: "2024-01-01T00:00:00.000Z",
      modifiedBefore: null,
      modifiedAfter: null,
    },
    jobType: "recurring",
    lastRunAt: "2026-05-23T02:00:00.000Z",
    nextRunAt: "2026-05-24T02:00:00.000Z",
    createdAt: "2026-03-15T10:30:00.000Z",
    updatedAt: "2026-05-23T02:15:00.000Z",
    backupConfig: {
      bucketType: "s3",
      bucketConfig: {
        region: "us-east-1",
        bucketName: "contoso-legal-backups",
        accessKeyId: "AKIA************",
        secretAccessKey: "********",
      },
    },
  },
  {
    id: "job-002",
    sharepointSiteUrl: "https://contoso.sharepoint.com/sites/hr",
    filter: {
      documentLibraryList: ["Employee Records"],
      minFileSize: null,
      maxFileSize: 10485760,
      createdBefore: "2026-01-01T00:00:00.000Z",
      createdAfter: null,
      modifiedBefore: null,
      modifiedAfter: "2025-06-01T00:00:00.000Z",
    },
    jobType: "onetime",
    lastRunAt: "2026-05-20T14:30:00.000Z",
    nextRunAt: null,
    createdAt: "2026-05-18T09:00:00.000Z",
    updatedAt: "2026-05-20T14:45:00.000Z",
    backupConfig: {
      bucketType: "azure_blob",
      bucketConfig: {
        connectionString: "DefaultEndpointsProtocol=https;AccountName=contosohr;...",
        containerName: "hr-backups",
      },
    },
  },
  {
    id: "job-003",
    sharepointSiteUrl: "https://contoso.sharepoint.com/sites/engineering",
    filter: {
      documentLibraryList: ["Design Docs", "Release Notes", "Architecture"],
      minFileSize: 0,
      maxFileSize: null,
      createdBefore: null,
      createdAfter: null,
      modifiedBefore: "2026-05-24T00:00:00.000Z",
      modifiedAfter: "2026-04-01T00:00:00.000Z",
    },
    jobType: "recurring",
    lastRunAt: null,
    nextRunAt: "2026-05-25T06:00:00.000Z",
    createdAt: "2026-05-22T16:20:00.000Z",
    updatedAt: "2026-05-22T16:20:00.000Z",
    backupConfig: {
      bucketType: "s3",
      bucketConfig: {
        region: "eu-west-1",
        bucketName: "contoso-eng-backups",
        accessKeyId: "AKIA************",
        secretAccessKey: "********",
      },
    },
  },
];

export const PLACEHOLDER_BACKUP_JOB_LOGS: BackupJobLog[] = [
  {
    id: "log-001",
    jobId: "job-001",
    createdAt: "2026-05-23T02:00:00.000Z",
    endedAt: "2026-05-23T02:14:32.000Z",
    status: "success",
  },
  {
    id: "log-002",
    jobId: "job-001",
    createdAt: "2026-05-22T02:00:00.000Z",
    endedAt: "2026-05-22T02:12:18.000Z",
    status: "success",
  },
  {
    id: "log-003",
    jobId: "job-002",
    createdAt: "2026-05-20T14:30:00.000Z",
    endedAt: "2026-05-20T14:44:55.000Z",
    status: "success",
  },
  {
    id: "log-004",
    jobId: "job-001",
    createdAt: "2026-05-21T02:00:00.000Z",
    endedAt: "2026-05-21T02:05:10.000Z",
    status: "failed",
  },
  {
    id: "log-005",
    jobId: "job-003",
    createdAt: "2026-05-24T06:00:00.000Z",
    endedAt: null,
    status: "running",
  },
];

export const PLACEHOLDER_BACKUP_JOB_LOG_FILES: BackupJobLogFileEntry[] = [
  {
    id: "file-001",
    logId: "log-001",
    filename: "Contracts/2026/vendor-agreement.pdf",
    size: 2457600,
    transferStatus: "completed",
    startAt: "2026-05-23T02:00:12.000Z",
    endAt: "2026-05-23T02:01:45.000Z",
  },
  {
    id: "file-002",
    logId: "log-001",
    filename: "Policies/remote-work-policy.docx",
    size: 524288,
    transferStatus: "completed",
    startAt: "2026-05-23T02:01:46.000Z",
    endAt: "2026-05-23T02:02:10.000Z",
  },
  {
    id: "file-003",
    logId: "log-001",
    filename: "Contracts/archive/legacy-terms.pdf",
    size: 8912896,
    transferStatus: "completed",
    startAt: "2026-05-23T02:02:11.000Z",
    endAt: "2026-05-23T02:14:20.000Z",
  },
  {
    id: "file-004",
    logId: "log-002",
    filename: "Policies/code-of-conduct.pdf",
    size: 1048576,
    transferStatus: "completed",
    startAt: "2026-05-22T02:00:08.000Z",
    endAt: "2026-05-22T02:00:55.000Z",
  },
  {
    id: "file-005",
    logId: "log-002",
    filename: "Contracts/nda-template.docx",
    size: 327680,
    transferStatus: "completed",
    startAt: "2026-05-22T02:00:56.000Z",
    endAt: "2026-05-22T02:12:10.000Z",
  },
  {
    id: "file-006",
    logId: "log-003",
    filename: "Employee Records/onboarding-checklist.xlsx",
    size: 204800,
    transferStatus: "completed",
    startAt: "2026-05-20T14:30:15.000Z",
    endAt: "2026-05-20T14:31:02.000Z",
  },
  {
    id: "file-007",
    logId: "log-003",
    filename: "Employee Records/benefits-summary.pdf",
    size: 1572864,
    transferStatus: "completed",
    startAt: "2026-05-20T14:31:03.000Z",
    endAt: "2026-05-20T14:44:40.000Z",
  },
  {
    id: "file-008",
    logId: "log-004",
    filename: "Contracts/2026/q1-review.pdf",
    size: 4194304,
    transferStatus: "failed",
    startAt: "2026-05-21T02:00:10.000Z",
    endAt: "2026-05-21T02:05:05.000Z",
  },
  {
    id: "file-009",
    logId: "log-004",
    filename: "Policies/expense-policy.docx",
    size: 262144,
    transferStatus: "skipped",
    startAt: null,
    endAt: null,
  },
  {
    id: "file-010",
    logId: "log-005",
    filename: "Design Docs/system-overview.vsdx",
    size: 5242880,
    transferStatus: "in_progress",
    startAt: "2026-05-24T06:00:18.000Z",
    endAt: null,
  },
  {
    id: "file-011",
    logId: "log-005",
    filename: "Release Notes/v2.4.0.md",
    size: 12288,
    transferStatus: "completed",
    startAt: "2026-05-24T06:00:05.000Z",
    endAt: "2026-05-24T06:00:12.000Z",
  },
  {
    id: "file-012",
    logId: "log-005",
    filename: "Architecture/network-diagram.png",
    size: 1048576,
    transferStatus: "pending",
    startAt: null,
    endAt: null,
  },
];

export function getPlaceholderBackupJobs(): BackupJob[] {
  return PLACEHOLDER_BACKUP_JOBS;
}

export function getPlaceholderBackupJobById(jobId: string): BackupJob | undefined {
  return PLACEHOLDER_BACKUP_JOBS.find((job) => job.id === jobId);
}

export function getPlaceholderBackupJobLogs(jobId: string): BackupJobLog[] {
  return PLACEHOLDER_BACKUP_JOB_LOGS.filter((log) => log.jobId === jobId).sort(
    (left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()
  );
}

export function getPlaceholderBackupJobLogById(logId: string): BackupJobLog | undefined {
  return PLACEHOLDER_BACKUP_JOB_LOGS.find((log) => log.id === logId);
}

export function getPlaceholderBackupJobLogFiles(logId: string): BackupJobLogFileEntry[] {
  return PLACEHOLDER_BACKUP_JOB_LOG_FILES.filter((entry) => entry.logId === logId);
}
