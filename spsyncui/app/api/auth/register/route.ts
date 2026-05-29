import { NextRequest, NextResponse } from "next/server";

import { API_BASE_PATH, SPSYNC_API_URL } from "@/lib/api/config";
import { setAuthCookie } from "@/lib/api/cookie";
import type { AuthTokenBody } from "@/lib/api/types";

/** Registers a member against spsyncapi and stores the returned JWT in a cookie. */
export async function POST(request: NextRequest): Promise<NextResponse> {
  const body = (await request.json().catch(() => null)) as {
    email?: unknown;
    password?: unknown;
  } | null;

  if (typeof body?.email !== "string" || typeof body?.password !== "string") {
    return NextResponse.json({ error: "Email and password are required." }, { status: 400 });
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${SPSYNC_API_URL}${API_BASE_PATH}/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ email: body.email, password: body.password }),
      cache: "no-store",
    });
  } catch (error) {
    console.error("Register upstream request failed.", error);
    return NextResponse.json({ error: "Authentication service is unreachable." }, { status: 502 });
  }

  const data = (await upstream.json().catch(() => null)) as (AuthTokenBody & { error?: string }) | null;

  if (!upstream.ok || !data?.token) {
    return NextResponse.json(
      { error: data?.error ?? "Unable to create account." },
      { status: upstream.status || 400 }
    );
  }

  const response = NextResponse.json({ success: true }, { status: 201 });
  setAuthCookie(response, data.token);
  return response;
}
