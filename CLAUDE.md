# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Start infrastructure only (for local service development)
make up-env

# Start everything via Docker (infra + services)
make up

# Run a service locally (example: gateway)
cd services/gateway
ENVIRONMENT=development go run cmd/gateway/main.go

# Lint (golangci-lint)
make lint

# Pull Ollama models needed for index-python
make setup

# Regenerate mocks (mockery)
mockery

# Regenerate easyjson code after adding new DTOs
go generate ./...

# Run tests in a specific package
go test ./services/auth/internal/service/...
```

## Architecture

```
Frontend (React)
    ↓ HTTP/WebSocket
Gateway (Go + Fiber)          ← sole entry point for all clients
    ↓ NATS Request-Reply (sync RPC)
auth  ·  documents  ·  users  ·  chat   ← Go services
    ↓ NATS Events (async)
index-python (Ollama + Qdrant)           ← Python ML service
```

**Infra:** PostgreSQL · Redis · NATS JetStream · MinIO · Qdrant · Jaeger · Prometheus · Grafana

## Go Workspace

All Go modules are in `go.work` — no need to run `go mod tidy` in each service separately.

```
libs/dto       — shared DTOs between services (with easyjson)
libs/natsw     — NATS wrapper with middleware and OTel trace propagation
libs/telemetry — OpenTelemetry initialization (OTLP → Jaeger)
libs/utils
services/{auth,chat,documents,gateway,users}
```

## Service Internal Structure (Clean Architecture)

Each Go service follows the same layout:
```
services/{name}/
├── cmd/{name}/       — entry point
├── internal/
│   ├── config/       — env-based config
│   ├── domain/       — domain models and errors
│   ├── repository/   — PostgreSQL access (auth/users/documents only)
│   ├── service/      — business logic
│   ├── handler/      — NATS subscriptions
│   └── server/       — HTTP (gateway only)
└── migrations/       — SQL migrations (golang-migrate)
```

## Communication Patterns

### NATS Request-Reply (sync)

Gateway → service via `natsw.Client.RequestMsg`. All subjects defined in [services/gateway/internal/connector/nats_connector.go](services/gateway/internal/connector/nats_connector.go):
- `documents.{list,get,create,update,delete}`, `documents.bookmarks.*`, `documents.notes.*`
- `auth.{login,logout,validate,refresh,change_password}`
- `users.*`
- `corpus.search` — routed to index-python

### NATS Events (async)

- `corpus.document.{created,updated,deleted}` — documents → index-python (Claim Check pattern: only `content_path` in message, content fetched from MinIO)
- `auth.user.registered`, `auth.registration.requested` — auth → users
- `admin.registration.{approved,rejected}` — users → auth (via Outbox pattern)

All NATS messages are JSON. Response format: `{"success": bool, "data": ..., "error": "..."}` (see [libs/dto/response.go](libs/dto/response.go)).

## Serialization

**easyjson** is used for performance-critical serialization. When adding a new DTO:
1. Add `//easyjson:json` annotation to the struct
2. Run `go generate` in the package
3. In NATS handlers use `msg.UnmarshalEasyJSON` / `msg.RespondEasyJSON` (not legacy `UnmarshalData`/`RespondJSON`)

## Auth Architecture

JWT with HttpOnly cookies: access token (15 min) + refresh token in HttpOnly cookie + fingerprint cookie.  
Every request: Gateway → `auth.validate` (NATS). WebSocket connections use `skip_fingerprint: true`.

## Outbox Pattern (users service)

`services/users/internal/outbox/processor.go` polls the DB for unprocessed events every 5s, publishes to NATS, and marks them processed — all within a `SELECT FOR UPDATE` transaction to prevent double-processing.

## index-python Config

Uses double underscore for nested sections via `pydantic-settings`:
`OLLAMA__URL`, `QDRANT__HOST`, `NATS__SERVERS`, `CHUNK__CHUNK_SIZE`, etc.

## Environment Variables

Config pattern from `deploy/docker-compose.dev.yml`:
- `DB_HOST/PORT/USER/PASSWORD/NAME` — PostgreSQL
- `NATS_URL` / `NATS_CONNECTION_NAME`
- `REDIS_HOST/PORT`
- `JWT_SECRET` / `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL`
- `TELEMETRY_ENABLED` / `TELEMETRY_OTLP_ENDPOINT` (→ `jaeger:4318`)
- `MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET`

## Tracing

Every service calls `libs/telemetry.InitTracer()`. `natsw.Client` automatically propagates OTel trace context via NATS headers. Jaeger UI: http://localhost:16686.

## Linting

golangci-lint with: `govet`, `errcheck`, `staticcheck`, `unused`, `ineffassign`, `revive`, `gocyclo`.  
Formatters: `gofmt`, `goimports`, `gci` (import order: std → external → libs → services).

## Mocks

mockery is configured in `.mockery.yml` — generates in-package mocks with `MockInterfaceName` pattern and `WithExpect` methods. Covered packages: `documents/service`, `gateway/server`, `auth/service`, `auth/pkg/jwt`.
