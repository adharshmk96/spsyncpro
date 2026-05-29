/**
 * Shared configuration for talking to the Go spsyncapi backend.
 *
 * `SPSYNC_API_URL` is a server-only variable (no NEXT_PUBLIC_ prefix) because
 * the browser never calls the Go API directly — it always goes through the
 * same-origin BFF routes under `/api`.
 */
export const SPSYNC_API_URL = process.env.SPSYNC_API_URL ?? "http://localhost:8080";

/** Base path of the JSON API exposed by spsyncapi. */
export const API_BASE_PATH = "/api/v1";

/** Name of the httpOnly cookie that stores the JWT access token. */
export const AUTH_COOKIE_NAME = "spsync_token";

/**
 * Cookie lifetime in seconds. Mirrors the API session TTL (30 days) so the
 * cookie outlives short-lived access tokens; the API silently refreshes the
 * token via the `X-Access-Token` response header while the session is active.
 */
export const AUTH_COOKIE_MAX_AGE = 60 * 60 * 24 * 30;
