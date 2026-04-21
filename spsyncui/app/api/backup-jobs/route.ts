import { desc, eq } from "drizzle-orm";
import { headers } from "next/headers";
import { NextResponse } from "next/server";

import { auth } from "@/lib/auth";
import { computeInitialNextRunAt, normalizeInputDateToUtcIso } from "@/lib/backup-jobs/scheduler";
import { db } from "@/lib/db";
import { backupjobs } from "@/lib/db/schema";

type StorageType = "AZURE_BLOB" | "AWS_S3";
type RunMode = "IMMEDIATE" | "ONE_TIME_AT" | "RECURRING";
type Recurrence = "DAILY" | "WEEKLY" | "MONTHLY";

type BackupStorageConfig = {
  azureBlobConfig?: {
    connectionString: string;
    containerName: string;
  };
  awsS3Config?: {
    region: string;
    bucketName: string;
    accessKeyID: string;
    secretAccessKey: string;
  };
};

type BackupFilterConfig = {
  minFileSize: number;
  maxFileSize: number;
  createdBefore?: string;
  createdAfter?: string;
  modifiedBefore?: string;
  modifiedAfter?: string;
};

type BackupJobCreatePayload = {
  siteUrl: string;
  documentLibraryList: string[];
  storageType: StorageType;
  storageConfig: BackupStorageConfig;
  filterConfig: BackupFilterConfig;
  runMode: RunMode;
  startAt?: string;
  recurrence?: Recurrence;
};

const STORAGE_TYPES: StorageType[] = ["AZURE_BLOB", "AWS_S3"];
const RUN_MODES: RunMode[] = ["IMMEDIATE", "ONE_TIME_AT", "RECURRING"];
const RECURRENCES: Recurrence[] = ["DAILY", "WEEKLY", "MONTHLY"];

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function parseIsoDate(value: unknown, fieldName: string): Date {
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`${fieldName} is required.`);
  }

  const parsedDate = new Date(value);
  if (Number.isNaN(parsedDate.getTime())) {
    throw new Error(`${fieldName} must be a valid datetime.`);
  }

  return parsedDate;
}

function normalizeOptionalIsoDate(value: unknown, fieldName: string): string | undefined {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }

  const parsedDate = parseIsoDate(value, fieldName);
  return parsedDate.toISOString();
}

function validateStorageConfig(
  storageType: StorageType,
  storageConfig: unknown
): BackupStorageConfig {
  if (!isRecord(storageConfig)) {
    throw new Error("storageConfig is required.");
  }

  if (storageType === "AZURE_BLOB") {
    const azureBlobConfig = storageConfig.azureBlobConfig;
    if (!isRecord(azureBlobConfig)) {
      throw new Error("storageConfig.azureBlobConfig is required.");
    }

    if (
      typeof azureBlobConfig.connectionString !== "string" ||
      azureBlobConfig.connectionString.trim().length === 0
    ) {
      throw new Error("storageConfig.azureBlobConfig.connectionString is required.");
    }

    if (
      typeof azureBlobConfig.containerName !== "string" ||
      azureBlobConfig.containerName.trim().length === 0
    ) {
      throw new Error("storageConfig.azureBlobConfig.containerName is required.");
    }

    return {
      azureBlobConfig: {
        connectionString: azureBlobConfig.connectionString.trim(),
        containerName: azureBlobConfig.containerName.trim(),
      },
    };
  }

  const awsS3Config = storageConfig.awsS3Config;
  if (!isRecord(awsS3Config)) {
    throw new Error("storageConfig.awsS3Config is required.");
  }

  if (typeof awsS3Config.region !== "string" || awsS3Config.region.trim().length === 0) {
    throw new Error("storageConfig.awsS3Config.region is required.");
  }

  if (
    typeof awsS3Config.bucketName !== "string" ||
    awsS3Config.bucketName.trim().length === 0
  ) {
    throw new Error("storageConfig.awsS3Config.bucketName is required.");
  }

  if (
    typeof awsS3Config.accessKeyID !== "string" ||
    awsS3Config.accessKeyID.trim().length === 0
  ) {
    throw new Error("storageConfig.awsS3Config.accessKeyID is required.");
  }

  if (
    typeof awsS3Config.secretAccessKey !== "string" ||
    awsS3Config.secretAccessKey.trim().length === 0
  ) {
    throw new Error("storageConfig.awsS3Config.secretAccessKey is required.");
  }

  return {
    awsS3Config: {
      region: awsS3Config.region.trim(),
      bucketName: awsS3Config.bucketName.trim(),
      accessKeyID: awsS3Config.accessKeyID.trim(),
      secretAccessKey: awsS3Config.secretAccessKey.trim(),
    },
  };
}

function validateFilterConfig(filterConfig: unknown): BackupFilterConfig {
  if (!isRecord(filterConfig)) {
    throw new Error("filterConfig is required.");
  }

  const minFileSize = Number(filterConfig.minFileSize);
  const maxFileSize = Number(filterConfig.maxFileSize);

  if (!Number.isFinite(minFileSize) || minFileSize < 0) {
    throw new Error("filterConfig.minFileSize must be a non-negative number.");
  }

  if (!Number.isFinite(maxFileSize) || maxFileSize < 0) {
    throw new Error("filterConfig.maxFileSize must be a non-negative number.");
  }

  return {
    minFileSize,
    maxFileSize,
    createdBefore: normalizeOptionalIsoDate(filterConfig.createdBefore, "filterConfig.createdBefore"),
    createdAfter: normalizeOptionalIsoDate(filterConfig.createdAfter, "filterConfig.createdAfter"),
    modifiedBefore: normalizeOptionalIsoDate(
      filterConfig.modifiedBefore,
      "filterConfig.modifiedBefore"
    ),
    modifiedAfter: normalizeOptionalIsoDate(filterConfig.modifiedAfter, "filterConfig.modifiedAfter"),
  };
}

function validateCreatePayload(payload: unknown): BackupJobCreatePayload {
  if (!isRecord(payload)) {
    throw new Error("Invalid request payload.");
  }

  if (typeof payload.siteUrl !== "string" || payload.siteUrl.trim().length === 0) {
    throw new Error("siteUrl is required.");
  }

  if (!Array.isArray(payload.documentLibraryList)) {
    throw new Error("documentLibraryList is required.");
  }

  const normalizedDocumentLibraryList = payload.documentLibraryList
    .filter((entry): entry is string => typeof entry === "string")
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);

  if (normalizedDocumentLibraryList.length === 0) {
    throw new Error("documentLibraryList must contain at least one value.");
  }

  if (typeof payload.storageType !== "string" || !STORAGE_TYPES.includes(payload.storageType as StorageType)) {
    throw new Error("storageType is invalid.");
  }

  const storageType = payload.storageType as StorageType;
  const storageConfig = validateStorageConfig(storageType, payload.storageConfig);
  const filterConfig = validateFilterConfig(payload.filterConfig);

  if (typeof payload.runMode !== "string" || !RUN_MODES.includes(payload.runMode as RunMode)) {
    throw new Error("runMode is invalid.");
  }

  const runMode = payload.runMode as RunMode;
  let startAt: string | undefined;
  let recurrence: Recurrence | undefined;

  if (runMode === "ONE_TIME_AT") {
    startAt = normalizeInputDateToUtcIso(String(payload.startAt), "startAt");
  }

  if (runMode === "RECURRING") {
    startAt = normalizeInputDateToUtcIso(String(payload.startAt), "startAt");
    if (
      typeof payload.recurrence !== "string" ||
      !RECURRENCES.includes(payload.recurrence as Recurrence)
    ) {
      throw new Error("recurrence is required for recurring jobs.");
    }
    recurrence = payload.recurrence as Recurrence;
  }

  return {
    siteUrl: payload.siteUrl.trim(),
    documentLibraryList: normalizedDocumentLibraryList,
    storageType,
    storageConfig,
    filterConfig,
    runMode,
    startAt,
    recurrence,
  };
}

async function requireSession() {
  return auth.api.getSession({
    headers: await headers(),
  });
}

export async function GET() {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized backup jobs list attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const jobs = await db.query.backupjobs.findMany({
      orderBy: [desc(backupjobs.createdAt)],
    });

    const jobList = jobs.map((job) => ({
      ...job,
      // Return UTC fields only; UI is responsible for local-time rendering.
      nextRunAt: job.nextRunAt,
      lastRunAt: job.lastRunAt,
      lastRunStatus: job.lastRunStatus,
    }));

    return NextResponse.json({ jobs: jobList }, { status: 200 });
  } catch (error) {
    console.error("Failed to list backup jobs.", error);
    return NextResponse.json({ error: "Failed to list backup jobs." }, { status: 500 });
  }
}

export async function POST(request: Request) {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized backup job create attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const body = await request.json();
    const payload = validateCreatePayload(body);
    const now = new Date();
    const id = crypto.randomUUID();
    const nextRunAt = computeInitialNextRunAt(
      payload.runMode,
      payload.startAt,
      payload.recurrence,
      now
    );

    await db.insert(backupjobs).values({
      id,
      siteUrl: payload.siteUrl,
      documentLibraryList: payload.documentLibraryList,
      storageType: payload.storageType,
      storageConfig: payload.storageConfig,
      filterConfig: payload.filterConfig,
      runMode: payload.runMode,
      startAt: payload.startAt ? new Date(payload.startAt) : null,
      recurrence: payload.recurrence ?? null,
      status: "ACTIVE",
      nextRunAt,
      lastRunAt: null,
      lastRunStatus: null,
      leaseOwner: null,
      leaseUntil: null,
      runnerMetadata: { dispatched: false },
      createdAt: now,
      updatedAt: now,
    });

    const createdJob = await db.query.backupjobs.findFirst({
      where: eq(backupjobs.id, id),
    });

    console.info("Backup job created with UTC schedule.", {
      id,
      nextRunAt: nextRunAt?.toISOString() ?? null,
    });
    return NextResponse.json({ job: createdJob }, { status: 201 });
  } catch (error) {
    if (error instanceof SyntaxError) {
      return NextResponse.json({ error: "Invalid JSON body." }, { status: 400 });
    }

    if (error instanceof Error) {
      console.warn("Backup job create rejected.", error.message);
      return NextResponse.json({ error: error.message }, { status: 400 });
    }

    console.error("Failed to create backup job.", error);
    return NextResponse.json({ error: "Failed to create backup job." }, { status: 500 });
  }
}
