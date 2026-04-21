"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

type BackupJob = {
  id: string;
  siteUrl: string;
  documentLibraryList: string[];
  runMode: "IMMEDIATE" | "ONE_TIME_AT" | "RECURRING";
  recurrence: "DAILY" | "WEEKLY" | "MONTHLY" | null;
  startAt: string | null;
  status: "ACTIVE" | "PAUSED";
  createdAt: string;
  nextRunAt: string | null;
  lastRunAt: string | null;
  lastRunStatus: "RUNNING" | "SUCCESS" | "FAILED" | "CANCELLED" | null;
};

function formatSchedule(job: BackupJob): string {
  if (job.runMode === "IMMEDIATE") {
    return "Immediate";
  }

  if (job.runMode === "ONE_TIME_AT") {
    return `One-time at ${job.startAt ? new Date(job.startAt).toLocaleString() : "-"}`;
  }

  return `${job.recurrence ?? "Recurring"} from ${
    job.startAt ? new Date(job.startAt).toLocaleString() : "-"
  }`;
}

export default function DashboardBackupJobListPage() {
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    const loadJobs = async () => {
      setIsLoading(true);
      setErrorMessage(null);

      try {
        const response = await fetch("/api/backup-jobs");
        const data = (await response.json()) as { jobs?: BackupJob[]; error?: string };
        if (!response.ok) {
          throw new Error(data.error ?? "Failed to load backup jobs.");
        }

        setJobs(data.jobs ?? []);
      } catch (error) {
        console.error("Failed to load backup jobs.", error);
        setErrorMessage(error instanceof Error ? error.message : "Failed to load backup jobs.");
      } finally {
        setIsLoading(false);
      }
    };

    void loadJobs();
  }, []);

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Backups</h1>
          <p className="mt-1 text-sm text-muted-foreground">List of all backup jobs.</p>
        </div>
        <Button asChild>
          <Link href="/dashboard/backup-job/create">Create Backup Job</Link>
        </Button>
      </div>

      {isLoading ? <p className="text-sm text-muted-foreground">Loading jobs...</p> : null}
      {errorMessage ? <p className="text-sm text-destructive">{errorMessage}</p> : null}

      {!isLoading && !errorMessage && jobs.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No backup jobs created yet.</Card>
      ) : null}

      <div className="grid gap-4">
        {jobs.map((job) => (
          <Card key={job.id} className="p-4">
            <div className="flex items-start justify-between gap-4">
              <div className="space-y-1">
                <p className="font-semibold">{job.siteUrl}</p>
                <p className="text-sm text-muted-foreground">
                  Libraries: {job.documentLibraryList.join(", ")}
                </p>
                <p className="text-sm text-muted-foreground">Schedule: {formatSchedule(job)}</p>
                <p className="text-sm text-muted-foreground">Status: {job.status}</p>
                <p className="text-sm text-muted-foreground">
                  Next Run: {job.nextRunAt ? new Date(job.nextRunAt).toLocaleString() : "-"}
                </p>
                <p className="text-sm text-muted-foreground">
                  Last Run: {job.lastRunAt ? new Date(job.lastRunAt).toLocaleString() : "-"}
                </p>
                <p className="text-sm text-muted-foreground">
                  Last Run Status: {job.lastRunStatus ?? "-"}
                </p>
                <p className="text-sm text-muted-foreground">
                  Created: {new Date(job.createdAt).toLocaleString()}
                </p>
              </div>
              <Button asChild variant="outline">
                <Link href={`/dashboard/backup-job/${job.id}`}>View</Link>
              </Button>
            </div>
          </Card>
        ))}
      </div>
    </main>
  );
}
