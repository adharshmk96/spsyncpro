"use client";

import Link from "next/link";
import { FormEvent, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type StorageType = "AZURE_BLOB" | "AWS_S3";
type RunMode = "IMMEDIATE" | "ONE_TIME_AT" | "RECURRING";
type Recurrence = "DAILY" | "WEEKLY" | "MONTHLY";

type BackupJobFormState = {
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
  recurrence: Recurrence;
};

const INITIAL_FORM_STATE: BackupJobFormState = {
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
  recurrence: "DAILY",
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

export default function DashboardBackupJobCreatePage() {
  const [formState, setFormState] = useState<BackupJobFormState>(INITIAL_FORM_STATE);
  const [isSaving, setIsSaving] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const scheduleSummary = useMemo(() => {
    if (formState.runMode === "IMMEDIATE") {
      return "Runs immediately after creation.";
    }

    if (formState.runMode === "ONE_TIME_AT") {
      return formState.startAt ? `Runs once at ${formState.startAt}.` : "Select run time.";
    }

    return formState.startAt
      ? `Runs ${formState.recurrence.toLowerCase()} from ${formState.startAt}.`
      : `Runs ${formState.recurrence.toLowerCase()} from selected start time.`;
  }, [formState.recurrence, formState.runMode, formState.startAt]);

  const setField = <K extends keyof BackupJobFormState>(field: K, value: BackupJobFormState[K]) => {
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

    if (formState.runMode !== "IMMEDIATE" && !formState.startAt) {
      setErrorMessage("Start time is required for one-time and recurring runs.");
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
      recurrence: formState.runMode === "RECURRING" ? formState.recurrence : undefined,
    };

    setIsSaving(true);
    try {
      const response = await fetch("/api/backup-jobs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const data = (await response.json()) as { error?: string };
      if (!response.ok) {
        throw new Error(data.error ?? "Failed to create backup job.");
      }

      setSuccessMessage("Backup job created successfully.");
      setFormState(INITIAL_FORM_STATE);
    } catch (error) {
      console.error("Backup job creation failed.", error);
      setErrorMessage(error instanceof Error ? error.message : "Failed to create backup job.");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <main className="p-6">
      <Card className="mx-auto max-w-5xl p-6">
        <div className="mb-6 flex items-center justify-between gap-3">
          <div>
            <h1 className="text-2xl font-semibold">Create Backup Job</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Configure run, storage, filters, and scheduling.
            </p>
          </div>
          <Button asChild variant="outline">
            <Link href="/dashboard/backup-job/list">Back to List</Link>
          </Button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-8">
          <section className="space-y-4">
            <h2 className="text-lg font-semibold">Run Config</h2>
            <div className="space-y-1.5">
              <Label htmlFor="siteUrl">Site URL</Label>
              <Input
                id="siteUrl"
                value={formState.siteUrl}
                onChange={(event) => setField("siteUrl", event.target.value)}
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
            <h2 className="text-lg font-semibold">Schedule Config</h2>
            <div className="space-y-1.5">
              <Label htmlFor="runMode">Run Mode</Label>
              <select
                id="runMode"
                className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                value={formState.runMode}
                onChange={(event) => setField("runMode", event.target.value as RunMode)}
              >
                <option value="IMMEDIATE">Immediately</option>
                <option value="ONE_TIME_AT">At specified time</option>
                <option value="RECURRING">Recurring</option>
              </select>
            </div>

            {formState.runMode !== "IMMEDIATE" ? (
              <div className="space-y-1.5">
                <Label htmlFor="startAt">Start At</Label>
                <Input
                  id="startAt"
                  type="datetime-local"
                  value={formState.startAt}
                  onChange={(event) => setField("startAt", event.target.value)}
                  required
                />
              </div>
            ) : null}

            {formState.runMode === "RECURRING" ? (
              <div className="space-y-1.5">
                <Label htmlFor="recurrence">Recurrence</Label>
                <select
                  id="recurrence"
                  className="w-full rounded-lg border border-input bg-transparent px-2.5 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
                  value={formState.recurrence}
                  onChange={(event) => setField("recurrence", event.target.value as Recurrence)}
                >
                  <option value="DAILY">Daily</option>
                  <option value="WEEKLY">Weekly</option>
                  <option value="MONTHLY">Monthly</option>
                </select>
              </div>
            ) : null}

            <p className="text-sm text-muted-foreground">{scheduleSummary}</p>
          </section>

          <section className="space-y-4">
            <h2 className="text-lg font-semibold">Filters</h2>
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
              {isSaving ? "Creating..." : "Create Backup Job"}
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
