"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  formatDateTime,
  formatFilterSummary,
  formatRestoreConfigSummary,
  formatRunMode,
} from "@/lib/restore-jobs/format";
import { getPlaceholderRestoreJobs } from "@/lib/restore-jobs/placeholder-data";

export default function DashboardRestoreJobListPage() {
  const jobs = getPlaceholderRestoreJobs();

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Restores</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            One-time jobs that restore files from a bucket to a SharePoint site.
          </p>
        </div>
        <Button asChild>
          <Link href="/dashboard/restore-job/create">Create Restore Job</Link>
        </Button>
      </div>

      {jobs.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No restore jobs created yet.</Card>
      ) : null}

      <div className="grid gap-4">
        {jobs.map((job) => (
          <Card key={job.id} className="p-4">
            <div className="flex items-start justify-between gap-4">
              <div className="space-y-1">
                <p className="font-semibold">{job.sharepointSiteUrl}</p>
                <p className="text-sm text-muted-foreground">{formatFilterSummary(job.filter)}</p>
                <p className="text-sm text-muted-foreground">Run: {formatRunMode(job.runMode)}</p>
                <p className="text-sm text-muted-foreground">
                  Source: {formatRestoreConfigSummary(job.restoreConfig)}
                </p>
                {job.runMode === "scheduled" ? (
                  <p className="text-sm text-muted-foreground">
                    Scheduled for: {formatDateTime(job.scheduledAt)}
                  </p>
                ) : null}
                <p className="text-sm text-muted-foreground">
                  Last run: {formatDateTime(job.lastRunAt)}
                </p>
                <p className="text-sm text-muted-foreground">
                  Created: {formatDateTime(job.createdAt)} · Updated:{" "}
                  {formatDateTime(job.updatedAt)}
                </p>
              </div>
              <Button asChild variant="outline">
                <Link href={`/dashboard/restore-job/${job.id}`}>View</Link>
              </Button>
            </div>
          </Card>
        ))}
      </div>
    </main>
  );
}
