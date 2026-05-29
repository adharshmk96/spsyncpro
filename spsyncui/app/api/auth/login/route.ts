import { NextRequest, NextResponse } from "next/server";

import { API_BASE_PATH, SPSYNC_API_URL } from "@/lib/api/config";
import { setAuthCookie } from "@/lib/api/cookie";
import type { AuthTokenBody } from "@/lib/api/types";

/** Logs in against spsyncapi and stores the returned JWT in an httpOnly cookie. */
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
    upstream = await fetch(`${SPSYNC_API_URL}${API_BASE_PATH}/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ email: body.email, password: body.password }),
      cache: "no-store",
    });
  } catch (error) {
    console.error("Login upstream request failed.", error);
    return NextResponse.json({ error: "Authentication service is unreachable." }, { status: 502 });
  }

  const data = (await upstream.json().catch(() => null)) as (AuthTokenBody & { error?: string }) | null;

  if (!upstream.ok || !data?.token) {
    return NextResponse.json(
      { error: data?.error ?? "Invalid email or password." },
      { status: upstream.status || 401 }
    );
  }

  const response = NextResponse.json({ success: true });
  setAuthCookie(response, data.token);
  return response;
}
