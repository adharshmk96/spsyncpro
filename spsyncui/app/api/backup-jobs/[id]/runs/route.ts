import { and, desc, eq } from "drizzle-orm";
import { headers } from "next/headers";
import { NextResponse } from "next/server";

import { auth } from "@/lib/auth";
import { db } from "@/lib/db";
import { backupJobRuns, backupjobs } from "@/lib/db/schema";

type RouteContext = {
  params: Promise<{
    id: string;
  }>;
};

async function requireSession() {
  return auth.api.getSession({
    headers: await headers(),
  });
}

export async function GET(request: Request, context: RouteContext) {
  const session = await requireSession();
  if (!session?.user) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await context.params;
  if (!id?.trim()) {
    return NextResponse.json({ error: "Job id is required." }, { status: 400 });
  }

  const url = new URL(request.url);
  const limit = Math.max(1, Math.min(Number(url.searchParams.get("limit") ?? "20"), 100));

  try {
    const runs = await db.query.backupJobRuns.findMany({
      where: eq(backupJobRuns.jobId, id.trim()),
      orderBy: [desc(backupJobRuns.createdAt)],
      limit,
    });

    return NextResponse.json({ runs }, { status: 200 });
  } catch (error) {
    console.error("Failed to load backup runs.", error);
    return NextResponse.json({ error: "Failed to load backup runs." }, { status: 500 });
  }
}

export async function POST(_request: Request, context: RouteContext) {
  const session = await requireSession();
  if (!session?.user) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await context.params;
  if (!id?.trim()) {
    return NextResponse.json({ error: "Job id is required." }, { status: 400 });
  }

  try {
    const now = new Date();
    const existingJob = await db.query.backupjobs.findFirst({
      where: and(eq(backupjobs.id, id.trim()), eq(backupjobs.status, "ACTIVE")),
    });

    if (!existingJob) {
      return NextResponse.json({ error: "Active backup job not found." }, { status: 404 });
    }

    const existingMeta =
      existingJob.runnerMetadata !== null &&
      typeof existingJob.runnerMetadata === "object" &&
      !Array.isArray(existingJob.runnerMetadata)
        ? { ...(existingJob.runnerMetadata as Record<string, unknown>) }
        : {};

    await db
      .update(backupjobs)
      .set({
        nextRunAt: now,
        updatedAt: now,
        runnerMetadata: { ...existingMeta, manualRunAt: now.toISOString() },
      })
      .where(eq(backupjobs.id, existingJob.id));

    return NextResponse.json(
      {
        accepted: true,
        message: "Run requested. spsyncworker Temporal worker will execute the job.",
      },
      { status: 202 }
    );
  } catch (error) {
    console.error("Failed to trigger backup run.", error);
    return NextResponse.json({ error: "Failed to trigger backup run." }, { status: 500 });
  }
}
