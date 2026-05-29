<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

Tech stack
- bun runtime
- nextjs
- shadcn ui

Data & auth
- This UI has NO database and NO ORM. All data and auth come from the Go `spsyncapi` backend (`/api/v1`).
- A thin BFF lives under `app/api`: `app/api/v1/[...path]` proxies to the Go API and injects the JWT; `app/api/auth/{login,register,logout}` manage the httpOnly auth cookie.
- Server components read via `serverApiFetch` (`lib/api/server.ts`); client components read via `clientApiFetch` (`lib/api/client.ts`). Never add direct DB access or business logic here.
- Configure `SPSYNC_API_URL` in `.env`.