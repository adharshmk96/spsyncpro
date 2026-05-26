# SPSync API

HTTP API server for the SPSync platform, built with Cobra, Viper, and Gin.

## Requirements

- Go 1.26+

## Run

From the project root:

```bash
go run . serve
```

The server listens on `0.0.0.0:8080` by default.

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
| `db.sqlite_path` | `./data/spsyncapi.sqlite` | `SPSYNCAPI_DB_SQLITE_PATH` |
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
├── internal/
│   ├── config/          # Configuration loading and validation
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # HTTP middleware (logging, metrics)
│   ├── routes/          # Route registration
│   ├── server/          # HTTP server setup and lifecycle
│   └── telemetry/       # OpenTelemetry metrics setup
├── config.yaml          # Example/default config
├── main.go              # Application entrypoint
└── go.mod
```

## Graceful shutdown

The server handles `SIGINT` and `SIGTERM`, draining in-flight requests before exit.
