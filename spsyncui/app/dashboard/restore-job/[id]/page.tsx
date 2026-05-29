"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useJobReferences } from "@/hooks/use-job-references";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { deriveRunStatus, formatDateTime, formatRunStatus } from "@/lib/api/format";
import type {
  RestoreJob,
  RestoreJobResponse,
  RestoreRunsResponse,
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

export default function DashboardRestoreJobDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const jobId = params.id ?? "";
  const references = useJobReferences();

  const [job, setJob] = useState<RestoreJob | null>(null);
  const [runs, setRuns] = useState<RunDetails[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [stoppingId, setStoppingId] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchRuns = useCallback(async (): Promise<RunDetails[]> => {
    const runData = await clientApiFetch<RestoreRunsResponse>(`/restore-runs?job_id=${jobId}`);
    return runData.restore_runs ?? [];
  }, [jobId]);

  useEffect(() => {
    if (!jobId) {
      return;
    }
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<RestoreJobResponse>(`/restore-jobs/${jobId}`);
        if (!active) return;
        setJob(data.restore_job);
        const runs = await fetchRuns();
        if (active) setRuns(runs);
      } catch (error) {
        console.error("Failed to load restore job.", error);
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
      await clientApiFetch(`/restore-jobs/${jobId}/runs`, { method: "POST" });
      setRuns(await fetchRuns());
    } catch (error) {
      console.error("Failed to start restore run.", error);
      setErrorMessage(toErrorMessage(error, "Failed to start restore run."));
    } finally {
      setIsRunning(false);
    }
  };

  const handleStop = async (runId: string) => {
    setStoppingId(runId);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/restore-runs/${runId}/stop`, { method: "POST" });
      setRuns(await fetchRuns());
    } catch (error) {
      console.error("Failed to stop restore run.", error);
      setErrorMessage(toErrorMessage(error, "Failed to stop restore run."));
    } finally {
      setStoppingId(null);
    }
  };

  const handleDelete = async () => {
    if (!window.confirm("Delete this restore job? This cannot be undone.")) {
      return;
    }
    setIsDeleting(true);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/restore-jobs/${jobId}`, { method: "DELETE" });
      router.push("/dashboard/restore-job/list");
      router.refresh();
    } catch (error) {
      console.error("Failed to delete restore job.", error);
      setErrorMessage(toErrorMessage(error, "Failed to delete restore job."));
      setIsDeleting(false);
    }
  };

  if (notFound) {
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

  if (isLoading || !job) {
    return (
      <main className="p-6">
        <Card className="p-6 text-sm text-muted-foreground">Loading restore job...</Card>
      </main>
    );
  }

  return (
    <main className="p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Restore job detail</h1>
          <p className="mt-1 text-sm text-muted-foreground">{job.job_config.share_point_site}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button onClick={handleRunNow} disabled={isRunning}>
            {isRunning ? "Starting..." : "Run now"}
          </Button>
          <Button asChild variant="outline">
            <Link href={`/dashboard/restore-job/${job.id}/edit`}>Edit</Link>
          </Button>
          <Button variant="destructive" onClick={handleDelete} disabled={isDeleting}>
            {isDeleting ? "Deleting..." : "Delete"}
          </Button>
          <Button asChild variant="outline">
            <Link href="/dashboard/restore-job/list">Back to list</Link>
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
            <ConfigRow label="Active" value={job.active ? "Yes" : "No"} />
            <ConfigRow label="Scheduled start" value={formatDateTime(job.start_at)} />
            <ConfigRow label="Last run" value={formatDateTime(job.last_run)} />
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
                        <Link href={`/dashboard/restore-job/${job.id}/logs/${run.id}`}>Detail</Link>
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
