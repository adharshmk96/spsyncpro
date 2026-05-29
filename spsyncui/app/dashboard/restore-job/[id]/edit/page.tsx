import Link from "next/link";

import { RestoreJobForm } from "@/components/restore-job-form";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { serverApiFetch } from "@/lib/api/server";
import type { RestoreJobResponse } from "@/lib/api/types";

export default async function DashboardRestoreJobEditPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let job: RestoreJobResponse["restore_job"] | null = null;
  try {
    job = (await serverApiFetch<RestoreJobResponse>(`/restore-jobs/${id}`)).restore_job;
  } catch (error) {
    console.error("Failed to load restore job.", error);
  }

  if (!job) {
    return (
      <main className="p-6">
        <Card className="mx-auto max-w-3xl p-6">
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
      <RestoreJobForm mode="edit" job={job} />
    </main>
  );
}
