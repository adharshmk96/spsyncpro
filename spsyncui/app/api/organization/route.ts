import { eq } from "drizzle-orm";
import { headers } from "next/headers";
import { NextResponse } from "next/server";

import { auth } from "@/lib/auth";
import { db } from "@/lib/db";
import { organization } from "@/lib/db/schema";

const DEFAULT_ORGANIZATION_ID = "default";

type OrganizationPayload = {
  name: string;
  description: string;
  tenantId: string;
  clientId: string;
};

function isNonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function validateOrganizationPayload(payload: unknown): OrganizationPayload {
  if (!payload || typeof payload !== "object") {
    throw new Error("Invalid request payload.");
  }

  const candidate = payload as Record<string, unknown>;

  if (!isNonEmptyString(candidate.name)) {
    throw new Error("Name is required.");
  }

  if (!isNonEmptyString(candidate.description)) {
    throw new Error("Description is required.");
  }

  if (!isNonEmptyString(candidate.tenantId)) {
    throw new Error("Tenant ID is required.");
  }

  if (!isNonEmptyString(candidate.clientId)) {
    throw new Error("Client ID is required.");
  }

  return {
    name: candidate.name.trim(),
    description: candidate.description.trim(),
    tenantId: candidate.tenantId.trim(),
    clientId: candidate.clientId.trim(),
  };
}

async function requireSession() {
  const session = await auth.api.getSession({
    headers: await headers(),
  });

  return session;
}

export async function GET() {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized organization read attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const existingOrganization = await db.query.organization.findFirst({
      where: eq(organization.id, DEFAULT_ORGANIZATION_ID),
    });

    return NextResponse.json(
      {
        organization: {
          id: DEFAULT_ORGANIZATION_ID,
          name: existingOrganization?.name ?? "",
          description: existingOrganization?.description ?? "",
          tenantId: existingOrganization?.tenantId ?? "",
          clientId: existingOrganization?.clientId ?? "",
          hasClientSecret: Boolean(existingOrganization?.encryptedClientSecret),
        },
      },
      { status: 200 }
    );
  } catch (error) {
    console.error("Failed to fetch organization settings.", error);
    return NextResponse.json(
      { error: "Failed to fetch organization settings." },
      { status: 500 }
    );
  }
}

export async function PUT(request: Request) {
  const session = await requireSession();
  if (!session?.user) {
    console.warn("Unauthorized organization upsert attempt.");
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const body = await request.json();
    const payload = validateOrganizationPayload(body);
    const now = new Date();

    await db
      .insert(organization)
      .values({
        id: DEFAULT_ORGANIZATION_ID,
        name: payload.name,
        description: payload.description,
        tenantId: payload.tenantId,
        clientId: payload.clientId,
        createdAt: now,
        updatedAt: now,
      })
      .onConflictDoUpdate({
        target: organization.id,
        set: {
          name: payload.name,
          description: payload.description,
          tenantId: payload.tenantId,
          clientId: payload.clientId,
          updatedAt: now,
        },
      });

    const updatedOrganization = await db.query.organization.findFirst({
      where: eq(organization.id, DEFAULT_ORGANIZATION_ID),
    });

    console.info("Organization settings upsert completed.");

    return NextResponse.json(
      {
        organization: {
          id: DEFAULT_ORGANIZATION_ID,
          name: updatedOrganization?.name ?? payload.name,
          description: updatedOrganization?.description ?? payload.description,
          tenantId: updatedOrganization?.tenantId ?? payload.tenantId,
          clientId: updatedOrganization?.clientId ?? payload.clientId,
          hasClientSecret: Boolean(updatedOrganization?.encryptedClientSecret),
        },
      },
      { status: 200 }
    );
  } catch (error) {
    if (error instanceof SyntaxError) {
      return NextResponse.json({ error: "Invalid JSON body." }, { status: 400 });
    }

    if (
      error instanceof Error &&
      [
        "Invalid request payload.",
        "Name is required.",
        "Description is required.",
        "Tenant ID is required.",
        "Client ID is required.",
      ].includes(error.message)
    ) {
      return NextResponse.json({ error: error.message }, { status: 400 });
    }

    console.error("Failed to upsert organization settings.", error);
    return NextResponse.json(
      { error: "Failed to save organization settings." },
      { status: 500 }
    );
  }
}
