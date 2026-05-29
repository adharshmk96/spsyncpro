import { ApiError } from "@/lib/api/errors";

/**
 * Browser-side auth calls. These target the BFF auth routes (`/api/auth/*`)
 * rather than the `/api/v1` proxy because they manage the httpOnly auth cookie.
 */
async function postAuth(path: string, body: Record<string, string>): Promise<void> {
  const response = await fetch(`/api/auth/${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(body),
    credentials: "same-origin",
  });

  if (!response.ok) {
    const data = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new ApiError(response.status, data?.error ?? "Request failed.");
  }
}

export function login(email: string, password: string): Promise<void> {
  return postAuth("login", { email, password });
}

export function register(email: string, password: string): Promise<void> {
  return postAuth("register", { email, password });
}

export async function logout(): Promise<void> {
  await fetch("/api/auth/logout", { method: "POST", credentials: "same-origin" });
}
