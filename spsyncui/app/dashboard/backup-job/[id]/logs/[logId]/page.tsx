"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { clientApiFetch } from "@/lib/api/client";
import { deriveRunStatus, formatDateTime, formatRunStatus } from "@/lib/api/format";
import type { BackupRunResponse, FileTransfer, RunDetails } from "@/lib/api/types";

export default function DashboardBackupRunLogPage() {
  const params = useParams<{ id: string; logId: string }>();
  const jobId = params.id ?? "";
  const runId = params.logId ?? "";

  const [run, setRun] = useState<RunDetails | null>(null);
  const [files, setFiles] = useState<FileTransfer[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (!runId) {
      return;
    }
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<BackupRunResponse>(`/backup-runs/${runId}`);
        if (!active) return;
        setRun(data.backup_run);
        setFiles(data.file_transfers ?? []);
        setTotal(data.pagination?.total ?? data.file_transfers?.length ?? 0);
      } catch (error) {
        console.error("Failed to load backup run.", error);
        if (active) setNotFound(true);
      } finally {
        if (active) setIsLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, [runId]);

  if (notFound) {
    return (
      <main className="p-6">
        <Card className="p-6">
          <p className="text-sm text-destructive">Run not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href={`/dashboard/backup-job/${jobId || "list"}`}>
              {jobId ? "Back to job" : "Back to list"}
            </Link>
          </Button>
        </Card>
      </main>
    );
  }

  if (isLoading || !run) {
    return (
      <main className="p-6">
        <Card className="p-6 text-sm text-muted-foreground">Loading run...</Card>
      </main>
    );
  }

  const status = deriveRunStatus(run);

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Run detail</h1>
          <p className="mt-1 text-sm text-muted-foreground">{formatRunStatus(status)}</p>
        </div>
        <Button asChild variant="outline">
          <Link href={`/dashboard/backup-job/${jobId}`}>Back to job</Link>
        </Button>
      </div>

      <div className="grid max-w-5xl gap-6">
        <Card className="p-6">
          <h2 className="text-lg font-semibold">Run summary</h2>
          <div className="mt-4 space-y-2 text-sm">
            <p>
              <span className="font-medium text-muted-foreground">Run ID:</span> {run.id}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Status:</span>{" "}
              {formatRunStatus(status)}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Started:</span>{" "}
              {formatDateTime(run.start_at)}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Ended:</span>{" "}
              {formatDateTime(run.end_at)}
            </p>
            <p>
              <span className="font-medium text-muted-foreground">Files:</span> {total}
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
                    <th className="px-3 py-2 font-medium">File path</th>
                    <th className="px-3 py-2 font-medium">Status</th>
                    <th className="px-3 py-2 font-medium">Library</th>
                    <th className="px-3 py-2 font-medium">Size</th>
                    <th className="px-3 py-2 font-medium">Start at</th>
                    <th className="px-3 py-2 font-medium">End at</th>
                  </tr>
                </thead>
                <tbody>
                  {files.map((file, index) => (
                    <tr key={`${file.file_path}-${index}`} className="border-b last:border-b-0">
                      <td className="px-3 py-2 break-all">{file.file_path}</td>
                      <td className="px-3 py-2 whitespace-nowrap">{file.status}</td>
                      <td className="px-3 py-2 whitespace-nowrap">{file.drive_name ?? "—"}</td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {file.size != null ? file.size.toLocaleString() : "—"}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">
                        {formatDateTime(file.start_at)}
                      </td>
                      <td className="px-3 py-2 whitespace-nowrap">{formatDateTime(file.end_at)}</td>
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
