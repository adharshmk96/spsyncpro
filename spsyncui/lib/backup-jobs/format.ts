export function formatDateTime(value: string | null): string {
  if (!value) {
    return "-";
  }

  return new Date(value).toLocaleString();
}

export function formatFileSize(bytes: number | null): string {
  if (bytes === null) {
    return "No limit";
  }

  if (bytes === 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }

  return `${size.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

import type { BackupConfig, BackupJobFilter } from "@/lib/backup-jobs/types";

export function formatJobType(jobType: "onetime" | "recurring"): string {
  return jobType === "onetime" ? "One-time" : "Recurring";
}

export function formatFilterSummary(filter: BackupJobFilter): string {
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

export function formatBackupConfigSummary(config: BackupConfig): string {
  if (config.bucketType === "s3") {
    return `S3 · ${config.bucketConfig.region} / ${config.bucketConfig.bucketName}`;
  }

  return `Azure Blob · ${config.bucketConfig.containerName}`;
}

export function formatLogStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

export function formatTransferStatus(status: string): string {
  return status
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
