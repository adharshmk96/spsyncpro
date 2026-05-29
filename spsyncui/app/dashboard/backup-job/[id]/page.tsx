"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useJobReferences } from "@/hooks/use-job-references";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import {
  deriveRunStatus,
  formatDateTime,
  formatRunStatus,
  formatSchedule,
  parseDocumentLibraries,
} from "@/lib/api/format";
import type {
  BackupJob,
  BackupJobResponse,
  BackupRunsResponse,
  RunDetails,
} from "@/lib/api/types";

function ConfigRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[200px_1fr]">
      <span className="font-medium text-muted-foreground">{label}</span>
      <span className="break-all">{value}</span>
    </div>
  );
}

export default function DashboardBackupJobDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const jobId = params.id ?? "";
  const references = useJobReferences();

  const [job, setJob] = useState<BackupJob | null>(null);
  const [runs, setRuns] = useState<RunDetails[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [stoppingId, setStoppingId] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchRuns = useCallback(async (): Promise<RunDetails[]> => {
    const runData = await clientApiFetch<BackupRunsResponse>(`/backup-runs?job_id=${jobId}`);
    return runData.backup_runs ?? [];
  }, [jobId]);

  useEffect(() => {
    if (!jobId) {
      return;
    }
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<BackupJobResponse>(`/backup-jobs/${jobId}`);
        if (!active) return;
        setJob(data.backup_job);
        const runs = await fetchRuns();
        if (active) setRuns(runs);
      } catch (error) {
        console.error("Failed to load backup job.", error);
        if (active) setNotFound(true);
      } finally {
        if (active) setIsLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [jobId, fetchRuns]);

  const orgName =
    references.organizations.find((org) => org.id === job?.job_config.organization)?.name ??
    job?.job_config.organization ??
    "-";
  const bucketName =
    references.bucketStores.find((store) => store.id === job?.job_config.bucket_store)?.bucket_name ??
    job?.job_config.bucket_store ??
    "-";

  const handleRunNow = async () => {
    setIsRunning(true);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/backup-jobs/${jobId}/runs`, { method: "POST" });
      setRuns(await fetchRuns());
    } catch (error) {
      console.error("Failed to start backup run.", error);
      setErrorMessage(toErrorMessage(error, "Failed to start backup run."));
    } finally {
      setIsRunning(false);
    }
  };

  const handleStop = async (runId: string) => {
    setStoppingId(runId);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/backup-runs/${runId}/stop`, { method: "POST" });
      setRuns(await fetchRuns());
    } catch (error) {
      console.error("Failed to stop backup run.", error);
      setErrorMessage(toErrorMessage(error, "Failed to stop backup run."));
    } finally {
      setStoppingId(null);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm("Delete this backup job? This cannot be undone.")) {
      return;
    }
    setIsDeleting(true);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/backup-jobs/${jobId}`, { method: "DELETE" });
      router.push("/dashboard/backup-job/list");
      router.refresh();
    } catch (error) {
      console.error("Failed to delete backup job.", error);
      setErrorMessage(toErrorMessage(error, "Failed to delete backup job."));
      setIsDeleting(false);
    }
  };

  if (notFound) {
    return (
      <main className="p-6">
        <Card className="p-6">
          <p className="text-sm text-destructive">Backup job not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href="/dashboard/backup-job/list">Back to list</Link>
          </Button>
        </Card>
      </main>
    );
  }

  if (isLoading || !job) {
    return (
      <main className="p-6">
        <Card className="p-6 text-sm text-muted-foreground">Loading backup job...</Card>
      </main>
    );
  }

  const filters = job.job_config.filters;

  return (
    <main className="p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Backup job detail</h1>
          <p className="mt-1 text-sm text-muted-foreground">{job.job_config.share_point_site}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button onClick={handleRunNow} disabled={isRunning}>
            {isRunning ? "Starting..." : "Run now"}
          </Button>
          <Button asChild variant="outline">
            <Link href={`/dashboard/backup-job/${job.id}/edit`}>Edit</Link>
          </Button>
          <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
            {isDeleting ? "Deleting..." : "Delete"}
          </Button>
          <Button asChild variant="outline">
            <Link href="/dashboard/backup-job/list">Back to list</Link>
          </Button>
        </div>
      </div>

      {errorMessage ? <p className="mb-4 text-sm text-destructive">{errorMessage}</p> : null}

      <div className="grid max-w-4xl gap-6">
        <Card className="p-6">
          <h2 className="text-lg font-semibold">Job configuration</h2>
          <div className="mt-4 space-y-2 text-sm">
            <ConfigRow label="Job ID" value={job.id} />
            <ConfigRow label="Organization" value={orgName} />
            <ConfigRow label="Bucket store" value={bucketName} />
            <ConfigRow label="SharePoint site URL" value={job.job_config.share_point_site} />
            <ConfigRow
              label="Document libraries"
              value={parseDocumentLibraries(filters.document_libraries).join(", ") || "-"}
            />
            <ConfigRow label="Schedule" value={formatSchedule(job.schedule)} />
            <ConfigRow label="Active" value={job.active ? "Yes" : "No"} />
            {job.start_at ? (
              <ConfigRow label="Schedule start" value={formatDateTime(job.start_at)} />
            ) : null}
            <ConfigRow label="Last run" value={formatDateTime(job.last_run)} />
            <ConfigRow label="Next run" value={formatDateTime(job.next_run)} />
            <ConfigRow label="Created at" value={formatDateTime(job.created_at)} />
            <ConfigRow label="Updated at" value={formatDateTime(job.updated_at)} />
          </div>
        </Card>

        <Card className="p-6">
          <h2 className="text-lg font-semibold">Run history</h2>
          {runs.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">No runs yet.</p>
          ) : (
            <div className="mt-4 space-y-3">
              {runs.map((run) => {
                const status = deriveRunStatus(run);
                return (
                  <div
                    key={run.id}
                    className="flex flex-col gap-3 rounded-md border p-4 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="space-y-1 text-sm">
                      <p>
                        <span className="font-medium">Status:</span> {formatRunStatus(status)}
                      </p>
                      <p>
                        <span className="font-medium">Started:</span> {formatDateTime(run.start_at)}
                      </p>
                      <p>
                        <span className="font-medium">Ended:</span> {formatDateTime(run.end_at)}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      {status === "running" ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleStop(run.id)}
                          disabled={stoppingId === run.id}
                        >
                          {stoppingId === run.id ? "Stopping..." : "Stop"}
                        </Button>
                      ) : null}
                      <Button asChild variant="outline" size="sm">
                        <Link href={`/dashboard/backup-job/${job.id}/logs/${run.id}`}>Detail</Link>
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </Card>
      </div>
    </main>
  );
}
