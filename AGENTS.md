# AGENTS.md — Raibecas Coding Agent Guide

## Project Overview

**Raibecas** — веб-платформа для работы с научными текстами с RAG-чатботом на локальных LLM (Ollama). Go-монорепозиторий с Python ML-сервисом.

## Skills

Скиллы лежат в .agents/skills

## Architecture

```
Frontend (React)
    ↓ HTTP/WebSocket
Gateway (Go + Fiber)          ← единственная точка входа
    ↓ NATS Request-Reply
auth-service  documents-service  users-service  chat-service
    ↓ NATS Events
index-python (Ollama + Qdrant)
```

**Infra:** PostgreSQL · Redis · NATS (JetStream) · MinIO · Qdrant · Jaeger · Prometheus · Grafana

## Go Workspace

Все Go-модули объединены в `go.work` — не нужно запускать `go mod tidy` в каждом сервисе отдельно.

```
libs/dto       — общие DTO между сервисами (с easyjson)
libs/natsw     — обёртка NATS с middleware и OTel trace propagation
libs/telemetry — инициализация OpenTelemetry (OTLP → Jaeger)
libs/utils
services/{auth,chat,documents,gateway,users}
```

## Key Developer Commands

```powershell
# Поднять только инфраструктуру (для локальной разработки сервисов)
make up-env

# Поднять всё (сервисы + инфра) через Docker
make up

# Запустить сервис локально (пример: gateway)
cd services/gateway
$env:ENVIRONMENT="development"
go run cmd/gateway/main.go

# Lint
make lint

# Ollama модели (нужны для index-python)
make setup   # ollama pull embeddinggemma:300m && ollama pull gemma3:4b
```

## Communication Patterns

### NATS Request-Reply (синхронный RPC)

Gateway → сервис через `natsw.Client.RequestMsg`. Все топики — в `services/gateway/internal/connector/nats_connector.go`:
- `documents.{list,get,create,update,delete}`
- `auth.{login,logout,validate,refresh,change_password}`
- `users.*`

### NATS Events (асинхронные события)

- `corpus.document.{created,updated,deleted}` — documents → index-python
- `auth.user.registered`, `auth.registration.requested` — auth → users/admin
- `admin.registration.{approved,rejected}` — users → auth

### Message Format

Все NATS-сообщения — JSON. Ответ всегда `{"success": bool, "data": ..., "error": "..."}` (см. `libs/dto/response.go`).

## Serialization

Проект использует **easyjson** для высокопроизводительной сериализации. При добавлении нового DTO:

1. Добавь аннотацию `//easyjson:json` к структуре
2. Запусти `go generate` в пакете
3. В обработчиках NATS используй `msg.UnmarshalEasyJSON` / `msg.RespondEasyJSON` (не legacy `UnmarshalData`/`RespondJSON`)

## Auth Architecture

JWT с HttpOnly cookies: access token (15 мин) + refresh token в HttpOnly cookie + fingerprint cookie.  
Gateway → `auth.validate` (NATS) при каждом запросе. WebSocket-соединения используют `skip_fingerprint: true`.

## Service Internal Structure (Clean Architecture)

```
services/{name}/
├── cmd/{name}/       — точка входа
├── internal/
│   ├── config/       — конфигурация из env
│   ├── domain/       — доменные модели и ошибки
│   ├── repository/   — PostgreSQL (только auth/users/documents)
│   ├── service/      — бизнес-логика
│   ├── handler/      — NATS-обработчики (подписки)
│   └── server/       — HTTP (только gateway)
└── migrations/       — SQL миграции
```

## Tracing

Каждый сервис инициализирует `libs/telemetry.InitTracer()`. `natsw.Client` автоматически пропагирует OTel trace context через NATS headers. Jaeger UI: http://localhost:16686.

## Environment Variables Pattern

Конфигурация читается из env. Пример из `deploy/docker-compose.dev.yml`:
- `DB_HOST/PORT/USER/PASSWORD/NAME` — PostgreSQL
- `NATS_URL` / `NATS_CONNECTION_NAME`
- `REDIS_HOST/PORT`
- `JWT_SECRET` / `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL`
- `TELEMETRY_ENABLED` / `TELEMETRY_OTLP_ENDPOINT` (→ `jaeger:4318`)
- `MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET`

## index-python Config

Двойной underscore для вложенных секций: `OLLAMA__URL`, `QDRANT__HOST`, `NATS__SERVERS`, `CHUNK__CHUNK_SIZE`.

## Frontend

Два React приложения в `frontend/apps/`: `admin-panel` и `user-app`.  
Пакетный менеджер: **Bun**. Shared packages в `frontend/packages/`.

