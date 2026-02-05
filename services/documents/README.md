# Documents Service

Сервис управления научными документами с поддержкой версионирования и интеграции с индексацией.

## Функциональность

- CRUD операции с документами
- Версионирование документов в MinIO
- Управление авторами, категориями и тегами
- Публикация событий в NATS для индексации
- Поддержка distributed tracing

## Архитектура

```
documents service
├── MinIO (хранилище файлов)
├── PostgreSQL (метаданные)
└── NATS (события и RPC)
```

## NATS Topics

### Request-Reply

- `documents.create` - создание документа (admin)
- `documents.update` - обновление документа (admin)
- `documents.delete` - удаление документа (admin)
- `documents.get` - получение документа (все)
- `documents.list` - список документов (все)
- `documents.get.content` - получение содержимого (internal)
- `documents.versions` - список версий (все)

### Events (Publish)

- `corpus.document.created` - документ создан
- `corpus.document.updated` - документ обновлен
- `corpus.document.deleted` - документ удален

### Events (Subscribe)

- `indexing.document.indexed` - документ проиндексирован

## Конфигурация

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=raibecas
DB_PASSWORD=raibecas_dev
DB_NAME=raibecas

# NATS
NATS_URL=nats://localhost:4222
NATS_CONNECTION_NAME=documents-service

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=raibecas
MINIO_SECRET_KEY=raibecas_minio_dev
MINIO_BUCKET=raibecas-documents
MINIO_USE_SSL=false

# Telemetry
TELEMETRY_ENABLED=true
TELEMETRY_SERVICE_NAME=documents
TELEMETRY_OTLP_ENDPOINT=localhost:4318
```

## Разработка

### Миграции

```powershell
# Подключение к БД
psql -h localhost -U raibecas -d raibecas

# Применить миграции
psql -h localhost -U raibecas -d raibecas -f migrations/001_create_tables.sql
psql -h localhost -U raibecas -d raibecas -f migrations/002_seed_data.sql
```

### Генерация кода с SQLC

```powershell
# Установить sqlc (если еще не установлен)
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Генерация
sqlc generate
```

### Запуск

```powershell
# Установить зависимости
go mod tidy

# Запустить сервис
go run cmd/documents/main.go
```

### Docker

```powershell
# Build
docker build -t raibecas/documents:latest .

# Run
docker run --rm `
  -e DB_HOST=postgres `
  -e DB_PASSWORD=raibecas_dev `
  -e NATS_URL=nats://nats:4222 `
  -e MINIO_ENDPOINT=minio:9000 `
  -e MINIO_SECRET_KEY=raibecas_minio_dev `
  raibecas/documents:latest
```

## Структура хранилища MinIO

```
raibecas-documents/
└── {document-id}/
    ├── v1.md
    ├── v2.md
    └── v3.md
```

## API через Gateway

```
GET    /api/v1/documents              - список документов
GET    /api/v1/documents/:id          - получить документ
GET    /api/v1/documents/:id/content  - содержимое документа
GET    /api/v1/documents/:id/versions - история версий
POST   /api/v1/documents              - создать (admin)
PUT    /api/v1/documents/:id          - обновить (admin)
DELETE /api/v1/documents/:id          - удалить (admin)
```
