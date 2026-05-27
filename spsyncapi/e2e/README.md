# SPSync API E2E tests (Bun)

## Prerequisites

- Bun installed
- API running locally (default: `http://localhost:8080`)

## Configuration

Copy `.env.example` values into your shell if needed:

```bash
export E2E_BASE_URL=http://localhost:8080/api/v1
```

## Run

From this directory:

```bash
bun test
```

Run one endpoint file:

```bash
bun test test/endpoints/health.test.js
```
