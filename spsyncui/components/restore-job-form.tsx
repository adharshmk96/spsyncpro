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
import { isoToLocalDateTime, localDateTimeToIso } from "@/lib/api/format";
import type {
  RestoreJob,
  RestoreJobInput,
  RestoreJobResponse,
  RestoreRunStartResponse,
} from "@/lib/api/types";

type RestoreJobFormProps = {
  mode: "create" | "edit";
  job?: RestoreJob;
};

const selectClassName =
  "w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50";

export function RestoreJobForm({ mode, job }: RestoreJobFormProps) {
  const router = useRouter();
  const references = useJobReferences();

  const [organization, setOrganization] = useState(job?.job_config.organization ?? "");
  const [bucketStore, setBucketStore] = useState(job?.job_config.bucket_store ?? "");
  const [sharePointSite, setSharePointSite] = useState(job?.job_config.share_point_site ?? "");
  const [startAt, setStartAt] = useState(isoToLocalDateTime(job?.start_at));
  const [active, setActive] = useState(job?.active ?? true);
  const [runImmediately, setRunImmediately] = useState(false);

  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);

    if (!organization) {
      setErrorMessage("An organization is required.");
      return;
    }
    if (!bucketStore) {
      setErrorMessage("A bucket store is required.");
      return;
    }

    const payload: RestoreJobInput = {
      active,
      start_at: localDateTimeToIso(startAt),
      job_config: {
        organization,
        bucket_store: bucketStore,
        share_point_site: sharePointSite.trim(),
      },
    };

    setIsSaving(true);
    try {
      if (mode === "create") {
        const created = await clientApiFetch<RestoreJobResponse>("/restore-jobs", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        if (runImmediately) {
          await clientApiFetch<RestoreRunStartResponse>(
            `/restore-jobs/${created.restore_job.id}/runs`,
            { method: "POST" }
          );
        }
        router.push("/dashboard/restore-job/list");
        router.refresh();
        return;
      }

      await clientApiFetch<RestoreJobResponse>(`/restore-jobs/${job?.id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      setSuccessMessage("Restore job updated successfully.");
      router.refresh();
    } catch (error) {
      console.error("Restore job save failed.", error);
      setErrorMessage(toErrorMessage(error, "Failed to save restore job."));
    } finally {
      setIsSaving(false);
    }
  };

  const hasReferences = references.organizations.length > 0 && references.bucketStores.length > 0;

  return (
    <Card className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">
            {mode === "create" ? "Create Restore Job" : "Edit Restore Job"}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Restore files from a bucket store into a SharePoint site.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href="/dashboard/restore-job/list">Back to List</Link>
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
          before creating a restore job.
        </p>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
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
          <Label htmlFor="startAt">Scheduled start (optional, must be in the future)</Label>
          <Input
            id="startAt"
            type="datetime-local"
            value={startAt}
            onChange={(event) => setStartAt(event.target.value)}
          />
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

        <div className="flex items-center gap-3 pt-2">
          <Button type="submit" disabled={isSaving || (mode === "create" && !hasReferences)}>
            {isSaving ? "Saving..." : mode === "create" ? "Create Restore Job" : "Save"}
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
