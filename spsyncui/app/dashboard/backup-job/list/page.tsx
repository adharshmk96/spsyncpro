"use client";

import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  formatBackupConfigSummary,
  formatDateTime,
  formatFilterSummary,
  formatJobType,
} from "@/lib/backup-jobs/format";
import { getPlaceholderBackupJobs } from "@/lib/backup-jobs/placeholder-data";

export default function DashboardBackupJobListPage() {
  const jobs = getPlaceholderBackupJobs();

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

      {jobs.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No backup jobs created yet.</Card>
      ) : null}

      <div className="grid gap-4">
        {jobs.map((job) => (
          <Card key={job.id} className="p-4">
            <div className="flex items-start justify-between gap-4">
              <div className="space-y-1">
                <p className="font-semibold">{job.sharepointSiteUrl}</p>
                <p className="text-sm text-muted-foreground">{formatFilterSummary(job.filter)}</p>
                <p className="text-sm text-muted-foreground">
                  Job type: {formatJobType(job.jobType)}
                </p>
                <p className="text-sm text-muted-foreground">
                  Backup: {formatBackupConfigSummary(job.backupConfig)}
                </p>
                <p className="text-sm text-muted-foreground">
                  Next run: {formatDateTime(job.nextRunAt)}
                </p>
                <p className="text-sm text-muted-foreground">
                  Last run: {formatDateTime(job.lastRunAt)}
                </p>
                <p className="text-sm text-muted-foreground">
                  Created: {formatDateTime(job.createdAt)} · Updated:{" "}
                  {formatDateTime(job.updatedAt)}
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
