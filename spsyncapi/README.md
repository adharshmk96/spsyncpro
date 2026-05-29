# SPSync API

HTTP API server for the SPSync platform, built with Cobra, Viper, and Gin.

## Requirements

- Go 1.26+
- [Task](https://taskfile.dev/) (optional, for common commands)

## Tasks

From the project root with [Task](https://taskfile.dev/) installed:

| Task | Description |
| --- | --- |
| `task run` | Start the API server (`go run . serve`) |
| `task test` | Run all tests (`go test ./...`) |
| `task spec` | Regenerate OpenAPI docs under `docs/` |
| `task build` | Build `bin/spsyncapi` |
| `task tidy` | Run `go mod tidy` |

Without Task, use the equivalent `go` commands directly (see sections below).

## Run

From the project root:

```bash
task run
# or
go run . serve
```

The server listens on `0.0.0.0:8080` by default.

## OpenAPI / Swagger

API documentation is generated with [swaggo/swag](https://github.com/swaggo/swag) from handler annotations.

Regenerate the spec after changing routes or handlers:

```bash
task spec
```

Artifacts:

- `docs/swagger.json` / `docs/swagger.yaml` — OpenAPI 2.0 spec
- `docs/docs.go` — embedded spec for the UI

With the server running, browse interactive docs at:

```text
http://localhost:8080/swagger/index.html
```

Protected endpoints use **BearerAuth**: set `Authorization` to `Bearer <token>` from login or register.

## Health check

```bash
curl http://localhost:8080/api/v1/health
```

Example response:

```json
{"status":"ok"}
```

## Configuration

Configuration is loaded from (in order):

1. Defaults in code
2. Optional `config.yaml` in the project root or `./config/`
3. Environment variables with the `SPSYNCAPI_` prefix

Use a custom config file:

```bash
go run . serve --config /path/to/config.yaml
```

### Settings

| Key | Default | Env var |
| --- | --- | --- |
| `server.host` | `0.0.0.0` | `SPSYNCAPI_SERVER_HOST` |
| `server.port` | `8080` | `SPSYNCAPI_SERVER_PORT` |
| `server.gin_mode` | `release` | `SPSYNCAPI_SERVER_GIN_MODE` |
| `server.read_timeout` | `15s` | `SPSYNCAPI_SERVER_READ_TIMEOUT` |
| `server.write_timeout` | `15s` | `SPSYNCAPI_SERVER_WRITE_TIMEOUT` |
| `server.shutdown_timeout` | `10s` | `SPSYNCAPI_SERVER_SHUTDOWN_TIMEOUT` |
| `log.level` | `info` | `SPSYNCAPI_LOG_LEVEL` |
| `metrics.enabled` | `true` | `SPSYNCAPI_METRICS_ENABLED` |
| `metrics.service_name` | `spsyncapi` | `SPSYNCAPI_METRICS_SERVICE_NAME` |
| `metrics.otlp_endpoint` | `localhost:4318` | `SPSYNCAPI_METRICS_OTLP_ENDPOINT` |
| `metrics.otlp_insecure` | `true` | `SPSYNCAPI_METRICS_OTLP_INSECURE` |
| `metrics.export_interval` | `15s` | `SPSYNCAPI_METRICS_EXPORT_INTERVAL` |
| `db.driver` | `sqlite` | `SPSYNCAPI_DB_DRIVER` |
| `db.sqlite_path` | `./data/spsyncapi.sqlite` | `SPSYNCAPI_DB_SQLITE_PATH` |
| `db.postgres_dsn` | *(see config.yaml)* | `SPSYNCAPI_DB_POSTGRES_DSN` |
| `temporal.host_port` | `localhost:7233` | `SPSYNCAPI_TEMPORAL_HOST_PORT` |
| `temporal.namespace` | `default` | `SPSYNCAPI_TEMPORAL_NAMESPACE` |
| `temporal.task_queue` | `spsync-transfer` | `SPSYNCAPI_TEMPORAL_TASK_QUEUE` |
| `temporal.reconcile_interval` | `2m` | `SPSYNCAPI_TEMPORAL_RECONCILE_INTERVAL` |
| `auth.jwt_secret` | *(required)* | `SPSYNCAPI_AUTH_JWT_SECRET` |
| `auth.jwt_issuer` | `spsyncapi` | `SPSYNCAPI_AUTH_JWT_ISSUER` |
| `auth.access_token_ttl` | `15m` | `SPSYNCAPI_AUTH_ACCESS_TOKEN_TTL` |
| `auth.session_ttl` | `720h` | `SPSYNCAPI_AUTH_SESSION_TTL` |
| `auth.password_reset_ttl` | `30m` | `SPSYNCAPI_AUTH_PASSWORD_RESET_TTL` |

Set `metrics.enabled` to `false` when no OTLP collector is available; request logging still works via slog.

Example override:

```bash
SPSYNCAPI_SERVER_PORT=9090 go run . serve
```

## Build

```bash
go build -o bin/spsyncapi .
./bin/spsyncapi serve
```

## Project layout

```text
.
├── cmd/                 # Cobra CLI commands
├── docs/                # Generated OpenAPI spec (run task spec)
├── internal/
│   ├── config/          # Configuration loading and validation
│   ├── handlers/        # HTTP handlers (swag annotations)
│   ├── middleware/      # HTTP middleware (logging, metrics)
│   ├── routes/          # Route registration
│   ├── server/          # HTTP server setup and lifecycle
│   └── telemetry/       # OpenTelemetry metrics setup
├── config.yaml          # Local dev (SQLite)
├── config.prod.yaml     # Production (PostgreSQL)
├── main.go              # Application entrypoint + API metadata
├── Taskfile.yml         # Task runner definitions
└── go.mod
```

## Graceful shutdown

The server handles `SIGINT` and `SIGTERM`, draining in-flight requests before exit.

## Environments: local dev vs production

| | Local dev | Production / staging |
|---|-----------|-------------------|
| Compose file | [`docker-compose.yml`](../docker-compose.yml) | [`docker-compose.prod.yml`](../docker-compose.prod.yml) |
| Temporal | Ephemeral `start-dev` (state lost on container restart) | PostgreSQL-backed `auto-setup` |
| App database | SQLite (`config.yaml`) | PostgreSQL (`config.prod.yaml`) |

### Docker (full stack)

From the **repository root**, copy [`.env.example`](../.env.example) to `.env`, set secrets, then:

```bash
docker compose up -d --build
# or from spsyncapi/: task compose:up
```

| Service | URL |
|---------|-----|
| UI | http://localhost:3000 |
| API | http://localhost:8080 |
| Temporal UI | http://localhost:8088 |

Production stack: `docker compose -f docker-compose.prod.yml up -d --build` (or `task compose:up:prod`).

### Local development (native)

From the **repository root**:

```bash
docker compose up -d temporal temporal-ui
# or from spsyncapi/: task temporal:up
```

From **spsyncapi/** (uses `config.yaml` → SQLite):

```bash
task run      # API
task worker   # Temporal worker + reconcile loop
```

Temporal UI: `http://localhost:8088` · gRPC: `localhost:7233`

After Temporal restarts locally, the worker reconcile loop (default every `2m`) rebuilds schedules from SQLite. Restart the worker once if you need immediate recovery.

### Production

From the **repository root**:

```bash
docker compose -f docker-compose.prod.yml up -d
# or from spsyncapi/: task temporal:up:prod
```

From **spsyncapi/** (uses `config.prod.yaml` → PostgreSQL on port `5433`):

```bash
task run:prod
task worker:prod
```

Override secrets and DSN via env vars (`SPSYNCAPI_DB_POSTGRES_DSN`, etc.) or a deployment-specific config file.

Job definitions remain in the app database (source of truth). The worker reconciles to Temporal on startup and every `temporal.reconcile_interval`, including after Temporal cluster recovery.
