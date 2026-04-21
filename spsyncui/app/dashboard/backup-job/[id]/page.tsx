"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

type BackupJobDetail = {
  id: string;
  siteUrl: string;
  documentLibraryList: string[];
  runMode: "IMMEDIATE" | "ONE_TIME_AT" | "RECURRING";
  recurrence: "DAILY" | "WEEKLY" | "MONTHLY" | null;
  startAt: string | null;
  status: "ACTIVE" | "PAUSED";
  nextRunAt: string | null;
  lastRunAt: string | null;
  lastRunStatus: "RUNNING" | "SUCCESS" | "FAILED" | "CANCELLED" | null;
  runSummary:
    | {
        id: string;
        status: "QUEUED" | "RUNNING" | "SUCCESS" | "FAILED" | "CANCELLED";
        scheduledFor: string;
        startedAt: string | null;
        finishedAt: string | null;
      }
    | null;
};

type BackupRun = {
  id: string;
  scheduledFor: string;
  startedAt: string | null;
  finishedAt: string | null;
  status: "QUEUED" | "RUNNING" | "SUCCESS" | "FAILED" | "CANCELLED";
  errorMessage: string | null;
};

function getRunConfigSummary(job: BackupJobDetail): string {
  if (job.runMode === "IMMEDIATE") {
    return "Job runs immediately after dispatch.";
  }

  if (job.runMode === "ONE_TIME_AT") {
    return `Job runs once at ${job.startAt ? new Date(job.startAt).toLocaleString() : "-"}.`;
  }

  return `Job runs ${job.recurrence?.toLowerCase() ?? "recurring"} from ${
    job.startAt ? new Date(job.startAt).toLocaleString() : "-"
  }.`;
}

export default function DashboardBackupJobDetailPage() {
  const params = useParams<{ id: string }>();
  const [job, setJob] = useState<BackupJobDetail | null>(null);
  const [runs, setRuns] = useState<BackupRun[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isTriggering, setIsTriggering] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    const loadJob = async () => {
      if (!params.id) {
        setErrorMessage("Missing backup job id.");
        setIsLoading(false);
        return;
      }

      setIsLoading(true);
      setErrorMessage(null);

      try {
        const response = await fetch(`/api/backup-jobs/${params.id}`);
        const data = (await response.json()) as { job?: BackupJobDetail; error?: string };
        if (!response.ok) {
          throw new Error(data.error ?? "Failed to load backup job.");
        }

        setJob(data.job ?? null);

        const runsResponse = await fetch(`/api/backup-jobs/${params.id}/runs?limit=20`);
        const runsData = (await runsResponse.json()) as { runs?: BackupRun[]; error?: string };
        if (!runsResponse.ok) {
          throw new Error(runsData.error ?? "Failed to load run history.");
        }
        setRuns(runsData.runs ?? []);
      } catch (error) {
        console.error("Failed to load backup job detail.", error);
        setErrorMessage(error instanceof Error ? error.message : "Failed to load backup job.");
      } finally {
        setIsLoading(false);
      }
    };

    void loadJob();
  }, [params.id]);

  const handleRunNow = async () => {
    if (!params.id) {
      return;
    }

    setIsTriggering(true);
    setErrorMessage(null);
    try {
      const response = await fetch(`/api/backup-jobs/${params.id}/runs`, {
        method: "POST",
      });
      const data = (await response.json()) as { error?: string };
      if (!response.ok) {
        throw new Error(data.error ?? "Failed to trigger run.");
      }

      const runsResponse = await fetch(`/api/backup-jobs/${params.id}/runs?limit=20`);
      const runsData = (await runsResponse.json()) as { runs?: BackupRun[]; error?: string };
      if (!runsResponse.ok) {
        throw new Error(runsData.error ?? "Failed to refresh run history.");
      }
      setRuns(runsData.runs ?? []);
    } catch (error) {
      console.error("Failed to trigger backup run.", error);
      setErrorMessage(error instanceof Error ? error.message : "Failed to trigger backup run.");
    } finally {
      setIsTriggering(false);
    }
  };

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Backup Job Detail</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Placeholder detail page with run configuration.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href="/dashboard/backup-job/list">Back to List</Link>
        </Button>
      </div>

      {isLoading ? <p className="text-sm text-muted-foreground">Loading job...</p> : null}
      {errorMessage ? <p className="text-sm text-destructive">{errorMessage}</p> : null}

      {!isLoading && !errorMessage && job ? (
        <Card className="max-w-3xl p-6">
          <h2 className="text-lg font-semibold">Run Config</h2>
          <div className="mt-4 space-y-2 text-sm">
            <p>
              <span className="font-medium">Site URL:</span> {job.siteUrl}
            </p>
            <p>
              <span className="font-medium">Libraries:</span> {job.documentLibraryList.join(", ")}
            </p>
            <p>
              <span className="font-medium">Run Mode:</span> {job.runMode}
            </p>
            <p>
              <span className="font-medium">Recurrence:</span> {job.recurrence ?? "-"}
            </p>
            <p>
              <span className="font-medium">Start At:</span>{" "}
              {job.startAt ? new Date(job.startAt).toLocaleString() : "-"}
            </p>
            <p>
              <span className="font-medium">Status:</span> {job.status}
            </p>
            <p>
              <span className="font-medium">Next Run:</span>{" "}
              {job.nextRunAt ? new Date(job.nextRunAt).toLocaleString() : "-"}
            </p>
            <p>
              <span className="font-medium">Last Run:</span>{" "}
              {job.lastRunAt ? new Date(job.lastRunAt).toLocaleString() : "-"}
            </p>
            <p>
              <span className="font-medium">Last Run Status:</span> {job.lastRunStatus ?? "-"}
            </p>
            <p className="text-muted-foreground">{getRunConfigSummary(job)}</p>
          </div>
        </Card>
      ) : null}

      {!isLoading && !errorMessage && job ? (
        <Card className="mt-6 max-w-3xl p-6">
          <div className="mb-4 flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">Run History</h2>
            <Button type="button" onClick={handleRunNow} disabled={isTriggering}>
              {isTriggering ? "Triggering..." : "Run Now"}
            </Button>
          </div>
          {runs.length === 0 ? (
            <p className="text-sm text-muted-foreground">No runs yet.</p>
          ) : (
            <div className="space-y-3">
              {runs.map((run) => (
                <div key={run.id} className="rounded-md border p-3 text-sm">
                  <p>
                    <span className="font-medium">Status:</span> {run.status}
                  </p>
                  <p>
                    <span className="font-medium">Scheduled For:</span>{" "}
                    {new Date(run.scheduledFor).toLocaleString()}
                  </p>
                  <p>
                    <span className="font-medium">Started At:</span>{" "}
                    {run.startedAt ? new Date(run.startedAt).toLocaleString() : "-"}
                  </p>
                  <p>
                    <span className="font-medium">Finished At:</span>{" "}
                    {run.finishedAt ? new Date(run.finishedAt).toLocaleString() : "-"}
                  </p>
                  <p>
                    <span className="font-medium">Error:</span> {run.errorMessage ?? "-"}
                  </p>
                </div>
              ))}
            </div>
          )}
        </Card>
      ) : null}
    </main>
  );
}
