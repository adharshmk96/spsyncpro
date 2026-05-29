import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { API_BASE_PATH, AUTH_COOKIE_NAME, SPSYNC_API_URL } from "@/lib/api/config";
import { clearAuthCookie } from "@/lib/api/cookie";

/** Revokes the session on spsyncapi (best effort) and clears the auth cookie. */
export async function POST(): Promise<NextResponse> {
  const cookieStore = await cookies();
  const token = cookieStore.get(AUTH_COOKIE_NAME)?.value;

  if (token) {
    try {
      await fetch(`${SPSYNC_API_URL}${API_BASE_PATH}/logout`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, Accept: "application/json" },
        cache: "no-store",
      });
    } catch (error) {
      console.warn("Logout upstream request failed; clearing cookie anyway.", error);
    }
  }

  const response = NextResponse.json({ success: true });
  clearAuthCookie(response);
  return response;
}
