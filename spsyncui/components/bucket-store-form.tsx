"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import type {
  BucketStore,
  BucketStoreInput,
  BucketStoreResponse,
  BucketType,
} from "@/lib/api/types";

type BucketStoreFormProps = {
  mode: "create" | "edit";
  bucketStore?: BucketStore;
};

export function BucketStoreForm({ mode, bucketStore }: BucketStoreFormProps) {
  const router = useRouter();
  const [bucketName, setBucketName] = useState(bucketStore?.bucket_name ?? "");
  const [bucketType, setBucketType] = useState<BucketType>(bucketStore?.bucket_type ?? "s3");
  const [server, setServer] = useState("");
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [connectionString, setConnectionString] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const buildConfig = (): BucketStoreInput["config"] | undefined => {
    if (bucketType === "s3") {
      if (!server && !accessKey && !secretKey) {
        return undefined;
      }
      return {
        server: server.trim(),
        access_key: accessKey.trim(),
        secret_key: secretKey.trim(),
      };
    }
    if (!connectionString) {
      return undefined;
    }
    return { connection_string: connectionString.trim() };
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);

    const config = buildConfig();
    if (mode === "create" && !config) {
      setErrorMessage("Connection details are required when creating a bucket store.");
      return;
    }

    const payload: BucketStoreInput = {
      bucket_name: bucketName.trim(),
      bucket_type: bucketType,
    };
    if (config) {
      payload.config = config;
    }

    setIsSaving(true);
    try {
      if (mode === "create") {
        await clientApiFetch<BucketStoreResponse>("/bucket-stores", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        router.push("/dashboard/bucket-store/list");
        router.refresh();
        return;
      }

      await clientApiFetch<BucketStoreResponse>(`/bucket-stores/${bucketStore?.id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      setServer("");
      setAccessKey("");
      setSecretKey("");
      setConnectionString("");
      setSuccessMessage("Bucket store updated successfully.");
      router.refresh();
    } catch (error) {
      console.error("Bucket store save failed.", error);
      setErrorMessage(toErrorMessage(error, "Failed to save bucket store."));
    } finally {
      setIsSaving(false);
    }
  };

  const configPlaceholder = mode === "edit" ? "Leave blank to keep current value" : undefined;

  return (
    <Card className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">
            {mode === "create" ? "Create Bucket Store" : "Edit Bucket Store"}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Object storage destination for backups and source for restores.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href="/dashboard/bucket-store/list">Back to List</Link>
        </Button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="bucketName">Bucket Name</Label>
          <Input
            id="bucketName"
            value={bucketName}
            onChange={(event) => setBucketName(event.target.value)}
            required
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="bucketType">Bucket Type</Label>
          <select
            id="bucketType"
            className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
            value={bucketType}
            onChange={(event) => setBucketType(event.target.value as BucketType)}
          >
            <option value="s3">S3</option>
            <option value="azure">Azure Blob</option>
          </select>
        </div>

        {bucketType === "s3" ? (
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-1.5 md:col-span-2">
              <Label htmlFor="server">Server</Label>
              <Input
                id="server"
                value={server}
                onChange={(event) => setServer(event.target.value)}
                placeholder={configPlaceholder ?? "https://s3.example.com"}
                required={mode === "create"}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="accessKey">Access Key</Label>
              <Input
                id="accessKey"
                value={accessKey}
                onChange={(event) => setAccessKey(event.target.value)}
                placeholder={configPlaceholder}
                required={mode === "create"}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="secretKey">Secret Key</Label>
              <Input
                id="secretKey"
                type="password"
                value={secretKey}
                onChange={(event) => setSecretKey(event.target.value)}
                placeholder={configPlaceholder}
                required={mode === "create"}
              />
            </div>
          </div>
        ) : (
          <div className="space-y-1.5">
            <Label htmlFor="connectionString">Connection String</Label>
            <Input
              id="connectionString"
              value={connectionString}
              onChange={(event) => setConnectionString(event.target.value)}
              placeholder={configPlaceholder ?? "DefaultEndpointsProtocol=https;AccountName=..."}
              required={mode === "create"}
            />
          </div>
        )}

        <p className="text-xs text-muted-foreground">
          Connection details are stored encrypted by the API and are never returned.
        </p>

        <div className="flex items-center gap-3 pt-2">
          <Button type="submit" disabled={isSaving}>
            {isSaving ? "Saving..." : mode === "create" ? "Create" : "Save"}
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
