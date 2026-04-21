export type BackupRunMode = "IMMEDIATE" | "ONE_TIME_AT" | "RECURRING";
export type BackupRecurrence = "DAILY" | "WEEKLY" | "MONTHLY";

function requireValidDate(date: Date, field: string): Date {
  if (Number.isNaN(date.getTime())) {
    throw new Error(`${field} must be a valid UTC datetime.`);
  }

  return date;
}

export function normalizeInputDateToUtcIso(value: string, field: string): string {
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`${field} is required.`);
  }

  const parsed = new Date(value);
  return requireValidDate(parsed, field).toISOString();
}

export function computeInitialNextRunAt(
  runMode: BackupRunMode,
  startAtIso: string | undefined,
  recurrence: BackupRecurrence | undefined,
  nowUtc: Date
): Date | null {
  requireValidDate(nowUtc, "nowUtc");

  if (runMode === "IMMEDIATE") {
    return new Date(nowUtc);
  }

  if (!startAtIso) {
    throw new Error("startAt is required for non-immediate schedules.");
  }

  const startAt = requireValidDate(new Date(startAtIso), "startAt");

  if (runMode === "ONE_TIME_AT") {
    return startAt;
  }

  if (!recurrence) {
    throw new Error("recurrence is required for recurring schedules.");
  }

  return startAt;
}
