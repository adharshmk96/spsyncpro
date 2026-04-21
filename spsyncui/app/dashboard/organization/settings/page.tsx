"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type OrganizationSettingsDto = {
  id: string;
  name: string;
  description: string;
  tenantId: string;
  clientId: string;
  hasClientSecret: boolean;
};

type OrganizationFormState = {
  name: string;
  description: string;
  tenantId: string;
  clientId: string;
};

const INITIAL_FORM_STATE: OrganizationFormState = {
  name: "",
  description: "",
  tenantId: "",
  clientId: "",
};

export default function DashboardOrganizationSettingsPage() {
  const [formState, setFormState] = useState<OrganizationFormState>(INITIAL_FORM_STATE);
  const [clientSecret, setClientSecret] = useState("");
  const [hasClientSecret, setHasClientSecret] = useState(false);
  const [isEditingClientSecret, setIsEditingClientSecret] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const authorizeUrl = useMemo(() => {
    if (!formState.tenantId || !formState.clientId) {
      return "";
    }

    return `https://login.microsoftonline.com/${formState.tenantId}/adminconsent?client_id=${formState.clientId}`;
  }, [formState.clientId, formState.tenantId]);

  useEffect(() => {
    const loadOrganization = async () => {
      setIsLoading(true);
      setErrorMessage(null);

      try {
        const response = await fetch("/api/organization", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
          },
        });

        const data = (await response.json()) as
          | { organization: OrganizationSettingsDto }
          | { error: string };

        if (!response.ok) {
          throw new Error("error" in data ? data.error : "Failed to load organization.");
        }

        if ("organization" in data) {
          setFormState({
            name: data.organization.name,
            description: data.organization.description,
            tenantId: data.organization.tenantId,
            clientId: data.organization.clientId,
          });
          setHasClientSecret(data.organization.hasClientSecret);
          setIsEditingClientSecret(!data.organization.hasClientSecret);
        }
      } catch (error) {
        console.error("Failed to fetch organization settings.", error);
        setErrorMessage(
          error instanceof Error ? error.message : "Failed to load organization settings."
        );
      } finally {
        setIsLoading(false);
      }
    };

    void loadOrganization();
  }, []);

  const updateField = (field: keyof OrganizationFormState, value: string) => {
    setFormState((current) => ({
      ...current,
      [field]: value,
    }));
  };

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);
    setIsSaving(true);

    try {
      const response = await fetch("/api/organization", {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(formState),
      });

      const data = (await response.json()) as
        | { organization: OrganizationSettingsDto }
        | { error: string };

      if (!response.ok) {
        throw new Error("error" in data ? data.error : "Failed to save organization settings.");
      }

      if ("organization" in data) {
        setFormState({
          name: data.organization.name,
          description: data.organization.description,
          tenantId: data.organization.tenantId,
          clientId: data.organization.clientId,
        });
        const hasSavedSecret = data.organization.hasClientSecret;
        setHasClientSecret(hasSavedSecret);

        // Update secret only when user explicitly enabled editing and provided a value.
        if (isEditingClientSecret && clientSecret.trim().length > 0) {
          const secretResponse = await fetch("/api/organization/client-secret", {
            method: "PUT",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ clientSecret }),
          });

          const secretData = (await secretResponse.json()) as { ok?: boolean; error?: string };

          if (!secretResponse.ok) {
            throw new Error(secretData.error ?? "Failed to update client secret.");
          }

          setHasClientSecret(true);
          setClientSecret("");
          setIsEditingClientSecret(false);
        } else if (!hasSavedSecret) {
          setIsEditingClientSecret(true);
        }
      }

      setSuccessMessage("Organization settings saved successfully.");
    } catch (error) {
      console.error("Organization save failed.", error);
      setErrorMessage(
        error instanceof Error ? error.message : "Failed to save organization settings."
      );
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <main className="p-6">
      <Card className="mx-auto max-w-3xl p-6">
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">Organization Settings</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Configure organization details and SharePoint app settings.
          </p>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading settings...</p>
        ) : (
          <form onSubmit={handleSave} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={formState.name}
                onChange={(event) => updateField("name", event.target.value)}
                required
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="description">Description</Label>
              <textarea
                id="description"
                className="min-h-24 w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={formState.description}
                onChange={(event) => updateField("description", event.target.value)}
                required
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="tenantId">Tenant ID</Label>
              <Input
                id="tenantId"
                value={formState.tenantId}
                onChange={(event) => updateField("tenantId", event.target.value)}
                required
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="clientId">Client ID</Label>
              <Input
                id="clientId"
                value={formState.clientId}
                onChange={(event) => updateField("clientId", event.target.value)}
                required
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between gap-2">
                <Label htmlFor="clientSecret">Client Secret</Label>
                {hasClientSecret ? (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() =>
                      setIsEditingClientSecret((current) => {
                        const nextValue = !current;
                        if (!nextValue) {
                          setClientSecret("");
                        }
                        return nextValue;
                      })
                    }
                  >
                    {isEditingClientSecret ? "Cancel Secret Update" : "Update Secret"}
                  </Button>
                ) : null}
              </div>
              <Input
                id="clientSecret"
                type="password"
                value={
                  isEditingClientSecret || !hasClientSecret ? clientSecret : "••••••••••••••••"
                }
                onChange={(event) => setClientSecret(event.target.value)}
                disabled={hasClientSecret && !isEditingClientSecret}
                placeholder={
                  hasClientSecret && !isEditingClientSecret
                    ? "Client secret already set"
                    : "Enter client secret"
                }
              />
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
                {isSaving ? "Saving..." : "Save"}
              </Button>
            </div>

            {successMessage ? (
              <p className="text-sm text-emerald-600 dark:text-emerald-500">{successMessage}</p>
            ) : null}

            {errorMessage ? (
              <p className="text-sm text-destructive">{errorMessage}</p>
            ) : null}
          </form>
        )}
      </Card>
    </main>
  );
}
