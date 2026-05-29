"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { clientApiFetch } from "@/lib/api/client";
import { toErrorMessage } from "@/lib/api/errors";
import type { Organization, OrganizationInput, OrganizationResponse } from "@/lib/api/types";

type OrganizationFormProps = {
  mode: "create" | "edit";
  organization?: Organization;
};

export function OrganizationForm({ mode, organization }: OrganizationFormProps) {
  const router = useRouter();
  const [name, setName] = useState(organization?.name ?? "");
  const [tenantId, setTenantId] = useState(organization?.tenant_id ?? "");
  const [clientId, setClientId] = useState(organization?.client_id ?? "");
  const [tenantSecret, setTenantSecret] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const authorizeUrl = useMemo(() => {
    if (!tenantId || !clientId) {
      return "";
    }
    return `https://login.microsoftonline.com/${tenantId}/adminconsent?client_id=${clientId}`;
  }, [tenantId, clientId]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);

    if (mode === "create" && tenantSecret.trim().length === 0) {
      setErrorMessage("Tenant secret is required when creating an organization.");
      return;
    }

    const payload: OrganizationInput = {
      name: name.trim(),
      tenant_id: tenantId.trim(),
      client_id: clientId.trim(),
    };
    if (tenantSecret.trim().length > 0) {
      payload.tenant_secret = tenantSecret;
    }

    setIsSaving(true);
    try {
      if (mode === "create") {
        await clientApiFetch<OrganizationResponse>("/organizations", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        router.push("/dashboard/organization/list");
        router.refresh();
        return;
      }

      await clientApiFetch<OrganizationResponse>(`/organizations/${organization?.id}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      setTenantSecret("");
      setSuccessMessage("Organization updated successfully.");
      router.refresh();
    } catch (error) {
      console.error("Organization save failed.", error);
      setErrorMessage(toErrorMessage(error, "Failed to save organization."));
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Card className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">
            {mode === "create" ? "Create Organization" : "Edit Organization"}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Configure SharePoint tenant and app registration details.
          </p>
        </div>
        <Button asChild variant="outline">
          <Link href="/dashboard/organization/list">Back to List</Link>
        </Button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="name">Name</Label>
          <Input id="name" value={name} onChange={(event) => setName(event.target.value)} required />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="tenantId">Tenant ID</Label>
          <Input
            id="tenantId"
            value={tenantId}
            onChange={(event) => setTenantId(event.target.value)}
            required
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="clientId">Client ID</Label>
          <Input
            id="clientId"
            value={clientId}
            onChange={(event) => setClientId(event.target.value)}
            required
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="tenantSecret">Tenant Secret</Label>
          <Input
            id="tenantSecret"
            type="password"
            value={tenantSecret}
            onChange={(event) => setTenantSecret(event.target.value)}
            placeholder={
              mode === "edit" ? "Leave blank to keep current secret" : "Enter tenant secret"
            }
            required={mode === "create"}
          />
          <p className="text-xs text-muted-foreground">
            The secret is stored encrypted by the API and is never returned.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3 pt-2">
          {authorizeUrl ? (
            <Button asChild variant="outline">
              <a href={authorizeUrl} target="_blank" rel="noreferrer">
                Authorize
              </a>
            </Button>
          ) : null}
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
