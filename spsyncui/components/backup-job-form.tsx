"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useJobReferences } from "@/hooks/use-job-references";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import {
  frequencyToSeconds,
  hasOptionalBackupFilters,
  isoToLocalDateTime,
  localDateTimeToIso,
  parseDocumentLibraries,
  secondsToFrequency,
  type FrequencyUnit,
} from "@/lib/api/format";
import type {
  BackupJob,
  BackupJobInput,
  BackupJobResponse,
  BackupRunStartResponse,
  ScheduleInput,
} from "@/lib/api/types";

type ScheduleType = "one_time" | "recurring";

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
  if (job?.schedule.interval != null) {
    return "recurring";
  }
  if (job?.schedule.cron) {
    return "recurring";
  }
  return "one_time";
}

function initialFrequency(job?: BackupJob): { value: string; unit: FrequencyUnit } {
  if (job?.schedule.interval != null) {
    const { value, unit } = secondsToFrequency(job.schedule.interval);
    return { value: String(value), unit };
  }
  return { value: "1", unit: "hour" };
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

  const [filtersEnabled, setFiltersEnabled] = useState(
    job ? hasOptionalBackupFilters(job.job_config.filters) : false
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
  const initialFreq = initialFrequency(job);
  const [frequencyValue, setFrequencyValue] = useState(initialFreq.value);
  const [frequencyUnit, setFrequencyUnit] = useState<FrequencyUnit>(initialFreq.unit);
  const [oneTimeStartAt, setOneTimeStartAt] = useState(isoToLocalDateTime(job?.schedule.one_time));
  const [recurringStartAt, setRecurringStartAt] = useState(isoToLocalDateTime(job?.start_at));
  const [runNow, setRunNow] = useState(false);

  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const hasLegacyCron = Boolean(job?.schedule.cron);

  const buildSchedule = (): ScheduleInput | null => {
    if (scheduleType === "one_time") {
      const iso = localDateTimeToIso(oneTimeStartAt);
      if (oneTimeStartAt && !iso) {
        setErrorMessage("A valid start date is required.");
        return null;
      }
      return {
        type: "one_time",
        one_time: iso,
      };
    }

    const freq = toNullableNumber(frequencyValue);
    if (freq == null || freq <= 0) {
      setErrorMessage("Frequency must be a positive number.");
      return null;
    }
    const interval = frequencyToSeconds(freq, frequencyUnit);
    if (interval <= 0) {
      setErrorMessage("Frequency must be a positive number.");
      return null;
    }
    return {
      type: "recurring",
      interval,
    };
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
      active: true,
      start_at: scheduleType === "recurring" ? localDateTimeToIso(recurringStartAt) : null,
      end_at: null,
      schedule,
      job_config: {
        organization,
        bucket_store: bucketStore,
        share_point_site: sharePointSite.trim(),
        filters: {
          document_libraries: libraries,
          min_file_size: filtersEnabled ? toNullableNumber(minFileSize) : null,
          max_file_size: filtersEnabled ? toNullableNumber(maxFileSize) : null,
          created_after: filtersEnabled ? localDateTimeToIso(createdAfter) : null,
          created_before: filtersEnabled ? localDateTimeToIso(createdBefore) : null,
          updated_after: filtersEnabled ? localDateTimeToIso(updatedAfter) : null,
          updated_before: filtersEnabled ? localDateTimeToIso(updatedBefore) : null,
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
        if (scheduleType === "recurring" && runNow) {
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
              <option value="one_time">One-time</option>
              <option value="recurring">Recurring</option>
            </select>
          </div>

          {hasLegacyCron && scheduleType === "recurring" ? (
            <p className="text-sm text-muted-foreground">
              This job uses a legacy cron schedule ({job?.schedule.cron}). Saving will replace it
              with the interval below.
            </p>
          ) : null}

          {scheduleType === "one_time" ? (
            <div className="space-y-1.5">
              <Label htmlFor="oneTimeStartAt">Start at (optional)</Label>
              <Input
                id="oneTimeStartAt"
                type="datetime-local"
                value={oneTimeStartAt}
                onChange={(event) => setOneTimeStartAt(event.target.value)}
              />
              <p className="text-sm text-muted-foreground">Runs immediately if left blank.</p>
            </div>
          ) : (
            <>
              <div className="space-y-1.5">
                <Label htmlFor="recurringStartAt">Start at (optional)</Label>
                <Input
                  id="recurringStartAt"
                  type="datetime-local"
                  value={recurringStartAt}
                  onChange={(event) => setRecurringStartAt(event.target.value)}
                />
                <p className="text-sm text-muted-foreground">
                  When the recurring schedule should begin. Leave blank to start immediately.
                </p>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor="frequencyValue">Frequency</Label>
                  <Input
                    id="frequencyValue"
                    type="number"
                    min={1}
                    value={frequencyValue}
                    onChange={(event) => setFrequencyValue(event.target.value)}
                    placeholder="1"
                    required
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="frequencyUnit">Unit</Label>
                  <select
                    id="frequencyUnit"
                    className={selectClassName}
                    value={frequencyUnit}
                    onChange={(event) => setFrequencyUnit(event.target.value as FrequencyUnit)}
                  >
                    <option value="minute">Minute(s)</option>
                    <option value="hour">Hour(s)</option>
                    <option value="day">Day(s)</option>
                  </select>
                </div>
              </div>
              {mode === "create" ? (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={runNow}
                    onChange={(event) => setRunNow(event.target.checked)}
                  />
                  Run now (one run immediately after creating)
                </label>
              ) : null}
            </>
          )}
        </section>

        <section className="space-y-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">Filters</h2>
            <div className="flex items-center gap-2">
              <Label htmlFor="filtersEnabled" className="text-sm font-normal">
                Enable filters
              </Label>
              <Switch
                id="filtersEnabled"
                checked={filtersEnabled}
                onCheckedChange={setFiltersEnabled}
              />
            </div>
          </div>
          {filtersEnabled ? (
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
          ) : (
            <p className="text-sm text-muted-foreground">
              Optional file size and date filters are disabled.
            </p>
          )}
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
