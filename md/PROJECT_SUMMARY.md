# Raibecas — Сводное описание проекта для дипломной работы

## 1. Общая характеристика системы

**Raibecas** («Цифровой Райбекас») — веб-платформа для работы с научным наследием философа А.Я. Райбекаса. Система обеспечивает централизованное хранение, каталогизацию и семантический поиск по философским текстам, а также предоставляет интеллектуального ассистента на основе технологии Retrieval-Augmented Generation (RAG) с использованием локальных языковых моделей.

**Проблемная область:** Научное наследие философов, как правило, рассредоточено по разрозненным источникам и недоступно в машиночитаемом, структурированном виде. Платформа решает задачу оцифровки, систематизации и интеллектуального доступа к корпусу текстов.

**Ключевые функциональные возможности:**
- Хранение и отображение документов в формате Markdown с расширенными метаданными (авторы, категории, теги, типы документов, дата публикации)
- Система закладок (bookmarks) с поддержкой цитирования и контекстуальных заметок
- RAG-чатбот для диалога с корпусом текстов на русском языке
- Административная панель с модерацией заявок на регистрацию
- Полнотекстовый и семантический (векторный) поиск
- Сохранение истории диалогов

---

## 2. Архитектура системы

### 2.1. Общая архитектура

Система реализована в виде **микросервисной архитектуры** с событийно-ориентированным взаимодействием (Event-Driven Architecture, EDA). Единственной публично доступной точкой входа является API Gateway; все остальные сервисы изолированы и взаимодействуют исключительно через брокер сообщений NATS.

```
Клиент (браузер)
        │  HTTP/WebSocket
        ▼
┌──────────────────┐
│   API Gateway    │  Go + Fiber  — единственная точка входа
│   (port 8080)    │
└──┬───┬───┬───┬───┘
   │   │   │   │   NATS Request-Reply (синхронный RPC)
   ▼   ▼   ▼   ▼
┌──────┐ ┌──────────┐ ┌───────┐ ┌──────┐
│ auth │ │documents │ │ users │ │ chat │
│ (Go) │ │  (Go)    │ │ (Go)  │ │ (Go) │
└──────┘ └────┬─────┘ └───────┘ └──────┘
              │  NATS Events (асинхронные)
              ▼
     ┌─────────────────┐
     │  index-python   │  Python — ML Pipeline
     │  (Ollama+Qdrant) │
     └─────────────────┘
```

**Инфраструктурный стек:**

| Компонент | Технология | Назначение |
|---|---|---|
| Реляционная БД | PostgreSQL 17 | Основное хранилище данных |
| Кэш / сессии | Redis 7 | JWT refresh tokens, rate limiting |
| Брокер сообщений | NATS (JetStream) | Межсервисное взаимодействие |
| Объектное хранилище | MinIO (S3-совместимый) | Содержимое документов |
| Векторная БД | Qdrant | Хранение и поиск векторных эмбеддингов |
| LLM / Embeddings | Ollama | Локальный инференс языковых моделей |
| Трассировка | Jaeger (OTLP) | Распределённое трассирование запросов |
| Мониторинг | Prometheus + Grafana | Метрики и дашборды |

### 2.2. Структура монорепозитория

```
raibecas/
├── go.work                      # Go workspace (объединяет все Go-модули)
├── libs/
│   ├── dto/                     # Общие DTO между сервисами (easyjson)
│   ├── natsw/                   # Обёртка NATS с OTel trace propagation
│   ├── telemetry/               # Инициализация OpenTelemetry
│   └── utils/
├── services/
│   ├── gateway/                 # API Gateway (Go + Fiber)
│   ├── auth/                    # Сервис аутентификации (Go)
│   ├── documents/               # Сервис документов (Go)
│   ├── users/                   # Сервис пользователей (Go)
│   ├── chat/                    # Сервис чата (Go)
│   └── index-python/            # ML Pipeline — индексация (Python)
├── frontend/
│   ├── apps/
│   │   ├── admin-panel/         # React — панель администратора
│   │   └── user-app/            # React — клиентское приложение
│   └── packages/                # Общие пакеты (Bun workspace)
└── deploy/
    └── docker-compose.dev.yml   # Dev-окружение
```

---

## 3. Описание сервисов

### 3.1. API Gateway (`services/gateway`)

**Язык:** Go | **Фреймворк:** Fiber v2

Единственный сервис, доступный внешним клиентам. Выполняет:
- Маршрутизацию HTTP/WebSocket запросов к бэкенд-сервисам через NATS Request-Reply
- JWT аутентификацию (HttpOnly cookies: access token 15 мин + refresh token 168 ч)
- Проверку fingerprint cookie при каждом запросе (кроме WebSocket)
- Rate limiting
- CORS и security headers

**REST API:**
- `POST /api/v1/auth/login`, `/logout`, `/refresh`, `/change-password`
- `POST /api/v1/auth/register`, `/admin/requests/*` (модерация регистраций)
- `GET/POST/PUT/DELETE /api/v1/documents/*`
- `GET/POST/DELETE /api/v1/bookmarks/*`
- `GET /api/v1/documents/metadata/*` (авторы, категории, теги, типы)
- `GET/PUT /api/v1/users/*`
- `WebSocket /ws/chat`

**NATS-топики (исходящие запросы):**
```
documents.list         documents.get          documents.get.content
documents.create       documents.update       documents.delete
documents.cover.upload documents.reindex
documents.bookmarks.{list,create,delete}
documents.authors.{list,create}
documents.categories.{list,create}
documents.types.list   documents.authorship-types.list
documents.tags.{list,create}
auth.login             auth.logout            auth.validate
auth.refresh           auth.change_password
auth.registration.{list,create,approve,reject}
users.get              users.update
```

### 3.2. Auth Service (`services/auth`)

**Язык:** Go

Ответственность:
- Регистрация (создание заявок с модерацией администратором)
- Аутентификация, управление JWT (выдача, обновление, отзыв)
- Хранение refresh tokens в Redis
- Смена пароля

**Публикуемые NATS-события:**
- `auth.user.registered` — после создания пользователя
- `auth.registration.requested` — при подаче заявки на регистрацию

**Подписки:**
- `admin.registration.approved` / `admin.registration.rejected` — из users-service

### 3.3. Documents Service (`services/documents`)

**Язык:** Go

Ответственность:
- CRUD для документов, авторов, категорий, тегов, типов документов
- Управление участниками документа (автор, редактор, переводчик и т.д.)
- Хранение бинарного содержимого документов в MinIO (паттерн **Claim Check**)
- Управление закладками пользователей
- Запуск переиндексации по запросу

**Публикуемые NATS-события:**
- `corpus.document.created` — с `content_path` (ссылкой на MinIO)
- `corpus.document.updated`
- `corpus.document.deleted`

### 3.4. Users Service (`services/users`)

**Язык:** Go

Ответственность:
- CRUD профилей пользователей
- Модерация заявок на регистрацию (список, одобрение/отклонение)

### 3.5. Chat Service (`services/chat`)

**Язык:** Go

Ответственность:
- Обработка WebSocket соединений
- Управление сессиями чата и историей диалогов
- Отправка запросов к ML-пайплайну (через NATS или HTTP к index-python)
- Стриминг ответов LLM клиенту через WebSocket

### 3.6. Index-Python Service (`services/index-python`)

**Язык:** Python | **Async runtime:** asyncio

ML Pipeline. Ответственность:
- Подписка на `corpus.document.*` события
- Загрузка содержимого документа из MinIO по `content_path`
- Разбивка текста на чанки (`CHUNK_SIZE=700`, `CHUNK_OVERLAP=80`)
- Генерация векторных эмбеддингов через Ollama (`embeddinggemma:300m`)
- Запись векторов в Qdrant (коллекция `documents`, Cosine distance, размерность 768)

**Используемые LLM-модели (Ollama):**
- `embeddinggemma:300m` — для генерации эмбеддингов
- `gemma3:4b` — для генерации ответов в чате

---

## 4. Паттерны проектирования

### 4.1. Claim Check (Enterprise Integration Patterns)

При загрузке документа его содержимое сохраняется в MinIO, а в NATS-сообщение включается только путь (`content_path`). Подписчики самостоятельно получают содержимое по ссылке. Это предотвращает перегрузку брокера объёмными payload-ами и позволяет масштабировать хранилище независимо от транспортного уровня. *(Hohpe G., Woolf B. Enterprise Integration Patterns, 2003)*

### 4.2. Publisher/Subscriber (EDA)

Documents-service публикует доменные события (документ создан/обновлён/удалён) без знания о подписчиках. Index-python подписывается и обрабатывает события независимо. Обеспечивает слабую связанность (loose coupling).

### 4.3. Request-Reply (синхронный RPC через NATS)

Gateway взаимодействует с бэкенд-сервисами через `natsw.Client.RequestMsg` с таймаутом 5 секунд. Это обеспечивает синхронную семантику при сохранении транспортной независимости.

### 4.4. Clean Architecture (внутри каждого сервиса)

```
cmd/{name}/          — точка входа, инициализация DI
internal/
  config/            — конфигурация из env-переменных
  domain/            — доменные модели, ошибки (нет зависимостей)
  repository/        — слой доступа к данным (PostgreSQL)
  service/           — бизнес-логика
  handler/           — NATS-обработчики (подписки)
  server/            — HTTP / WebSocket (только gateway)
```

### 4.5. Outbox Pattern

В documents-service реализован Outbox Pattern для надёжной доставки событий: событие о создании/обновлении документа записывается в БД транзакционно вместе с самим документом, а фоновый воркер доставляет его в NATS. Это гарантирует «exactly-once» семантику при сбоях.

---

## 5. Сериализация

Проект использует **easyjson** (github.com/mailru/easyjson) — кодогенерируемую библиотеку JSON-сериализации для Go, обеспечивающую производительность, значительно превышающую стандартную `encoding/json` за счёт отсутствия рефлексии.

Все DTO аннотированы `//easyjson:json`. В обработчиках NATS используются методы `UnmarshalEasyJSON` / `RespondEasyJSON`.

Формат всех NATS-ответов (от `libs/dto/response.go`):
```json
{ "success": true/false, "data": {...}, "error": "error_code" }
```

---

## 6. Аутентификация и авторизация

**Схема:** JWT с HttpOnly cookies (защита от XSS)
- **Access token:** 15 минут, передаётся в HttpOnly cookie
- **Refresh token:** 168 часов, HttpOnly cookie + хранится в Redis
- **Fingerprint cookie:** дополнительная защита от CSRF/Cookie theft; при каждом запросе Gateway выполняет `auth.validate` через NATS
- **WebSocket:** `skip_fingerprint: true` при handshake

**RBAC:** роли `user` / `admin`. Роль передаётся в NATS-сообщениях через заголовок `X-User-Role`.

---

## 7. Observability (наблюдаемость)

### Распределённое трассирование
Каждый Go-сервис инициализирует `libs/telemetry.InitTracer()` с экспортом трасс в Jaeger через OTLP HTTP (`jaeger:4318`). Библиотека `natsw.Client` автоматически пропагирует OpenTelemetry trace context через заголовки NATS-сообщений, что обеспечивает сквозную трассировку запроса от Gateway до бэкенд-сервиса.

**Jaeger UI:** `http://localhost:16686`

### Метрики
Prometheus собирает метрики со всех сервисов. Grafana (`http://localhost:3001`) предоставляет дашборды.

---

## 8. Фронтенд

Два независимых React-приложения в рамках Bun-монорепозитория (`frontend/`):
- **`user-app`** — клиентское приложение: просмотр документов, закладки, чат с ботом
- **`admin-panel`** — административная панель: управление документами, пользователями, модерация регистраций

Пакетный менеджер: **Bun**. Общие пакеты вынесены в `frontend/packages/`.

---

## 9. Процесс индексации документа (end-to-end)

```
1. Администратор загружает документ через admin-panel
2. Gateway → NATS: documents.create → documents-service
3. documents-service:
   а. Сохраняет содержимое в MinIO → получает content_path
   б. Сохраняет метаданные в PostgreSQL (транзакция + Outbox)
   в. Outbox-воркер асинхронно публикует corpus.document.created
      { document_id, title, content_path, ... }
4. index-python получает событие:
   а. Загружает содержимое из MinIO по content_path
   б. Разбивает текст на чанки (размер 700, перекрытие 80 токенов)
   в. Генерирует эмбеддинги через Ollama (embeddinggemma:300m)
   г. Записывает векторы в Qdrant
5. Документ доступен для семантического поиска в чат-боте
```

---

## 10. RAG-Pipeline (процесс ответа чатбота)

```
1. Пользователь отправляет вопрос через WebSocket
2. Gateway → chat-service (WebSocket проксирование)
3. chat-service:
   а. Генерирует эмбеддинг вопроса через Ollama
   б. Выполняет векторный поиск в Qdrant (top-K чанков)
   в. Формирует prompt: system + найденные фрагменты + история + вопрос
   г. Отправляет запрос в Ollama (gemma3:4b, streaming)
   д. Стримит токены ответа клиенту через WebSocket
   е. Сохраняет сессию и историю диалога
```

---

## 11. Технологический стек (итоговая таблица)

| Слой | Технология | Версия / Детали |
|---|---|---|
| Основной язык бэкенда | Go | workspace (`go.work`) |
| HTTP-фреймворк | Fiber | v2 |
| ML-сервис | Python + asyncio | |
| Конфигурация (Python) | pydantic-settings | Двойной `__` для вложенных секций |
| Frontend | React | Bun workspace |
| Брокер сообщений | NATS | JetStream для персистентности |
| Реляционная БД | PostgreSQL | v17 |
| Кэш | Redis | v7 |
| Объектное хранилище | MinIO | S3-совместимый API |
| Векторная БД | Qdrant | Cosine distance |
| LLM Inference | Ollama | gemma3:4b + embeddinggemma:300m |
| JSON-сериализация (Go) | easyjson | Кодогенерация без рефлексии |
| Трассировка | OpenTelemetry → Jaeger | OTLP HTTP |
| Мониторинг | Prometheus + Grafana | |
| Контейнеризация | Docker + Docker Compose | |

---

## 12. Применённые архитектурные паттерны (список)

| Паттерн | Где применяется |
|---|---|
| API Gateway | services/gateway — единая точка входа |
| Микросервисная архитектура | 5 Go-сервисов + 1 Python-сервис |
| Event-Driven Architecture (EDA) | NATS publish/subscribe между сервисами |
| Request-Reply (NATS RPC) | Gateway → бэкенд-сервисы |
| Claim Check | Передача содержимого документов через MinIO |
| Outbox Pattern | Надёжная доставка событий из documents-service |
| Clean Architecture | Внутренняя структура каждого сервиса |
| Repository Pattern | Слой доступа к данным (PostgreSQL) |
| RBAC | Авторизация через роли user/admin |
| HttpOnly Cookie + Fingerprint | Защита JWT-сессий |
| Retrieval-Augmented Generation (RAG) | Чат-бот с поиском по корпусу текстов |

---

## 13. Ссылки на использованную литературу (основа)

- Hohpe G., Woolf B. *Enterprise Integration Patterns*. Addison-Wesley, 2003. — Claim Check, Publisher-Subscriber, Request-Reply.
- Richardson C. *Microservices Patterns*. Manning, 2018. — Outbox Pattern, API Gateway, Event-Driven Architecture.
- Lewis J., Fowler M. *Microservices* (martinfowler.com, 2014) — принципы декомпозиции на микросервисы.
- OpenTelemetry specification — распределённое трассирование (W3C TraceContext).
- Lewis P. et al. *Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks*. NeurIPS, 2020. — архитектура RAG.

