"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { formatDateTime, formatSchedule, parseDocumentLibraries } from "@/lib/api/format";
import type { BackupJob, BackupJobsResponse } from "@/lib/api/types";

export default function DashboardBackupJobListPage() {
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<BackupJobsResponse>("/backup-jobs");
        if (!active) return;
        setJobs(data.backup_jobs ?? []);
      } catch (error) {
        console.error("Failed to load backup jobs.", error);
        if (active) setErrorMessage(toErrorMessage(error, "Failed to load backup jobs."));
      } finally {
        if (active) setIsLoading(false);
      }
    })();
    return () => {
      active = false;
    };
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

      {errorMessage ? <p className="mb-4 text-sm text-destructive">{errorMessage}</p> : null}

      {isLoading ? (
        <Card className="p-4 text-sm text-muted-foreground">Loading backup jobs...</Card>
      ) : jobs.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No backup jobs created yet.</Card>
      ) : (
        <div className="grid gap-4">
          {jobs.map((job) => (
            <Card key={job.id} className="p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                  <p className="font-semibold">{job.job_config.share_point_site}</p>
                  <p className="text-sm text-muted-foreground">
                    Libraries:{" "}
                    {parseDocumentLibraries(job.job_config.filters.document_libraries).join(", ") ||
                      "-"}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Schedule: {formatSchedule(job.schedule)} · {job.active ? "Active" : "Paused"}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Next run: {formatDateTime(job.next_run)} · Last run:{" "}
                    {formatDateTime(job.last_run)}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Created: {formatDateTime(job.created_at)} · Updated:{" "}
                    {formatDateTime(job.updated_at)}
                  </p>
                </div>
                <Button asChild variant="outline">
                  <Link href={`/dashboard/backup-job/${job.id}`}>View</Link>
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </main>
  );
}
