import { NextRequest, NextResponse } from "next/server";

import { AUTH_COOKIE_NAME } from "@/lib/api/config";

/**
 * Gates the dashboard. Presence of the auth cookie is a cheap pre-check; the
 * BFF proxy and server components still enforce real auth against the API.
 */
export function middleware(request: NextRequest): NextResponse {
  const hasToken = Boolean(request.cookies.get(AUTH_COOKIE_NAME)?.value);

  if (!hasToken) {
    const loginUrl = new URL("/login", request.url);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*"],
};
