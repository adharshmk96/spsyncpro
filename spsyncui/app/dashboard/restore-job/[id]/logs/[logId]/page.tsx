"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  formatDateTime,
  formatFileSize,
  formatLogStatus,
  formatTransferStatus,
} from "@/lib/restore-jobs/format";
import {
  getPlaceholderRestoreJobById,
  getPlaceholderRestoreJobLogById,
  getPlaceholderRestoreJobLogFiles,
} from "@/lib/restore-jobs/placeholder-data";

export default function DashboardRestoreJobLogDetailPage() {
  const params = useParams<{ id: string; logId: string }>();
  const jobId = params.id ?? "";
  const logId = params.logId ?? "";

  const job = getPlaceholderRestoreJobById(jobId);
  const log = getPlaceholderRestoreJobLogById(logId);
  const files = getPlaceholderRestoreJobLogFiles(logId);

  if (!job || !log || log.jobId !== job.id) {
    return (
      <main className="p-6">
        <Card className="p-6">
          <p className="text-sm text-destructive">Run log not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href={`/dashboard/restore-job/${jobId || "list"}`}>
              {jobId ? "Back to job" : "Back to list"}
            </Link>
          </Button>
        </Card>
      </main>
    );
  }

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Restore run log</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {job.sharepointSiteUrl} · {formatLogStatus(log.status)} · bucket → SharePoint
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href={`/dashboard/restore-job/${job.id}`}>Back to job</Link>
        </Button>
      </div>

      <div className="grid max-w-5xl gap-6">
        <Card className="p-6">
          <h2 className="text-lg font-semibold">Run summary</h2>
          <div className="mt-4 space-y-2 text-sm">
            <p>
              <span className="font-medium text-muted-foreground">Log ID:</span> {log.id}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Status:</span>{" "}
              {formatLogStatus(log.status)}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Started:</span>{" "}
              {formatDateTime(log.createdAt)}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Ended:</span>{" "}
              {formatDateTime(log.endedAt)}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Files:</span> {files.length}
            </p>
          </div>
        </Card>

        <Card className="p-6">
          <h2 className="text-lg font-semibold">File transfers</h2>
          {files.length === 0 ? (
            <p className="mt-4 text-sm text-muted-foreground">No files recorded for this run.</p>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="w-full min-w-[640px] border-collapse text-left text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="px-3 py-2 font-medium">Filename</th>
                    <th className="px-3 py-2 font-medium">Size</th>
                    <th className="px-3 py-2 font-medium">Transfer status</th>
                    <th className="px-3 py-2 font-medium">Start at</th>
                    <th className="px-3 py-2 font-medium">End at</th>
                  </tr>
                </thead>
                <tbody>
                  {files.map((file) => (
                    <tr key={file.id} className="border-b last:border-b-0">
                      <td className="px-3 py-2 break-all">{file.filename}</td>
                      <td className="px-3 py-2 whitespace-nowrap">{formatFileSize(file.size)}</td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {formatTransferStatus(file.transferStatus)}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {formatDateTime(file.startAt)}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {formatDateTime(file.endAt)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </div>
    </main>
  );
}
