import { cookies } from "next/headers";

import { API_BASE_PATH, AUTH_COOKIE_NAME, SPSYNC_API_URL } from "@/lib/api/config";
import { ApiError } from "@/lib/api/errors";

/**
 * Server-side fetch helper for React Server Components and route handlers.
 *
 * Reads the JWT from the httpOnly cookie and forwards it as a Bearer token to
 * the Go API. Server components cannot persist refreshed tokens (only route
 * handlers can set cookies), but the API keeps minting fresh tokens from the
 * still-valid session, so reads continue to work until the session expires.
 */
export async function serverApiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_COOKIE_NAME)?.value;

  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${SPSYNC_API_URL}${API_BASE_PATH}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });

  const raw = await response.text();
  const data = raw ? JSON.parse(raw) : null;

  if (!response.ok) {
    const message =
      (data && typeof data === "object" && "error" in data && typeof data.error === "string"
        ? data.error
        : null) ?? `Request failed with status ${response.status}`;
    throw new ApiError(response.status, message);
  }

  return data as T;
}
