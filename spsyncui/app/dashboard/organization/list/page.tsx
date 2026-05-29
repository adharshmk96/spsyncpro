"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import { formatDateTime } from "@/lib/api/format";
import type { Organization, OrganizationsResponse } from "@/lib/api/types";

export default function DashboardOrganizationListPage() {
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    void (async () => {
      try {
        const data = await clientApiFetch<OrganizationsResponse>("/organizations");
        if (!active) return;
        setOrganizations(data.organizations ?? []);
      } catch (error) {
        console.error("Failed to load organizations.", error);
        if (active) setErrorMessage(toErrorMessage(error, "Failed to load organizations."));
      } finally {
        if (active) setIsLoading(false);
      }
    })();
    return () => {
      active = false;
    };
  }, []);

  const handleDelete = async (id: string) => {
    if (!window.confirm("Delete this organization? This cannot be undone.")) {
      return;
    }
    setDeletingId(id);
    setErrorMessage(null);
    try {
      await clientApiFetch(`/organizations/${id}`, { method: "DELETE" });
      setOrganizations((current) => current.filter((organization) => organization.id !== id));
    } catch (error) {
      console.error("Failed to delete organization.", error);
      setErrorMessage(toErrorMessage(error, "Failed to delete organization."));
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <main className="p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Organizations</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            SharePoint tenants connected to SPSyncPro.
          </p>
        </div>
        <Button asChild>
          <Link href="/dashboard/organization/create">Create Organization</Link>
        </Button>
      </div>

      {errorMessage ? <p className="mb-4 text-sm text-destructive">{errorMessage}</p> : null}

      {isLoading ? (
        <Card className="p-4 text-sm text-muted-foreground">Loading organizations...</Card>
      ) : organizations.length === 0 ? (
        <Card className="p-4 text-sm text-muted-foreground">No organizations created yet.</Card>
      ) : (
        <div className="grid gap-4">
          {organizations.map((organization) => (
            <Card key={organization.id} className="p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                  <p className="font-semibold">{organization.name}</p>
                  <p className="text-sm text-muted-foreground">Tenant ID: {organization.tenant_id}</p>
                  <p className="text-sm text-muted-foreground">Client ID: {organization.client_id}</p>
                  <p className="text-sm text-muted-foreground">
                    Created: {formatDateTime(organization.created_at)} · Updated:{" "}
                    {formatDateTime(organization.updated_at)}
                  </p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button asChild variant="outline" size="sm">
                    <Link href={`/dashboard/organization/${organization.id}`}>Edit</Link>
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => handleDelete(organization.id)}
                    disabled={deletingId === organization.id}
                  >
                    {deletingId === organization.id ? "Deleting..." : "Delete"}
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
