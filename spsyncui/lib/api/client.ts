import { ApiError } from "@/lib/api/errors";

/**
 * Browser-side fetch helper. Always targets the same-origin BFF proxy
 * (`/api/v1/...`), which injects the Bearer token from the httpOnly cookie and
 * persists silent token refreshes. The cookie is sent automatically.
 */
export async function clientApiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`/api/v1${path}`, {
    ...init,
    headers,
    credentials: "same-origin",
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

/** Serializes a value to a JSON request body. */
export function jsonBody(value: unknown): string {
  return JSON.stringify(value);
}
