import type { RestoreJob, RestoreJobLog, RestoreJobLogFileEntry } from "@/lib/restore-jobs/types";

export const PLACEHOLDER_RESTORE_JOBS: RestoreJob[] = [
  {
    id: "restore-001",
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
    runMode: "immediate",
    scheduledAt: null,
    lastRunAt: "2026-05-22T10:00:00.000Z",
    createdAt: "2026-05-22T09:45:00.000Z",
    updatedAt: "2026-05-22T10:18:00.000Z",
    restoreConfig: {
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
    id: "restore-002",
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
    runMode: "scheduled",
    scheduledAt: "2026-05-25T08:00:00.000Z",
    lastRunAt: null,
    createdAt: "2026-05-23T14:00:00.000Z",
    updatedAt: "2026-05-23T14:00:00.000Z",
    restoreConfig: {
      bucketType: "azure_blob",
      bucketConfig: {
        connectionString: "DefaultEndpointsProtocol=https;AccountName=contosohr;...",
        containerName: "hr-backups",
      },
    },
  },
  {
    id: "restore-003",
    sharepointSiteUrl: "https://contoso.sharepoint.com/sites/engineering",
    filter: {
      documentLibraryList: ["Design Docs", "Release Notes"],
      minFileSize: 0,
      maxFileSize: null,
      createdBefore: null,
      createdAfter: null,
      modifiedBefore: null,
      modifiedAfter: null,
    },
    runMode: "immediate",
    scheduledAt: null,
    lastRunAt: "2026-05-20T16:30:00.000Z",
    createdAt: "2026-05-20T16:00:00.000Z",
    updatedAt: "2026-05-20T16:45:00.000Z",
    restoreConfig: {
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

export const PLACEHOLDER_RESTORE_JOB_LOGS: RestoreJobLog[] = [
  {
    id: "rlog-001",
    jobId: "restore-001",
    createdAt: "2026-05-22T10:00:00.000Z",
    endedAt: "2026-05-22T10:17:45.000Z",
    status: "success",
  },
  {
    id: "rlog-002",
    jobId: "restore-003",
    createdAt: "2026-05-20T16:30:00.000Z",
    endedAt: "2026-05-20T16:44:10.000Z",
    status: "success",
  },
  {
    id: "rlog-003",
    jobId: "restore-003",
    createdAt: "2026-05-19T11:00:00.000Z",
    endedAt: "2026-05-19T11:02:30.000Z",
    status: "failed",
  },
];

export const PLACEHOLDER_RESTORE_JOB_LOG_FILES: RestoreJobLogFileEntry[] = [
  {
    id: "rfile-001",
    logId: "rlog-001",
    filename: "Contracts/2026/vendor-agreement.pdf",
    size: 2457600,
    transferStatus: "completed",
    startAt: "2026-05-22T10:00:12.000Z",
    endAt: "2026-05-22T10:01:40.000Z",
  },
  {
    id: "rfile-002",
    logId: "rlog-001",
    filename: "Policies/remote-work-policy.docx",
    size: 524288,
    transferStatus: "completed",
    startAt: "2026-05-22T10:01:41.000Z",
    endAt: "2026-05-22T10:02:05.000Z",
  },
  {
    id: "rfile-003",
    logId: "rlog-002",
    filename: "Design Docs/system-overview.vsdx",
    size: 5242880,
    transferStatus: "completed",
    startAt: "2026-05-20T16:30:15.000Z",
    endAt: "2026-05-20T16:43:50.000Z",
  },
  {
    id: "rfile-004",
    logId: "rlog-003",
    filename: "Release Notes/v2.3.0.md",
    size: 8192,
    transferStatus: "failed",
    startAt: "2026-05-19T11:00:08.000Z",
    endAt: "2026-05-19T11:02:28.000Z",
  },
];

export function getPlaceholderRestoreJobs(): RestoreJob[] {
  return PLACEHOLDER_RESTORE_JOBS;
}

export function getPlaceholderRestoreJobById(jobId: string): RestoreJob | undefined {
  return PLACEHOLDER_RESTORE_JOBS.find((job) => job.id === jobId);
}

export function getPlaceholderRestoreJobLogs(jobId: string): RestoreJobLog[] {
  return PLACEHOLDER_RESTORE_JOB_LOGS.filter((log) => log.jobId === jobId).sort(
    (left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime()
  );
}

export function getPlaceholderRestoreJobLogById(logId: string): RestoreJobLog | undefined {
  return PLACEHOLDER_RESTORE_JOB_LOGS.find((log) => log.id === logId);
}

export function getPlaceholderRestoreJobLogFiles(logId: string): RestoreJobLogFileEntry[] {
  return PLACEHOLDER_RESTORE_JOB_LOG_FILES.filter((entry) => entry.logId === logId);
}
