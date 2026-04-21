import { eq } from "drizzle-orm";
import { headers } from "next/headers";
import { NextResponse } from "next/server";

import { auth } from "@/lib/auth";
import { db } from "@/lib/db";
import { backupjobs } from "@/lib/db/schema";
import { backupJobRuns } from "@/lib/db/schema";

async function requireSession() {
  return auth.api.getSession({
    headers: await headers(),
  });
}

type RouteContext = {
  params: Promise<{
    id: string;
  }>;
};

type JobStatusUpdatePayload = {
  status?: "ACTIVE" | "PAUSED";
};

export async function GET(_request: Request, context: RouteContext) {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized backup job read attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await context.params;
  if (!id || id.trim().length === 0) {
    return NextResponse.json({ error: "Job id is required." }, { status: 400 });
  }

  try {
    const job = await db.query.backupjobs.findFirst({
      where: eq(backupjobs.id, id.trim()),
    });

    if (!job) {
      return NextResponse.json({ error: "Backup job not found." }, { status: 404 });
    }

    const latestRun = await db.query.backupJobRuns.findFirst({
      where: eq(backupJobRuns.jobId, id.trim()),
      orderBy: (table, { desc }) => [desc(table.createdAt)],
    });

    return NextResponse.json(
      {
        job: {
          ...job,
          runSummary: latestRun
            ? {
                id: latestRun.id,
                status: latestRun.status,
                scheduledFor: latestRun.scheduledFor,
                startedAt: latestRun.startedAt,
                finishedAt: latestRun.finishedAt,
              }
            : null,
        },
      },
      { status: 200 }
    );
  } catch (error) {
    console.error("Failed to fetch backup job.", error);
    return NextResponse.json({ error: "Failed to fetch backup job." }, { status: 500 });
  }
}

export async function PATCH(request: Request, context: RouteContext) {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized backup job status update attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await context.params;
  if (!id || id.trim().length === 0) {
    return NextResponse.json({ error: "Job id is required." }, { status: 400 });
  }

  try {
    const body = (await request.json()) as JobStatusUpdatePayload;
    if (body.status !== "ACTIVE" && body.status !== "PAUSED") {
      return NextResponse.json({ error: "status must be ACTIVE or PAUSED." }, { status: 400 });
    }

    const now = new Date();
    const [updatedJob] = await db
      .update(backupjobs)
      .set({
        status: body.status,
        updatedAt: now,
      })
      .where(eq(backupjobs.id, id.trim()))
      .returning();

    if (!updatedJob) {
      return NextResponse.json({ error: "Backup job not found." }, { status: 404 });
    }

    return NextResponse.json({ job: updatedJob }, { status: 200 });
  } catch (error) {
    if (error instanceof SyntaxError) {
      return NextResponse.json({ error: "Invalid JSON body." }, { status: 400 });
    }

    console.error("Failed to update backup job status.", error);
    return NextResponse.json({ error: "Failed to update backup job status." }, { status: 500 });
  }
}
