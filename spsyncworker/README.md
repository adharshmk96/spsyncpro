# spsyncworker

Go Temporal worker that owns backup job scheduling and execution simulation.

## Responsibilities

- Read active jobs from shared Postgres (`backupjobs` table).
- Start one Temporal workflow per active job (`workflowID = job/<id>`).
- Run deterministic timer-based simulation loops in workflows.
- Log sanitized job configuration and simulated run timestamps via Activities.

`spsyncui` remains the control plane (create, start, stop, view jobs).

## Required Environment Variables

- `DATABASE_URL` - Postgres connection string (shared with `spsyncui` DB).
- `ENCRYPTION_KEY` - reserved for decryption flows and validated at startup.

## Optional Environment Variables

- `TEMPORAL_HOST_PORT` (default: `localhost:7233`)
- `TEMPORAL_NAMESPACE` (default: `default`)
- `TEMPORAL_TASK_QUEUE` (default: `spsync-jobs`)
- `JOBS_RECONCILE_INTERVAL_SECONDS` (default: `30`)

## Local Development

1. Start Temporal dev server:
   - `temporal server start-dev`
2. Ensure Postgres is available and `DATABASE_URL` points to the same schema used by `spsyncui`.
3. Run worker:
   - `go run ./cmd/worker`
4. Create or update jobs in `spsyncui`.
5. Verify worker logs:
   - job config logging (sanitized)
   - simulated run start/end timestamps

## Tests

- `go test ./...`
