import type { BucketType, DerivedRunStatus, RunDetails, Schedule } from "@/lib/api/types";

/** Renders an ISO timestamp as a localized string, or a dash when empty. */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

/** Human-readable byte size (limits stored as bytes by the API). */
export function formatFileSize(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined) {
    return "No limit";
  }
  if (bytes === 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

/**
 * Derives a run's status from its timestamps. The API exposes no status field:
 * a run is "running" once started, "completed" once ended, else "pending".
 */
export function deriveRunStatus(run: Pick<RunDetails, "start_at" | "end_at">): DerivedRunStatus {
  if (run.end_at) {
    return "completed";
  }
  if (run.start_at) {
    return "running";
  }
  return "pending";
}

/** Capitalizes a derived run status for display. */
export function formatRunStatus(status: DerivedRunStatus): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

/** Friendly label for a bucket store backend type. */
export function formatBucketType(type: BucketType): string {
  return type === "s3" ? "S3" : "Azure Blob";
}

/** Summarizes an API schedule (exactly one of interval / cron / one_time). */
export function formatSchedule(schedule: Schedule | null | undefined): string {
  if (!schedule) {
    return "-";
  }
  if (schedule.interval != null) {
    return `Every ${schedule.interval}s`;
  }
  if (schedule.cron) {
    return `Cron: ${schedule.cron}`;
  }
  if (schedule.one_time) {
    return `Once at ${formatDateTime(schedule.one_time)}`;
  }
  return "-";
}

/** Splits the API's comma-separated document_libraries string into a list. */
export function parseDocumentLibraries(value: string | null | undefined): string[] {
  if (!value) {
    return [];
  }
  return value
    .split(",")
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
}

/** Converts a `datetime-local` input value to an ISO string (or null). */
export function localDateTimeToIso(localDateTime: string): string | null {
  if (!localDateTime) {
    return null;
  }
  const date = new Date(localDateTime);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date.toISOString();
}

/** Converts an ISO string to a `datetime-local` input value (or empty). */
export function isoToLocalDateTime(value: string | null | undefined): string {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  // Adjust to local time for the input control, then trim seconds/zone.
  const offsetMs = date.getTimezoneOffset() * 60 * 1000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}
