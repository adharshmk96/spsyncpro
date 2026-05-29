import Link from "next/link";

import { BackupJobForm } from "@/components/backup-job-form";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { serverApiFetch } from "@/lib/api/server";
import type { BackupJobResponse } from "@/lib/api/types";

export default async function DashboardBackupJobEditPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let job: BackupJobResponse["backup_job"] | null = null;
  try {
    job = (await serverApiFetch<BackupJobResponse>(`/backup-jobs/${id}`)).backup_job;
  } catch (error) {
    console.error("Failed to load backup job.", error);
  }

  if (!job) {
    return (
      <main className="p-6">
        <Card className="mx-auto max-w-5xl p-6">
          <p className="text-sm text-destructive">Backup job not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href="/dashboard/backup-job/list">Back to list</Link>
          </Button>
        </Card>
      </main>
    );
  }

  return (
    <main className="p-6">
      <BackupJobForm mode="edit" job={job} />
    </main>
  );
}
