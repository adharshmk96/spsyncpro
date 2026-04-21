import {
  boolean,
  index,
  integer,
  jsonb,
  pgEnum,
  pgTable,
  text,
  timestamp,
  uniqueIndex,
} from "drizzle-orm/pg-core";

export const user = pgTable("user", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  email: text("email").notNull().unique(),
  emailVerified: boolean("email_verified").notNull().default(false),
  image: text("image"),
  createdAt: timestamp("created_at").notNull(),
  updatedAt: timestamp("updated_at").notNull(),
});

export const session = pgTable("session", {
  id: text("id").primaryKey(),
  expiresAt: timestamp("expires_at").notNull(),
  token: text("token").notNull().unique(),
  createdAt: timestamp("created_at").notNull(),
  updatedAt: timestamp("updated_at").notNull(),
  ipAddress: text("ip_address"),
  userAgent: text("user_agent"),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
});

export const account = pgTable("account", {
  id: text("id").primaryKey(),
  accountId: text("account_id").notNull(),
  providerId: text("provider_id").notNull(),
  userId: text("user_id")
    .notNull()
    .references(() => user.id, { onDelete: "cascade" }),
  accessToken: text("access_token"),
  refreshToken: text("refresh_token"),
  idToken: text("id_token"),
  accessTokenExpiresAt: timestamp("access_token_expires_at"),
  refreshTokenExpiresAt: timestamp("refresh_token_expires_at"),
  scope: text("scope"),
  password: text("password"),
  createdAt: timestamp("created_at").notNull(),
  updatedAt: timestamp("updated_at").notNull(),
});

export const verification = pgTable("verification", {
  id: text("id").primaryKey(),
  identifier: text("identifier").notNull(),
  value: text("value").notNull(),
  expiresAt: timestamp("expires_at").notNull(),
  createdAt: timestamp("created_at"),
  updatedAt: timestamp("updated_at"),
});

export const organization = pgTable("organization", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  description: text("description").notNull(),
  tenantId: text("tenant_id").notNull(),
  clientId: text("client_id").notNull(),
  encryptedClientSecret: text("encrypted_client_secret"),
  createdAt: timestamp("created_at").notNull(),
  updatedAt: timestamp("updated_at").notNull(),
});

export const backupStorageTypeEnum = pgEnum("backup_storage_type", [
  "AZURE_BLOB",
  "AWS_S3",
]);

export const backupRunModeEnum = pgEnum("backup_run_mode", [
  "IMMEDIATE",
  "ONE_TIME_AT",
  "RECURRING",
]);

export const backupRecurrenceEnum = pgEnum("backup_recurrence", [
  "DAILY",
  "WEEKLY",
  "MONTHLY",
]);

export const backupJobStatusEnum = pgEnum("backup_job_status", [
  "ACTIVE",
  "PAUSED",
]);

export const backupRunStatusEnum = pgEnum("backup_run_status", [
  "QUEUED",
  "RUNNING",
  "SUCCESS",
  "FAILED",
  "CANCELLED",
]);

export const backupJobLastRunStatusEnum = pgEnum("backup_job_last_run_status", [
  "RUNNING",
  "SUCCESS",
  "FAILED",
  "CANCELLED",
]);

export const backupjobs = pgTable(
  "backupjobs",
  {
    id: text("id").primaryKey(),
    siteUrl: text("site_url").notNull(),
    documentLibraryList: text("document_library_list").array().notNull(),
    storageType: backupStorageTypeEnum("storage_type").notNull(),
    storageConfig: jsonb("storage_config").notNull(),
    filterConfig: jsonb("filter_config").notNull(),
    runMode: backupRunModeEnum("run_mode").notNull(),
    startAt: timestamp("start_at"),
    recurrence: backupRecurrenceEnum("recurrence"),
    status: backupJobStatusEnum("status").notNull().default("ACTIVE"),
    nextRunAt: timestamp("next_run_at"),
    lastRunAt: timestamp("last_run_at"),
    lastRunStatus: backupJobLastRunStatusEnum("last_run_status"),
    leaseOwner: text("lease_owner"),
    leaseUntil: timestamp("lease_until"),
    runnerMetadata: jsonb("runner_metadata"),
    createdAt: timestamp("created_at").notNull(),
    updatedAt: timestamp("updated_at").notNull(),
  },
  (table) => [index("backupjobs_next_run_at_idx").on(table.nextRunAt)]
);

export const backupJobRuns = pgTable(
  "backup_job_runs",
  {
    id: text("id").primaryKey(),
    jobId: text("job_id")
      .notNull()
      .references(() => backupjobs.id, { onDelete: "cascade" }),
    scheduledFor: timestamp("scheduled_for").notNull(),
    startedAt: timestamp("started_at"),
    finishedAt: timestamp("finished_at"),
    status: backupRunStatusEnum("status").notNull().default("QUEUED"),
    attempt: integer("attempt").notNull().default(1),
    runnerId: text("runner_id"),
    errorMessage: text("error_message"),
    errorDetails: jsonb("error_details"),
    resultSummary: jsonb("result_summary"),
    createdAt: timestamp("created_at").notNull(),
    updatedAt: timestamp("updated_at").notNull(),
  },
  (table) => [
    uniqueIndex("backup_job_runs_job_id_scheduled_for_uidx").on(
      table.jobId,
      table.scheduledFor
    ),
    index("backup_job_runs_status_scheduled_for_idx").on(table.status, table.scheduledFor),
  ]
);
