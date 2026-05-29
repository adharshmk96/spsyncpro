import { cookies } from "next/headers";
import { NextRequest, NextResponse } from "next/server";

import { API_BASE_PATH, AUTH_COOKIE_NAME, SPSYNC_API_URL } from "@/lib/api/config";
import { setAuthCookie } from "@/lib/api/cookie";

/**
 * Same-origin BFF proxy. Forwards every request under `/api/v1/*` to the Go
 * spsyncapi backend, injecting the JWT from the httpOnly cookie as a Bearer
 * token and persisting silent token refreshes (`X-Access-Token`) back into the
 * cookie. The browser never talks to the Go API directly.
 */
async function proxy(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
): Promise<NextResponse> {
  const { path } = await context.params;
  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_COOKIE_NAME)?.value;

  const targetUrl = `${SPSYNC_API_URL}${API_BASE_PATH}/${path.join("/")}${request.nextUrl.search}`;

  const headers = new Headers();
  headers.set("Accept", "application/json");
  const contentType = request.headers.get("content-type");
  if (contentType) {
    headers.set("Content-Type", contentType);
  }
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const method = request.method.toUpperCase();
  const hasBody = method !== "GET" && method !== "HEAD";

  let upstream: Response;
  try {
    upstream = await fetch(targetUrl, {
      method,
      headers,
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });
  } catch (error) {
    console.error("BFF proxy request failed.", { targetUrl, error });
    return NextResponse.json({ error: "Upstream API is unreachable." }, { status: 502 });
  }

  const body = await upstream.text();
  const response = new NextResponse(body, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") ?? "application/json",
    },
  });

  const refreshedToken = upstream.headers.get("X-Access-Token");
  if (refreshedToken) {
    setAuthCookie(response, refreshedToken);
  }

  return response;
}

export {
  proxy as GET,
  proxy as POST,
  proxy as PUT,
  proxy as PATCH,
  proxy as DELETE,
};
