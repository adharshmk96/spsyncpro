"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { formatBucketType, formatDateTime } from "@/lib/api/format";
import type { BucketStore, BucketStoresResponse } from "@/lib/api/types";

export default function DashboardBucketStoreListPage() {
  const [bucketStores, setBucketStores] = useState<BucketStore[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<BucketStoresResponse>("/bucket-stores");
        if (!active) return;
        setBucketStores(data.bucket_stores ?? []);
      } catch (error) {
        console.error("Failed to load bucket stores.", error);
        if (active) setErrorMessage(toErrorMessage(error, "Failed to load bucket stores."));
      } finally {
        if (active) setIsLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  const handleDelete = async (id: string) => {
    if (!window.confirm("Delete this bucket store? This cannot be undone.")) {
      return;
    }
    setDeletingId(id);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/bucket-stores/${id}`, { method: "DELETE" });
      setBucketStores((current) => current.filter((store) => store.id !== id));
    } catch (error) {
      console.error("Failed to delete bucket store.", error);
      setErrorMessage(toErrorMessage(error, "Failed to delete bucket store."));
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Bucket Stores</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Object storage backends used by backup and restore jobs.
          </p>
        </div>
        <Button asChild>
          <Link href="/dashboard/bucket-store/create">Create Bucket Store</Link>
        </Button>
      </div>

      {errorMessage ? <p className="mb-4 text-sm text-destructive">{errorMessage}</p> : null}

      {isLoading ? (
        <Card className="p-4 text-sm text-muted-foreground">Loading bucket stores...</Card>
      ) : bucketStores.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No bucket stores created yet.</Card>
      ) : (
        <div className="grid gap-4">
          {bucketStores.map((store) => (
            <Card key={store.id} className="p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                  <p className="font-semibold">{store.bucket_name}</p>
                  <p className="text-sm text-muted-foreground">
                    Type: {formatBucketType(store.bucket_type)}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Created: {formatDateTime(store.created_at)} · Updated:{" "}
                    {formatDateTime(store.updated_at)}
                  </p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button asChild variant="outline" size="sm">
                    <Link href={`/dashboard/bucket-store/${store.id}`}>Edit</Link>
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => handleDelete(store.id)}
                    disabled={deletingId === store.id}
                  >
                    {deletingId === store.id ? "Deleting..." : "Delete"}
                  </Button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </main>
  );
}
