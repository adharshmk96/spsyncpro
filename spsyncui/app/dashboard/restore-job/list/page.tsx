"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { formatDateTime } from "@/lib/api/format";
import type { RestoreJob, RestoreJobsResponse } from "@/lib/api/types";

export default function DashboardRestoreJobListPage() {
  const [jobs, setJobs] = useState<RestoreJob[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<RestoreJobsResponse>("/restore-jobs");
        if (!active) return;
        setJobs(data.restore_jobs ?? []);
      } catch (error) {
        console.error("Failed to load restore jobs.", error);
        if (active) setErrorMessage(toErrorMessage(error, "Failed to load restore jobs."));
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
          <h1 className="text-2xl font-semibold">Restores</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Jobs that restore files from a bucket store to a SharePoint site.
          </p>
        </div>
        <Button asChild>
          <Link href="/dashboard/restore-job/create">Create Restore Job</Link>
        </Button>
      </div>

      {errorMessage ? <p className="mb-4 text-sm text-destructive">{errorMessage}</p> : null}

      {isLoading ? (
        <Card className="p-4 text-sm text-muted-foreground">Loading restore jobs...</Card>
      ) : jobs.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No restore jobs created yet.</Card>
      ) : (
        <div className="grid gap-4">
          {jobs.map((job) => (
            <Card key={job.id} className="p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                  <p className="font-semibold">{job.job_config.share_point_site}</p>
                  <p className="text-sm text-muted-foreground">
                    {job.active ? "Active" : "Paused"} · Scheduled start:{" "}
                    {formatDateTime(job.start_at)}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Last run: {formatDateTime(job.last_run)}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Created: {formatDateTime(job.created_at)} · Updated:{" "}
                    {formatDateTime(job.updated_at)}
                  </p>
                </div>
                <Button asChild variant="outline">
                  <Link href={`/dashboard/restore-job/${job.id}`}>View</Link>
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </main>
  );
}
