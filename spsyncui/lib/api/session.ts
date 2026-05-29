import { cookies } from "next/headers";

import { AUTH_COOKIE_NAME } from "@/lib/api/config";
import { serverApiFetch } from "@/lib/api/server";
import type { Member, MeResponse } from "@/lib/api/types";

/** Returns true when an auth cookie is present (cheap, no network call). */
export async function hasAuthCookie(): Promise<boolean> {
  const cookieStore = await cookies();
  return Boolean(cookieStore.get(AUTH_COOKIE_NAME)?.value);
}

/** Fetches the authenticated member, or null if the session is invalid. */
export async function getCurrentMember(): Promise<Member | null> {
  try {
    const { user } = await serverApiFetch<MeResponse>("/me");
    return user;
  } catch {
    return null;
  }
}
