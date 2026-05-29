import Link from "next/link";

import { BucketStoreForm } from "@/components/bucket-store-form";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { serverApiFetch } from "@/lib/api/server";
import type { BucketStoreResponse } from "@/lib/api/types";

export default async function DashboardBucketStoreEditPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let bucketStore: BucketStoreResponse["bucket_store"] | null = null;
  try {
    bucketStore = (await serverApiFetch<BucketStoreResponse>(`/bucket-stores/${id}`)).bucket_store;
  } catch (error) {
    console.error("Failed to load bucket store.", error);
  }

  if (!bucketStore) {
    return (
      <main className="p-6">
        <Card className="mx-auto max-w-3xl p-6">
          <p className="text-sm text-destructive">Bucket store not found.</p>
          <Button asChild variant="outline" className="mt-4">
            <Link href="/dashboard/bucket-store/list">Back to list</Link>
          </Button>
        </Card>
      </main>
    );
  }

  return (
    <main className="p-6">
      <BucketStoreForm mode="edit" bucketStore={bucketStore} />
    </main>
  );
}
