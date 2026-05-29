# spsyncui

The web UI for SPSyncPro. This is a pure presentation layer: it has **no
database and no business logic**. Every screen renders data fetched from the Go
`spsyncapi` backend.

## Architecture

- **Next.js (App Router) + shadcn/ui**, run with bun.
- **BFF (Backend-for-Frontend)** under `app/api`:
  - `app/api/v1/[...path]/route.ts` — same-origin proxy to `spsyncapi`. It
    injects the JWT (read from an httpOnly cookie) as a `Bearer` token and
    persists silent token refreshes (`X-Access-Token`).
  - `app/api/auth/{login,register,logout}/route.ts` — wrap the API auth
    endpoints and manage the httpOnly auth cookie.
- **Data access helpers** in `lib/api/`:
  - `server.ts` (`serverApiFetch`) for React Server Components.
  - `client.ts` (`clientApiFetch`) for client components (hits the BFF proxy).
  - `types.ts` mirrors the API DTOs.
- **Route protection**: `middleware.ts` gates `/dashboard/*` on cookie presence;
  the dashboard layout additionally validates the session via `/me`.

The browser never calls the Go API directly — all traffic flows through the
same-origin BFF, so no CORS configuration is required on the API.

## Getting started

1. Start the Go `spsyncapi` backend (default `http://localhost:8080`).
2. Set `SPSYNC_API_URL` in `.env` (see `.env.example`).
3. Install and run:

```bash
bun install
bun dev
```

Open [http://localhost:3000](http://localhost:3000).
