# Plan: Создание сервиса научных документов (Documents/Corpus) с MinIO и контролем доступа

Создание нового Go-сервиса для управления научными документами с хранением в MinIO, версионированием, интеграцией с индексацией и role-based авторизацией (админы создают/редактируют, все читают).

## Steps

1. **Добавить MinIO в инфраструктуру**: добавить сервис `minio` в [deploy/docker-compose.dev.yml](deploy/docker-compose.dev.yml) (порты 9000/9001, volumes, credentials), создать volume `minio_data`

2. **Создать структуру сервиса documents**: создать директорию [services/documents](services/documents) со структурой `cmd/documents/main.go`, `internal/{config,domain,handler,nats,postgres,server,service,storage}`, `migrations/`, `Dockerfile`, скопировать паттерны из [services/users](services/users) и [services/auth](services/auth), добавить модуль в [go.work](go.work)

3. **Определить domain модели и события**: создать `internal/domain/document.go` с `Document` (id, title, author_id, category_id, tags, publication_date, content_path, version, indexed), `DocumentVersion` (id, document_id, version, content_path, changes), `Author`, добавить в [libs/dto](libs/dto) события `DocumentCreatedEvent`, `DocumentUpdatedEvent`, `DocumentDeletedEvent` с easyjson

4. **Создать PostgreSQL схему и SQL**: написать миграции в `migrations/001_create_tables.sql` для таблиц `documents`, `document_versions`, `authors`, `categories`, `tags`, `document_tags` с индексами, создать `internal/postgres/queries/*.sql` для CRUD операций, настроить sqlc

5. **Реализовать MinIO storage слой**: создать `internal/storage/minio.go` с `MinIOStorage` struct, методы `SaveDocument(ctx, documentID, version, content)`, `GetDocument(ctx, documentID, version)`, `ListVersions(ctx, documentID)`, использовать bucket `raibecas-documents` с naming pattern `{document_id}/v{version}.md`

6. **Создать NATS handlers с авторизацией**: реализовать `internal/handler/document_handler.go` с методами для топиков `documents.{create,update,delete}` (проверка role Admin/SuperAdmin из msg.Ctx), `documents.{get,list}` (доступны всем), публиковать события `corpus.document.{created,updated,deleted}` через [libs/natsw](libs/natsw)

7. **Интегрировать с gateway**: добавить routes в [services/gateway](services/gateway) (`POST /api/v1/documents`, `PUT /api/v1/documents/:id`, `DELETE /api/v1/documents/:id` - admin only, `GET /api/v1/documents`, `GET /api/v1/documents/:id` - public), создать `DocumentServiceConnector` с NATS request-reply

8. **Настроить подписку index-python**: обновить [services/index-python](services/index-python) для подписки на `corpus.document.{created,updated}`, добавить логику получения контента документа через NATS (`documents.get.content`), публиковать `indexing.document.indexed` после индексации

9. **Создать конфигурацию и Dockerfile**: добавить `internal/config/config.go` с `MinIOConfig` (endpoint, access_key, secret_key, bucket, ssl), создать `Dockerfile` multi-stage build, добавить `documents-service` в [deploy/docker-compose.dev.yml](deploy/docker-compose.dev.yml)

10. **Написать миграции и инициализацию MinIO**: создать SQL миграции с seed данными (категории, начальные теги), добавить инициализацию bucket в MinIO при старте сервиса (`internal/storage/init.go`)

## Further Considerations

1. **Формат версионирования в MinIO** — использовать встроенный MinIO versioning для bucket или собственную схему с явными путями `{id}/v{N}.md`? Собственная схема даст больше контроля над метаданными версий.

2. **Обработка больших файлов** — нужен ли multipart upload для документов или предполагаются файлы до нескольких MB? Можно добавить streaming upload через gateway.

3. **Миграция существующих документов** — есть ли уже документы для импорта или сервис стартует с нуля? Может понадобиться CLI для bulk import.

4. **Backup стратегия** — MinIO поддерживает репликацию, но нужна ли отдельная backup система для документов и БД?

## Архитектурные решения

### Хранение документов
- **Выбор**: MinIO (S3-совместимое хранилище)
- **Преимущества**: версионирование, масштабируемость, репликация, S3 API
- **Bucket**: `raibecas-documents`
- **Структура**: `{document_id}/v{version}.md`

### Метаданные документа
- **Обязательные поля**: title, author_id, publication_date
- **Дополнительные**: category_id, tags (многие-ко-многим), description
- **Версионирование**: отдельная таблица `document_versions` с историей изменений

### Контроль доступа
- **Чтение**: доступно всем аутентифицированным пользователям
- **Создание/Редактирование**: только Admin и SuperAdmin
- **Проверка**: на уровне gateway (middleware) + на уровне сервиса (handler)
- **Источник ролей**: Auth сервис предоставляет роли в JWT, gateway валидирует

### Интеграция с индексацией
- **Поток**: Documents Service → NATS event `corpus.document.created` → Index-Python
- **Получение контента**: Index-Python запрашивает через NATS топик `documents.get.content`
- **Подтверждение**: Index-Python публикует `indexing.document.indexed` → Documents Service обновляет флаг `indexed`

### NATS топики

#### Requests (Request-Reply)
- `documents.create` - создание документа (admin)
- `documents.update` - обновление документа (admin)
- `documents.delete` - удаление документа (admin)
- `documents.get` - получение документа (все)
- `documents.list` - список документов с фильтрами (все)
- `documents.get.content` - получение содержимого для индексации (internal)

#### Events (Pub-Sub)
- `corpus.document.created` - документ создан
- `corpus.document.updated` - документ обновлен
- `corpus.document.deleted` - документ удален
- `indexing.document.indexed` - документ проиндексирован (подписка)

### База данных

#### Таблицы
```sql
-- Авторы научных работ
CREATE TABLE authors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    bio TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Категории документов
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Теги для документов
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    title VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Основная таблица документов
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    author_id UUID NOT NULL REFERENCES authors(id),
    category_id INT NOT NULL REFERENCES categories(id),
    publication_date DATE NOT NULL,
    content_path VARCHAR(500) NOT NULL, -- путь в MinIO
    current_version INT NOT NULL DEFAULT 1,
    indexed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- История версий документов
CREATE TABLE document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version INT NOT NULL,
    content_path VARCHAR(500) NOT NULL,
    changes TEXT, -- описание изменений
    created_by UUID, -- кто создал версию
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(document_id, version)
);

-- Связь многие-ко-многим документы-теги
CREATE TABLE document_tags (
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (document_id, tag_id)
);

-- Индексы
CREATE INDEX idx_documents_author ON documents(author_id);
CREATE INDEX idx_documents_category ON documents(category_id);
CREATE INDEX idx_documents_publication_date ON documents(publication_date);
CREATE INDEX idx_documents_indexed ON documents(indexed);
CREATE INDEX idx_documents_title ON documents USING gin(to_tsvector('russian', title));
CREATE INDEX idx_document_versions_document ON document_versions(document_id, version DESC);
CREATE INDEX idx_document_tags_tag ON document_tags(tag_id);
```

### API Gateway Routes

```
Public (authenticated):
GET    /api/v1/documents              - список документов с фильтрами
GET    /api/v1/documents/:id          - получить документ
GET    /api/v1/documents/:id/versions - история версий
GET    /api/v1/documents/:id/content  - содержимое документа
GET    /api/v1/authors                - список авторов
GET    /api/v1/categories             - список категорий
GET    /api/v1/tags                   - список тегов

Admin only:
POST   /api/v1/documents              - создать документ
PUT    /api/v1/documents/:id          - обновить документ
DELETE /api/v1/documents/:id          - удалить документ
POST   /api/v1/authors                - создать автора
POST   /api/v1/categories             - создать категорию
POST   /api/v1/tags                   - создать тег
```

### Структура проекта

```
services/documents/
├── cmd/
│   └── documents/
│       └── main.go
├── internal/
│   ├── app/
│   │   └── app.go                    # инициализация приложения
│   ├── config/
│   │   └── config.go                 # конфигурация (env-based)
│   ├── domain/
│   │   ├── document.go               # Document, DocumentVersion
│   │   ├── author.go                 # Author
│   │   ├── category.go               # Category, Tag
│   │   └── events.go                 # события для NATS
│   ├── handler/
│   │   ├── document_handler.go       # NATS handlers для документов
│   │   ├── author_handler.go         # NATS handlers для авторов
│   │   └── category_handler.go       # NATS handlers для категорий/тегов
│   ├── nats/
│   │   └── publisher.go              # публикация событий
│   ├── postgres/
│   │   ├── postgres.go               # подключение к БД
│   │   ├── queries/
│   │   │   ├── documents.sql         # SQL запросы для sqlc
│   │   │   ├── authors.sql
│   │   │   └── categories.sql
│   │   └── sqlc/                     # сгенерированный код
│   ├── server/
│   │   └── server.go                 # регистрация NATS подписок
│   ├── service/
│   │   ├── document_service.go       # бизнес-логика документов
│   │   ├── author_service.go
│   │   └── category_service.go
│   └── storage/
│       ├── storage.go                # интерфейс хранилища
│       ├── minio.go                  # реализация MinIO
│       └── init.go                   # инициализация bucket
├── migrations/
│   ├── 001_create_tables.sql
│   └── 002_seed_data.sql
├── Dockerfile
├── go.mod
├── go.sum
├── sqlc.yaml
└── README.md
```

### MinIO конфигурация

```yaml
# docker-compose.dev.yml
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  environment:
    MINIO_ROOT_USER: raibecas
    MINIO_ROOT_PASSWORD: raibecas_minio_dev
  ports:
    - "9000:9000"   # S3 API
    - "9001:9001"   # Web Console
  volumes:
    - minio_data:/data
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
    interval: 30s
    timeout: 20s
    retries: 3
```

### Environment Variables (сервис documents)

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=raibecas
DB_PASSWORD=raibecas_dev
DB_NAME=raibecas
DB_SSL_MODE=disable

# NATS
NATS_URL=nats://localhost:4222
NATS_MAX_RECONNECTS=-1

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=raibecas
MINIO_SECRET_KEY=raibecas_minio_dev
MINIO_BUCKET=raibecas-documents
MINIO_USE_SSL=false

# Telemetry
TELEMETRY_ENABLED=true
TELEMETRY_SERVICE_NAME=documents
TELEMETRY_OTLP_ENDPOINT=localhost:4317
```

### Зависимости (go.mod)

```go
require (
    github.com/artmexbet/raibecas/libs/dto v0.0.0
    github.com/artmexbet/raibecas/libs/natsw v0.0.0
    github.com/artmexbet/raibecas/libs/telemetry v0.0.0
    
    github.com/google/uuid v1.6.0
    github.com/ilyakaznacheev/cleanenv v1.5.0
    github.com/jackc/pgx/v5 v5.5.5
    github.com/minio/minio-go/v7 v7.0.80
    github.com/nats-io/nats.go v1.37.0
    go.opentelemetry.io/otel v1.32.0
)
```
