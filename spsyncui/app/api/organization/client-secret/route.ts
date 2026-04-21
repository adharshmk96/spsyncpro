import { headers } from "next/headers";
import { NextResponse } from "next/server";
import { eq } from "drizzle-orm";

import { auth } from "@/lib/auth";
import { db } from "@/lib/db";
import { organization } from "@/lib/db/schema";
import { encryptOrganizationClientSecret } from "@/lib/security/organization-secret";

const DEFAULT_ORGANIZATION_ID = "default";

function validateClientSecretPayload(payload: unknown): string {
  if (!payload || typeof payload !== "object") {
    throw new Error("Invalid request payload.");
  }

  const candidate = payload as Record<string, unknown>;

  if (
    typeof candidate.clientSecret !== "string" ||
    candidate.clientSecret.trim().length === 0
  ) {
    throw new Error("Client secret is required.");
  }

  return candidate.clientSecret.trim();
}

async function requireSession() {
  const session = await auth.api.getSession({
    headers: await headers(),
  });

  return session;
}

export async function PUT(request: Request) {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized client secret update attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const body = await request.json();
    const clientSecret = validateClientSecretPayload(body);
    const encryptedClientSecret = encryptOrganizationClientSecret(clientSecret);
    const now = new Date();
    const existingOrganization = await db.query.organization.findFirst({
      where: eq(organization.id, DEFAULT_ORGANIZATION_ID),
    });

    if (!existingOrganization) {
      return NextResponse.json(
        { error: "Organization settings must be saved before setting client secret." },
        { status: 400 }
      );
    }

    await db
      .update(organization)
      .set({
        encryptedClientSecret,
        updatedAt: now,
      })
      .where(eq(organization.id, DEFAULT_ORGANIZATION_ID));

    console.info("Organization client secret updated.");
    return NextResponse.json({ ok: true }, { status: 200 });
  } catch (error) {
    if (error instanceof SyntaxError) {
      return NextResponse.json({ error: "Invalid JSON body." }, { status: 400 });
    }

    if (
      error instanceof Error &&
      [
        "Invalid request payload.",
        "Client secret is required.",
        "ORG_SECRET_ENCRYPTION_KEY is required.",
        "ORG_SECRET_ENCRYPTION_KEY must be a 64-character hex string.",
      ].includes(error.message)
    ) {
      console.warn("Client secret update rejected.", error.message);
      return NextResponse.json({ error: error.message }, { status: 400 });
    }

    console.error("Failed to update organization client secret.", error);
    return NextResponse.json(
      { error: "Failed to update organization client secret." },
      { status: 500 }
    );
  }
}
