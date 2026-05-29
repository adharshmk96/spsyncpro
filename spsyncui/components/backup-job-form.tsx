"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useJobReferences } from "@/hooks/use-job-references";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { isoToLocalDateTime, localDateTimeToIso, parseDocumentLibraries } from "@/lib/api/format";
import type {
  BackupJob,
  BackupJobInput,
  BackupJobResponse,
  BackupRunStartResponse,
  Schedule,
} from "@/lib/api/types";

type ScheduleType = "interval" | "cron" | "one_time";

type BackupJobFormProps = {
  mode: "create" | "edit";
  job?: BackupJob;
};

function toNullableNumber(value: string): number | null {
  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return null;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : null;
}

function initialScheduleType(job?: BackupJob): ScheduleType {
  if (job?.schedule.cron) {
    return "cron";
  }
  if (job?.schedule.one_time) {
    return "one_time";
  }
  return "interval";
}

const selectClassName =
  "w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50";

export function BackupJobForm({ mode, job }: BackupJobFormProps) {
  const router = useRouter();
  const references = useJobReferences();

  const [organization, setOrganization] = useState(job?.job_config.organization ?? "");
  const [bucketStore, setBucketStore] = useState(job?.job_config.bucket_store ?? "");
  const [sharePointSite, setSharePointSite] = useState(job?.job_config.share_point_site ?? "");
  const [documentLibraries, setDocumentLibraries] = useState(
    job?.job_config.filters.document_libraries ?? ""
  );
  const [minFileSize, setMinFileSize] = useState(
    job?.job_config.filters.min_file_size != null ? String(job.job_config.filters.min_file_size) : ""
  );
  const [maxFileSize, setMaxFileSize] = useState(
    job?.job_config.filters.max_file_size != null ? String(job.job_config.filters.max_file_size) : ""
  );
  const [createdAfter, setCreatedAfter] = useState(
    isoToLocalDateTime(job?.job_config.filters.created_after)
  );
  const [createdBefore, setCreatedBefore] = useState(
    isoToLocalDateTime(job?.job_config.filters.created_before)
  );
  const [updatedAfter, setUpdatedAfter] = useState(
    isoToLocalDateTime(job?.job_config.filters.updated_after)
  );
  const [updatedBefore, setUpdatedBefore] = useState(
    isoToLocalDateTime(job?.job_config.filters.updated_before)
  );

  const [scheduleType, setScheduleType] = useState<ScheduleType>(initialScheduleType(job));
  const [intervalSeconds, setIntervalSeconds] = useState(
    job?.schedule.interval != null ? String(job.schedule.interval) : ""
  );
  const [cron, setCron] = useState(job?.schedule.cron ?? "");
  const [oneTime, setOneTime] = useState(isoToLocalDateTime(job?.schedule.one_time));

  const [active, setActive] = useState(job?.active ?? true);
  const [startAt, setStartAt] = useState(isoToLocalDateTime(job?.start_at));
  const [endAt, setEndAt] = useState(isoToLocalDateTime(job?.end_at));
  const [runImmediately, setRunImmediately] = useState(false);

  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const buildSchedule = (): Schedule | null => {
    if (scheduleType === "interval") {
      const seconds = toNullableNumber(intervalSeconds);
      if (seconds == null || seconds <= 0) {
        setErrorMessage("Interval must be a positive number of seconds.");
        return null;
      }
      return { interval: seconds };
    }
    if (scheduleType === "cron") {
      if (cron.trim().length === 0) {
        setErrorMessage("A cron expression is required.");
        return null;
      }
      return { cron: cron.trim() };
    }
    const iso = localDateTimeToIso(oneTime);
    if (!iso) {
      setErrorMessage("A valid one-time run date is required.");
      return null;
    }
    return { one_time: iso };
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);

    const libraries = parseDocumentLibraries(documentLibraries);
    if (libraries.length === 0) {
      setErrorMessage("At least one document library is required.");
      return;
    }

    if (!organization) {
      setErrorMessage("An organization is required.");
      return;
    }
    if (!bucketStore) {
      setErrorMessage("A bucket store is required.");
      return;
    }

    const schedule = buildSchedule();
    if (!schedule) {
      return;
    }

    const payload: BackupJobInput = {
      active,
      start_at: localDateTimeToIso(startAt),
      end_at: localDateTimeToIso(endAt),
      schedule,
      job_config: {
        organization,
        bucket_store: bucketStore,
        share_point_site: sharePointSite.trim(),
        filters: {
          document_libraries: libraries,
          min_file_size: toNullableNumber(minFileSize),
          max_file_size: toNullableNumber(maxFileSize),
          created_after: localDateTimeToIso(createdAfter),
          created_before: localDateTimeToIso(createdBefore),
          updated_after: localDateTimeToIso(updatedAfter),
          updated_before: localDateTimeToIso(updatedBefore),
        },
      },
    };

    setIsSaving(true);
    try {
      if (mode === "create") {
        const created = await clientApiFetch<BackupJobResponse>("/backup-jobs", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        if (runImmediately) {
          await clientApiFetch<BackupRunStartResponse>(
            `/backup-jobs/${created.backup_job.id}/runs`,
            { method: "POST" }
          );
        }
        router.push("/dashboard/backup-job/list");
        router.refresh();
        return;
      }

      await clientApiFetch<BackupJobResponse>(`/backup-jobs/${job?.id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      setSuccessMessage("Backup job updated successfully.");
      router.refresh();
    } catch (error) {
      console.error("Backup job save failed.", error);
      setErrorMessage(toErrorMessage(error, "Failed to save backup job."));
    } finally {
      setIsSaving(false);
    }
  };

  const hasReferences = references.organizations.length > 0 && references.bucketStores.length > 0;

  return (
    <Card className="mx-auto max-w-5xl p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">
            {mode === "create" ? "Create Backup Job" : "Edit Backup Job"}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Back up a SharePoint site to a bucket store on a schedule.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href="/dashboard/backup-job/list">Back to List</Link>
        </Button>
      </div>

      {references.error ? (
        <p className="mb-4 text-sm text-destructive">{references.error}</p>
      ) : null}

      {!references.isLoading && !hasReferences ? (
        <p className="mb-4 text-sm text-muted-foreground">
          You need at least one{" "}
          <Link className="underline" href="/dashboard/organization/create">
            organization
          </Link>{" "}
          and one{" "}
          <Link className="underline" href="/dashboard/bucket-store/create">
            bucket store
          </Link>{" "}
          before creating a backup job.
        </p>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-8">
        <section className="space-y-4">
          <h2 className="text-lg font-semibold">Source &amp; destination</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="organization">Organization</Label>
              <select
                id="organization"
                className={selectClassName}
                value={organization}
                onChange={(event) => setOrganization(event.target.value)}
                required
              >
                <option value="">Select organization</option>
                {references.organizations.map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="bucketStore">Bucket Store</Label>
              <select
                id="bucketStore"
                className={selectClassName}
                value={bucketStore}
                onChange={(event) => setBucketStore(event.target.value)}
                required
              >
                <option value="">Select bucket store</option>
                {references.bucketStores.map((store) => (
                  <option key={store.id} value={store.id}>
                    {store.bucket_name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="sharePointSite">SharePoint Site URL</Label>
            <Input
              id="sharePointSite"
              value={sharePointSite}
              onChange={(event) => setSharePointSite(event.target.value)}
              placeholder="https://contoso.sharepoint.com/sites/example"
              required
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="documentLibraries">Document Libraries (comma separated)</Label>
            <textarea
              id="documentLibraries"
              className="min-h-24 w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
              value={documentLibraries}
              onChange={(event) => setDocumentLibraries(event.target.value)}
              required
            />
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-lg font-semibold">Schedule</h2>
          <div className="space-y-1.5">
            <Label htmlFor="scheduleType">Schedule Type</Label>
            <select
              id="scheduleType"
              className={selectClassName}
              value={scheduleType}
              onChange={(event) => setScheduleType(event.target.value as ScheduleType)}
            >
              <option value="interval">Interval (seconds)</option>
              <option value="cron">Cron expression</option>
              <option value="one_time">One-time</option>
            </select>
          </div>

          {scheduleType === "interval" ? (
            <div className="space-y-1.5">
              <Label htmlFor="intervalSeconds">Interval (seconds)</Label>
              <Input
                id="intervalSeconds"
                type="number"
                min={1}
                value={intervalSeconds}
                onChange={(event) => setIntervalSeconds(event.target.value)}
                placeholder="3600"
              />
            </div>
          ) : null}

          {scheduleType === "cron" ? (
            <div className="space-y-1.5">
              <Label htmlFor="cron">Cron Expression</Label>
              <Input
                id="cron"
                value={cron}
                onChange={(event) => setCron(event.target.value)}
                placeholder="0 2 * * *"
              />
            </div>
          ) : null}

          {scheduleType === "one_time" ? (
            <div className="space-y-1.5">
              <Label htmlFor="oneTime">Run At</Label>
              <Input
                id="oneTime"
                type="datetime-local"
                value={oneTime}
                onChange={(event) => setOneTime(event.target.value)}
              />
            </div>
          ) : null}

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="startAt">Active window start (optional)</Label>
              <Input
                id="startAt"
                type="datetime-local"
                value={startAt}
                onChange={(event) => setStartAt(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="endAt">Active window end (optional)</Label>
              <Input
                id="endAt"
                type="datetime-local"
                value={endAt}
                onChange={(event) => setEndAt(event.target.value)}
              />
            </div>
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={active}
              onChange={(event) => setActive(event.target.checked)}
            />
            Job is active
          </label>

          {mode === "create" ? (
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={runImmediately}
                onChange={(event) => setRunImmediately(event.target.checked)}
              />
              Run once immediately after creating
            </label>
          ) : null}
        </section>

        <section className="space-y-4">
          <h2 className="text-lg font-semibold">Filters (optional)</h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="minFileSize">Min File Size (bytes)</Label>
              <Input
                id="minFileSize"
                type="number"
                min={0}
                value={minFileSize}
                onChange={(event) => setMinFileSize(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="maxFileSize">Max File Size (bytes)</Label>
              <Input
                id="maxFileSize"
                type="number"
                min={0}
                value={maxFileSize}
                onChange={(event) => setMaxFileSize(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="createdAfter">Created After</Label>
              <Input
                id="createdAfter"
                type="datetime-local"
                value={createdAfter}
                onChange={(event) => setCreatedAfter(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="createdBefore">Created Before</Label>
              <Input
                id="createdBefore"
                type="datetime-local"
                value={createdBefore}
                onChange={(event) => setCreatedBefore(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="updatedAfter">Updated After</Label>
              <Input
                id="updatedAfter"
                type="datetime-local"
                value={updatedAfter}
                onChange={(event) => setUpdatedAfter(event.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="updatedBefore">Updated Before</Label>
              <Input
                id="updatedBefore"
                type="datetime-local"
                value={updatedBefore}
                onChange={(event) => setUpdatedBefore(event.target.value)}
              />
            </div>
          </div>
        </section>

        <div className="flex items-center gap-3">
          <Button type="submit" disabled={isSaving || (mode === "create" && !hasReferences)}>
            {isSaving ? "Saving..." : mode === "create" ? "Create Backup Job" : "Save"}
          </Button>
        </div>

        {successMessage ? (
          <p className="text-sm text-emerald-600 dark:text-emerald-500">{successMessage}</p>
        ) : null}
        {errorMessage ? <p className="text-sm text-destructive">{errorMessage}</p> : null}
      </form>
    </Card>
  );
}
