"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  formatDateTime,
  formatFileSize,
  formatLogStatus,
  formatRunMode,
} from "@/lib/restore-jobs/format";
import {
  getPlaceholderRestoreJobById,
  getPlaceholderRestoreJobLogs,
} from "@/lib/restore-jobs/placeholder-data";
import type { RestoreConfig, RestoreJob, RestoreJobFilter } from "@/lib/restore-jobs/types";

function ConfigRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[180px_1fr]">
      <span className="font-medium text-muted-foreground">{label}</span>
      <span className="break-all">{value}</span>
    </div>
  );
}

function FilterConfigSection({ filter }: { filter: RestoreJobFilter }) {
  return (
    <div className="space-y-2 text-sm">
      <ConfigRow label="Document libraries" value={filter.documentLibraryList.join(", ")} />
      <ConfigRow label="Min file size" value={formatFileSize(filter.minFileSize)} />
      <ConfigRow label="Max file size" value={formatFileSize(filter.maxFileSize)} />
      <ConfigRow label="Created after" value={formatDateTime(filter.createdAfter)} />
      <ConfigRow label="Created before" value={formatDateTime(filter.createdBefore)} />
      <ConfigRow label="Modified after" value={formatDateTime(filter.modifiedAfter)} />
      <ConfigRow label="Modified before" value={formatDateTime(filter.modifiedBefore)} />
    </div>
  );
}

function RestoreConfigSection({ config }: { config: RestoreConfig }) {
  if (config.bucketType === "s3") {
    return (
      <div className="space-y-2 text-sm">
        <ConfigRow label="Bucket type" value="S3" />
        <ConfigRow label="Region" value={config.bucketConfig.region} />
        <ConfigRow label="Bucket name" value={config.bucketConfig.bucketName} />
        <ConfigRow label="Access key ID" value={config.bucketConfig.accessKeyId} />
        <ConfigRow label="Secret access key" value={config.bucketConfig.secretAccessKey} />
      </div>
    );
  }

  return (
    <div className="space-y-2 text-sm">
      <ConfigRow label="Bucket type" value="Azure Blob" />
      <ConfigRow label="Container name" value={config.bucketConfig.containerName} />
      <ConfigRow label="Connection string" value={config.bucketConfig.connectionString} />
    </div>
  );
}

function JobConfigSection({ job }: { job: RestoreJob }) {
  return (
    <Card className="p-6">
      <h2 className="text-lg font-semibold">Job configuration</h2>
      <p className="mt-1 text-sm text-muted-foreground">
        One-time restore from bucket to SharePoint.
      </p>
      <div className="mt-4 space-y-6">
        <div className="space-y-2 text-sm">
          <ConfigRow label="Job ID" value={job.id} />
          <ConfigRow label="SharePoint site URL" value={job.sharepointSiteUrl} />
          <ConfigRow label="Run mode" value={formatRunMode(job.runMode)} />
          <ConfigRow label="Scheduled at" value={formatDateTime(job.scheduledAt)} />
          <ConfigRow label="Last run at" value={formatDateTime(job.lastRunAt)} />
          <ConfigRow label="Created at" value={formatDateTime(job.createdAt)} />
          <ConfigRow label="Updated at" value={formatDateTime(job.updatedAt)} />
        </div>

        <div>
          <h3 className="mb-2 text-sm font-semibold">Filter</h3>
          <FilterConfigSection filter={job.filter} />
        </div>

        <div>
          <h3 className="mb-2 text-sm font-semibold">Source bucket</h3>
          <RestoreConfigSection config={job.restoreConfig} />
        </div>
      </div>
    </Card>
  );
}

export default function DashboardRestoreJobDetailPage() {
  const params = useParams<{ id: string }>();
  const jobId = params.id ?? "";
  const job = getPlaceholderRestoreJobById(jobId);
  const logs = getPlaceholderRestoreJobLogs(jobId);

  if (!job) {
    return (
      <main className="p-6">
        <Card className="p-6">
          <p className="text-sm text-destructive">Restore job not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href="/dashboard/restore-job/list">Back to list</Link>
          </Button>
        </Card>
      </main>
    );
  }

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Restore job detail</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {job.sharepointSiteUrl} · bucket → SharePoint
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href="/dashboard/restore-job/list">Back to list</Link>
        </Button>
      </div>

      <div className="grid max-w-4xl gap-6">
        <JobConfigSection job={job} />

        <Card className="p-6">
          <h2 className="text-lg font-semibold">Run logs</h2>
          {logs.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">No run logs yet.</p>
          ) : (
            <div className="mt-4 space-y-3">
              {logs.map((log) => (
                <div
                  key={log.id}
                  className="flex flex-col gap-3 rounded-md border p-4 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div className="space-y-1 text-sm">
                    <p>
                      <span className="font-medium">Status:</span> {formatLogStatus(log.status)}
                    </p>
                    <p>
                      <span className="font-medium">Started:</span>{" "}
                      {formatDateTime(log.createdAt)}
                    </p>
                    <p>
                      <span className="font-medium">Ended:</span> {formatDateTime(log.endedAt)}
                    </p>
                  </div>
                  <Button asChild variant="outline" size="sm">
                    <Link href={`/dashboard/restore-job/${job.id}/logs/${log.id}`}>Detail</Link>
                  </Button>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>
    </main>
  );
}
