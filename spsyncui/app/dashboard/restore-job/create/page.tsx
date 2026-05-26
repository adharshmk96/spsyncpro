"use client";

import Link from "next/link";
import { FormEvent, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type StorageType = "AZURE_BLOB" | "AWS_S3";
type RunMode = "IMMEDIATE" | "ONE_TIME_AT";

type RestoreJobFormState = {
  siteUrl: string;
  documentLibraryList: string;
  storageType: StorageType;
  storageConfig: {
    azureBlobConfig: {
      connectionString: string;
      containerName: string;
    };
    awsS3Config: {
      region: string;
      bucketName: string;
      accessKeyID: string;
      secretAccessKey: string;
    };
  };
  filterConfig: {
    minFileSize: string;
    maxFileSize: string;
    createdBefore: string;
    createdAfter: string;
    modifiedBefore: string;
    modifiedAfter: string;
  };
  runMode: RunMode;
  startAt: string;
};

const INITIAL_FORM_STATE: RestoreJobFormState = {
  siteUrl: "",
  documentLibraryList: "",
  storageType: "AZURE_BLOB",
  storageConfig: {
    azureBlobConfig: {
      connectionString: "",
      containerName: "",
    },
    awsS3Config: {
      region: "",
      bucketName: "",
      accessKeyID: "",
      secretAccessKey: "",
    },
  },
  filterConfig: {
    minFileSize: "0",
    maxFileSize: "0",
    createdBefore: "",
    createdAfter: "",
    modifiedBefore: "",
    modifiedAfter: "",
  },
  runMode: "IMMEDIATE",
  startAt: "",
};

function localDateTimeToIso(localDateTime: string): string | undefined {
  if (!localDateTime) {
    return undefined;
  }

  const date = new Date(localDateTime);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toISOString();
}

export default function DashboardRestoreJobCreatePage() {
  const [formState, setFormState] = useState<RestoreJobFormState>(INITIAL_FORM_STATE);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const runSummary = useMemo(() => {
    if (formState.runMode === "IMMEDIATE") {
      return "Runs immediately after creation. Restore jobs are one-time only.";
    }

    return formState.startAt
      ? `Runs once at ${formState.startAt}. Restore jobs are one-time only.`
      : "Select a run time for this one-time restore.";
  }, [formState.runMode, formState.startAt]);

  const setField = <K extends keyof RestoreJobFormState>(
    field: K,
    value: RestoreJobFormState[K]
  ) => {
    setFormState((current) => ({ ...current, [field]: value }));
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);

    const documentLibraryList = formState.documentLibraryList
      .split(",")
      .map((entry) => entry.trim())
      .filter((entry) => entry.length > 0);

    if (documentLibraryList.length === 0) {
      setErrorMessage("At least one document library is required.");
      return;
    }

    if (formState.runMode === "ONE_TIME_AT" && !formState.startAt) {
      setErrorMessage("Run time is required for a scheduled one-time restore.");
      return;
    }

    const payload = {
      siteUrl: formState.siteUrl.trim(),
      documentLibraryList,
      storageType: formState.storageType,
      storageConfig:
        formState.storageType === "AZURE_BLOB"
          ? {
              azureBlobConfig: {
                connectionString: formState.storageConfig.azureBlobConfig.connectionString.trim(),
                containerName: formState.storageConfig.azureBlobConfig.containerName.trim(),
              },
            }
          : {
              awsS3Config: {
                region: formState.storageConfig.awsS3Config.region.trim(),
                bucketName: formState.storageConfig.awsS3Config.bucketName.trim(),
                accessKeyID: formState.storageConfig.awsS3Config.accessKeyID.trim(),
                secretAccessKey: formState.storageConfig.awsS3Config.secretAccessKey.trim(),
              },
            },
      filterConfig: {
        minFileSize: Number(formState.filterConfig.minFileSize),
        maxFileSize: Number(formState.filterConfig.maxFileSize),
        createdBefore: localDateTimeToIso(formState.filterConfig.createdBefore),
        createdAfter: localDateTimeToIso(formState.filterConfig.createdAfter),
        modifiedBefore: localDateTimeToIso(formState.filterConfig.modifiedBefore),
        modifiedAfter: localDateTimeToIso(formState.filterConfig.modifiedAfter),
      },
      runMode: formState.runMode,
      startAt:
        formState.runMode === "IMMEDIATE" ? undefined : localDateTimeToIso(formState.startAt),
    };

    setIsSaving(true);
    try {
      console.info("Restore job payload (placeholder):", payload);
      setSuccessMessage("Restore job created successfully.");
      setFormState(INITIAL_FORM_STATE);
    } catch (error) {
      console.error("Restore job creation failed.", error);
      setErrorMessage(error instanceof Error ? error.message : "Failed to create restore job.");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <main className="p-6">
      <Card className="mx-auto max-w-5xl p-6">
        <div className="mb-6 flex items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold">Create Restore Job</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Restore files from a bucket into a SharePoint site. One-time runs only — no
              recurring schedules.
            </p>
          </div>
          <Button asChild variant="outline">
            <Link href="/dashboard/restore-job/list">Back to List</Link>
          </Button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-8">
          <section className="space-y-4">
            <h2 className="text-lg font-semibold">Destination (SharePoint)</h2>
            <div className="space-y-1.5">
              <Label htmlFor="siteUrl">Site URL</Label>
              <Input
                id="siteUrl"
                value={formState.siteUrl}
                onChange={(event) => setField("siteUrl", event.target.value)}
                placeholder="https://contoso.sharepoint.com/sites/example"
                required
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="documentLibraryList">Document Library List (comma separated)</Label>
              <textarea
                id="documentLibraryList"
                className="min-h-24 w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={formState.documentLibraryList}
                onChange={(event) => setField("documentLibraryList", event.target.value)}
                required
              />
            </div>
          </section>

          <section className="space-y-4">
            <h2 className="text-lg font-semibold">Source storage (bucket)</h2>
            <p className="text-sm text-muted-foreground">
              Files are read from this bucket and written to the SharePoint site above.
            </p>
            <div className="space-y-1.5">
              <Label htmlFor="storageType">Storage Type</Label>
              <select
                id="storageType"
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={formState.storageType}
                onChange={(event) => setField("storageType", event.target.value as StorageType)}
              >
                <option value="AZURE_BLOB">Azure Blob</option>
                <option value="AWS_S3">AWS S3</option>
              </select>
            </div>

            {formState.storageType === "AZURE_BLOB" ? (
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor="connectionString">Connection String</Label>
                  <Input
                    id="connectionString"
                    value={formState.storageConfig.azureBlobConfig.connectionString}
                    onChange={(event) =>
                      setFormState((current) => ({
                        ...current,
                        storageConfig: {
                          ...current.storageConfig,
                          azureBlobConfig: {
                            ...current.storageConfig.azureBlobConfig,
                            connectionString: event.target.value,
                          },
                        },
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="containerName">Container Name</Label>
                  <Input
                    id="containerName"
                    value={formState.storageConfig.azureBlobConfig.containerName}
                    onChange={(event) =>
                      setFormState((current) => ({
                        ...current,
                        storageConfig: {
                          ...current.storageConfig,
                          azureBlobConfig: {
                            ...current.storageConfig.azureBlobConfig,
                            containerName: event.target.value,
                          },
                        },
                      }))
                    }
                    required
                  />
                </div>
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor="region">Region</Label>
                  <Input
                    id="region"
                    value={formState.storageConfig.awsS3Config.region}
                    onChange={(event) =>
                      setFormState((current) => ({
                        ...current,
                        storageConfig: {
                          ...current.storageConfig,
                          awsS3Config: {
                            ...current.storageConfig.awsS3Config,
                            region: event.target.value,
                          },
                        },
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="bucketName">Bucket Name</Label>
                  <Input
                    id="bucketName"
                    value={formState.storageConfig.awsS3Config.bucketName}
                    onChange={(event) =>
                      setFormState((current) => ({
                        ...current,
                        storageConfig: {
                          ...current.storageConfig,
                          awsS3Config: {
                            ...current.storageConfig.awsS3Config,
                            bucketName: event.target.value,
                          },
                        },
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="accessKeyID">Access Key ID</Label>
                  <Input
                    id="accessKeyID"
                    value={formState.storageConfig.awsS3Config.accessKeyID}
                    onChange={(event) =>
                      setFormState((current) => ({
                        ...current,
                        storageConfig: {
                          ...current.storageConfig,
                          awsS3Config: {
                            ...current.storageConfig.awsS3Config,
                            accessKeyID: event.target.value,
                          },
                        },
                      }))
                    }
                    required
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="secretAccessKey">Secret Access Key</Label>
                  <Input
                    id="secretAccessKey"
                    type="password"
                    value={formState.storageConfig.awsS3Config.secretAccessKey}
                    onChange={(event) =>
                      setFormState((current) => ({
                        ...current,
                        storageConfig: {
                          ...current.storageConfig,
                          awsS3Config: {
                            ...current.storageConfig.awsS3Config,
                            secretAccessKey: event.target.value,
                          },
                        },
                      }))
                    }
                    required
                  />
                </div>
              </div>
            )}
          </section>

          <section className="space-y-4">
            <h2 className="text-lg font-semibold">Run timing</h2>
            <p className="text-sm text-muted-foreground">
              Restore jobs run once. Choose to start immediately or at a specific time.
            </p>
            <div className="space-y-1.5">
              <Label htmlFor="runMode">When to run</Label>
              <select
                id="runMode"
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={formState.runMode}
                onChange={(event) => setField("runMode", event.target.value as RunMode)}
              >
                <option value="IMMEDIATE">Immediately</option>
                <option value="ONE_TIME_AT">At specified time (one-time)</option>
              </select>
            </div>

            {formState.runMode === "ONE_TIME_AT" ? (
              <div className="space-y-1.5">
                <Label htmlFor="startAt">Run At</Label>
                <Input
                  id="startAt"
                  type="datetime-local"
                  value={formState.startAt}
                  onChange={(event) => setField("startAt", event.target.value)}
                  required
                />
              </div>
            ) : null}

            <p className="text-sm text-muted-foreground">{runSummary}</p>
          </section>

          <section className="space-y-4">
            <h2 className="text-lg font-semibold">Filters</h2>
            <p className="text-sm text-muted-foreground">
              Optional limits on which backed-up objects to restore into SharePoint.
            </p>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="minFileSize">Min File Size (kb)</Label>
                <Input
                  id="minFileSize"
                  type="number"
                  min={0}
                  value={formState.filterConfig.minFileSize}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      filterConfig: {
                        ...current.filterConfig,
                        minFileSize: event.target.value,
                      },
                    }))
                  }
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="maxFileSize">Max File Size (kb)</Label>
                <Input
                  id="maxFileSize"
                  type="number"
                  min={0}
                  value={formState.filterConfig.maxFileSize}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      filterConfig: {
                        ...current.filterConfig,
                        maxFileSize: event.target.value,
                      },
                    }))
                  }
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="createdBefore">Created Before</Label>
                <Input
                  id="createdBefore"
                  type="datetime-local"
                  value={formState.filterConfig.createdBefore}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      filterConfig: {
                        ...current.filterConfig,
                        createdBefore: event.target.value,
                      },
                    }))
                  }
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="createdAfter">Created After</Label>
                <Input
                  id="createdAfter"
                  type="datetime-local"
                  value={formState.filterConfig.createdAfter}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      filterConfig: {
                        ...current.filterConfig,
                        createdAfter: event.target.value,
                      },
                    }))
                  }
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="modifiedBefore">Modified Before</Label>
                <Input
                  id="modifiedBefore"
                  type="datetime-local"
                  value={formState.filterConfig.modifiedBefore}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      filterConfig: {
                        ...current.filterConfig,
                        modifiedBefore: event.target.value,
                      },
                    }))
                  }
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="modifiedAfter">Modified After</Label>
                <Input
                  id="modifiedAfter"
                  type="datetime-local"
                  value={formState.filterConfig.modifiedAfter}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      filterConfig: {
                        ...current.filterConfig,
                        modifiedAfter: event.target.value,
                      },
                    }))
                  }
                />
              </div>
            </div>
          </section>

          <div className="flex items-center gap-3">
            <Button type="submit" disabled={isSaving}>
              {isSaving ? "Creating..." : "Create Restore Job"}
            </Button>
          </div>

          {successMessage ? (
            <p className="text-sm text-emerald-600 dark:text-emerald-500">{successMessage}</p>
          ) : null}
          {errorMessage ? <p className="text-sm text-destructive">{errorMessage}</p> : null}
        </form>
      </Card>
    </main>
  );
}
