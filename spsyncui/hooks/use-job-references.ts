"use client";

import { useEffect, useState } from "react";

import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import type {
  BucketStore,
  BucketStoresResponse,
  Organization,
  OrganizationsResponse,
} from "@/lib/api/types";

type JobReferences = {
  organizations: Organization[];
  bucketStores: BucketStore[];
  isLoading: boolean;
  error: string | null;
};

/**
 * Loads the organizations and bucket stores needed to populate the reference
 * dropdowns on backup/restore job forms.
 */
export function useJobReferences(): JobReferences {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [bucketStores, setBucketStores] = useState<BucketStore[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const [orgs, stores] = await Promise.all([
          clientApiFetch<OrganizationsResponse>("/organizations"),
          clientApiFetch<BucketStoresResponse>("/bucket-stores"),
        ]);
        if (cancelled) {
          return;
        }
        setOrganizations(orgs.organizations ?? []);
        setBucketStores(stores.bucket_stores ?? []);
      } catch (loadError) {
        if (!cancelled) {
          setError(toErrorMessage(loadError, "Failed to load organizations and bucket stores."));
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  return { organizations, bucketStores, isLoading, error };
}
