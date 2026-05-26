import {
  formatDateTime,
  formatFileSize,
  formatLogStatus,
  formatTransferStatus,
} from "@/lib/backup-jobs/format";
import type { RestoreConfig, RestoreJobFilter, RestoreRunMode } from "@/lib/restore-jobs/types";

export { formatDateTime, formatFileSize, formatLogStatus, formatTransferStatus };

export function formatRunMode(runMode: RestoreRunMode): string {
  return runMode === "immediate" ? "Run immediately" : "Scheduled one-time";
}

export function formatFilterSummary(filter: RestoreJobFilter): string {
  const parts = [`Libraries: ${filter.documentLibraryList.join(", ")}`];

  parts.push(
    `Size: ${formatFileSize(filter.minFileSize)} – ${formatFileSize(filter.maxFileSize)}`
  );

  if (filter.createdAfter || filter.createdBefore) {
    parts.push(
      `Created: ${formatDateTime(filter.createdAfter)} → ${formatDateTime(filter.createdBefore)}`
    );
  }

  if (filter.modifiedAfter || filter.modifiedBefore) {
    parts.push(
      `Modified: ${formatDateTime(filter.modifiedAfter)} → ${formatDateTime(filter.modifiedBefore)}`
    );
  }

  return parts.join(" · ");
}

export function formatRestoreConfigSummary(config: RestoreConfig): string {
  if (config.bucketType === "s3") {
    return `S3 · ${config.bucketConfig.region} / ${config.bucketConfig.bucketName}`;
  }

  return `Azure Blob · ${config.bucketConfig.containerName}`;
}
