import type { NextResponse } from "next/server";

import { AUTH_COOKIE_MAX_AGE, AUTH_COOKIE_NAME } from "@/lib/api/config";

/** Persists the JWT access token in a hardened httpOnly cookie. */
export function setAuthCookie(response: NextResponse, token: string): void {
  response.cookies.set({
    name: AUTH_COOKIE_NAME,
    value: token,
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: AUTH_COOKIE_MAX_AGE,
  });
}

/** Removes the auth cookie (used on logout / failed auth). */
export function clearAuthCookie(response: NextResponse): void {
  response.cookies.set({
    name: AUTH_COOKIE_NAME,
    value: "",
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 0,
  });
}
