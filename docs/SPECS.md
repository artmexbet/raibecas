# Техническое задание: Цифровой Райбекас (v2.0)

## 1. Обзор проекта

**Цель**: Создать веб-платформу для работы с научными текстами философа А.Я. Райбекаса с интеграцией локального AI-ассистента на основе RAG.

**Ключевые фичи**:
- Хранение и отображение Markdown-документов с метаданными
- Интеллектуальное сохранение фрагментов текста с устойчивыми ссылками
- RAG-based чат-бот с локальными моделями (Ollama)
- Система модерации регистраций
- Приватные заметки пользователей
- Полнотекстовый и семантический поиск

**Технологический стек**:

### Backend
- **Языки**: Go (основные сервисы), Python (ML Pipeline)
- **API Gateway**: Go + Fiber
- **Message Broker**: NATS (или RabbitMQ)
- **Databases**:
    - PostgreSQL (основная БД)
    - pgvector (векторные эмбеддинги)
    - Redis (кэш, сессии)
- **Search**: Meilisearch
- **LLM**: Ollama (Llama 3.1, Mistral)
- **Embeddings**: sentence-transformers (all-MiniLM-L6-v2)

### Frontend
- **Framework**: SvelteKit
- **State Management**: Svelte stores
- **Styling**: TailwindCSS
- **Build**: Vite

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Reverse Proxy**: Nginx
- **Monitoring**: Prometheus + Grafana
- **Logging**: Loki

---

## 2. Архитектура системы

### 2.1. Микросервисная архитектура с Event-Driven подходом

```
┌─────────────────────────────────────────────────────────────┐
│                    NGINX (Reverse Proxy)                    │
└────────────────┬───────────────────────────┬────────────────┘
                 │                           │
    ┌────────────▼────────┐         ┌───────▼────────┐
    │  SvelteKit Frontend │         │  Static Assets │
    └────────────┬────────┘         └────────────────┘
                 │
    ┌────────────▼──────────────────────────────────────────┐
    │          API Gateway (Go + Fiber)                     │
    │  - Authentication/Authorization (JWT)                 │
    │  - Rate Limiting                                      │
    │  - Request Validation                                 │
    │  - WebSocket handler (для чата)                       │
    └────┬──────────┬──────────────────┬────────────────────┘
         │          │                  │
         │  ┌───────▼──────────────┐   │
         │  │   NATS Message       │   │
         │  │      Broker          │◄──┼──────────────┐
         │  └───────┬──────────────┘   │              │
         │          │                  │              │
    ┌────▼────┐ ┌──▼────────┐ ┌──────▼──────┐ ┌────▼─────────┐
    │  Auth   │ │  Corpus   │ │    User     │ │   Indexing   │
    │ Service │ │  Service  │ │   Service   │ │   Service    │
    │  (Go)   │ │   (Go)    │ │    (Go)     │ │   (Python)   │
    └────┬────┘ └─────┬─────┘ └──────┬──────┘ └──────┬───────┘
         │            │              │                │
         └────────────┴──────────────┴────────────────┘
                            │
          ┌─────────────────┴─────────────────────────┐
          │                                           │
    ┌─────▼──────────┐  ┌──────────────┐  ┌─────────▼────────┐
    │  PostgreSQL    │  │ Meilisearch  │  │   Redis Cache    │
    │  (+ pgvector)  │  │   (Search)   │  │   (Sessions)     │
    └────────────────┘  └──────────────┘  └──────────────────┘
                            │
              ┌─────────────▼─────────────┐
              │                           │
    ┌─────────▼────────┐      ┌──────────▼────────┐
    │   Chat Service   │      │   Ollama Server   │
    │    (Python)      │──────►  (LLM + Embed.)   │
    └──────────────────┘      └───────────────────┘
```

### 2.2. Потоки данных (Event-Driven)

#### Пример 1: Загрузка нового документа

```
Admin UI → API Gateway → Corpus Service
                              ↓
                    Publish: document.created
                              ↓
                         NATS Broker
                         ↙          ↘
        Indexing Service          Search Service
        (создаёт embeddings)      (индексирует в Meilisearch)
                ↓                        ↓
          Vector DB              Meilisearch
                ↓
    Publish: document.indexed
                ↓
          Chat Service
      (подтверждает готовность)
```

#### Пример 2: Отправка сообщения в чат

```
User → WebSocket (API Gateway) → Chat Service
                                       ↓
                              1. Semantic Search
                              2. Context Building
                              3. LLM Query (Ollama)
                                       ↓
                              Publish: message.created
                                       ↓
                                  NATS Broker
                                       ↓
                               User Service
                          (сохраняет в историю)
                                       ↓
                        WebSocket Response → User
```

---

## 3. Сервисы и их ответственность

### 3.1. API Gateway (Go)
**Порт**: 8080  
**Ответственность**:
- Единая точка входа для всех HTTP/WebSocket запросов
- JWT аутентификация и авторизация
- Rate limiting (per user, per endpoint)
- Маршрутизация запросов к соответствующим сервисам
- WebSocket handler для real-time чата
- CORS, security headers

**REST Endpoints**:
- `/api/v1/auth/*` → Auth Service
- `/api/v1/documents/*` → Corpus Service
- `/api/v1/fragments/*` → User Service
- `/api/v1/notes/*` → User Service
- `/api/v1/search/*` → Corpus Service
- `/api/v1/admin/*` → Admin operations (multiple services)

**WebSocket**:
- `/ws/chat` → Chat Service (через NATS)

**Dependencies**:
- NATS (для pub/sub)
- Redis (для rate limiting и sessions)

---

### 3.2. Auth Service (Go)
**Порт**: 8081  
**Ответственность**:
- Регистрация (создание запросов)
- Аутентификация (login/logout)
- JWT token management (issue, refresh, revoke)
- Password reset (через админа)
- User session management

**Database Tables**:
- `users`
- `registration_requests`
- `refresh_tokens`

**Published Events**:
- `user.registered` (username, user_id)
- `user.login` (user_id, timestamp)
- `user.logout` (user_id)
- `password.reset` (user_id)

**Subscribed Events**:
- `registration.approved` (от Admin Service)
- `registration.rejected` (от Admin Service)

**REST API**:
- `POST /register` - создать запрос на регистрацию
- `POST /login` - вход
- `POST /logout` - выход
- `POST /refresh` - обновить токен
- `POST /change-password` - смена пароля

---

### 3.3. Corpus Service (Go)
**Порт**: 8082  
**Ответственность**:
- CRUD операции с документами
- Управление авторами/философами
- Версионирование документов
- Полнотекстовый поиск (через Meilisearch)
- Управление метаданными

**Database Tables**:
- `authors`
- `documents`
- `document_versions`

**Published Events**:
- `document.created` (document_id, content, metadata)
- `document.updated` (document_id, old_version, new_version, content)
- `document.deleted` (document_id)
- `author.created` (author_id, name)

**Subscribed Events**:
- `document.indexed` (от Indexing Service) - помечает документ как проиндексированный

**REST API**:
- `GET /documents` - список документов (с фильтрами)
- `GET /documents/:id` - получить документ
- `POST /documents` (admin) - создать документ
- `PUT /documents/:id` (admin) - обновить документ
- `DELETE /documents/:id` (admin) - удалить документ
- `GET /documents/:id/versions` - история версий
- `GET /search` - полнотекстовый поиск

---

### 3.4. User Service (Go)
**Порт**: 8083  
**Ответственность**:
- Управление фрагментами (сохранение, поиск, sharing)
- Приватные заметки
- История чата (хранение)
- Профиль пользователя

**Database Tables**:
- `fragments`
- `notes`
- `chat_sessions`
- `chat_messages`

**Published Events**:
- `fragment.created` (fragment_id, user_id, document_id)
- `fragment.deleted` (fragment_id)
- `note.created` (note_id, user_id)
- `chat.history_deleted` (user_id) - для GDPR compliance

**Subscribed Events**:
- `message.created` (от Chat Service) - сохраняет в историю
- `document.updated` (от Corpus Service) - проверяет фрагменты на актуальность

**REST API**:
- `GET /fragments` - мои фрагменты
- `POST /fragments` - сохранить фрагмент
- `GET /fragments/:token` - получить по share-ссылке
- `DELETE /fragments/:id` - удалить фрагмент
- `GET /notes` - мои заметки
- `POST /notes` - создать заметку
- `PUT /notes/:id` - обновить заметку
- `DELETE /notes/:id` - удалить заметку
- `GET /chat/sessions` - история сессий
- `GET /chat/sessions/:id` - сообщения сессии
- `DELETE /chat/sessions/:id` - удалить сессию
- `DELETE /chat/history` - удалить всю историю

---

### 3.5. Indexing Service (Python)
**Порт**: 8084  
**Ответственность**:
- Разбивка документов на чанки (chunking)
- Генерация эмбеддингов (через Ollama)
- Индексация в векторную БД (pgvector)
- Индексация в Meilisearch
- Batch processing для больших документов

**Database Tables**:
- `document_embeddings` (через pgvector)

**Published Events**:
- `document.indexed` (document_id, chunks_count, status)
- `indexing.failed` (document_id, error)

**Subscribed Events**:
- `document.created` (от Corpus Service)
- `document.updated` (от Corpus Service)
- `document.deleted` (от Corpus Service)

**Dependencies**:
- Ollama (для эмбеддингов: nomic-embed-text или all-minilm)
- PostgreSQL + pgvector
- Meilisearch

**REST API** (internal):
- `POST /index-document` - ручная индексация
- `GET /index-status/:document_id` - статус индексации

---

### 3.6. Chat Service (Python)
**Порт**: 8085  
**Ответственность**:
- Обработка запросов к чат-боту
- RAG: semantic search + context building
- Взаимодействие с Ollama LLM
- Управление промптами (популярный/академический режим)
- Streaming ответов через WebSocket

**Published Events**:
- `message.created` (session_id, role, content, sources, timestamp)
- `chat.session_started` (session_id, user_id, mode)

**Subscribed Events**:
- `chat.query` (от API Gateway через WebSocket)

**Dependencies**:
- Ollama (LLM: llama3.1, mistral)
- PostgreSQL + pgvector (для semantic search)
- Redis (для кэширования контекста)

**WebSocket Protocol**:
```json
// Client → Server
{
  "type": "chat.message",
  "session_id": "uuid",
  "message": "Что такое диалектика?",
  "mode": "popular"
}

// Server → Client (streaming)
{
  "type": "chat.response.chunk",
  "session_id": "uuid",
  "chunk": "Диалектика — это...",
  "done": false
}

// Server → Client (final)
{
  "type": "chat.response.complete",
  "session_id": "uuid",
  "sources": [
    {
      "document_id": "uuid",
      "document_title": "Основы диалектики",
      "fragment": "...",
      "relevance": 0.89
    }
  ]
}
```

---

### 3.7. Admin Service (Go) - опционально
**Альтернатива**: Админские операции могут быть распределены по другим сервисам с проверкой роли.

Если выделять отдельно:
- Управление пользователями
- Модерация регистраций
- Управление системными настройками
- Мониторинг и логи

---

## 4. Message Broker (NATS)

### 4.1. Почему NATS?
- ✅ Легковесный и быстрый
- ✅ Встроенная поддержка pub/sub, request-reply, queueing
- ✅ JetStream для персистентности событий
- ✅ Простая интеграция с Go и Python
- ✅ Меньше overhead чем RabbitMQ/Kafka

### 4.2. Топология subjects

```
# Authentication
auth.user.registered
auth.user.login
auth.user.logout
auth.password.reset

# Documents
corpus.document.created
corpus.document.updated
corpus.document.deleted
corpus.author.created

# Indexing
indexing.document.indexed
indexing.document.failed

# User actions
user.fragment.created
user.fragment.deleted
user.note.created
user.chat.history_deleted

# Chat
chat.message.created
chat.session.started
```

### 4.3. JetStream для критичных событий

**Персистентные стримы**:
- `DOCUMENTS` - все события с документами
- `CHAT_HISTORY` - история чата (для восстановления)
- `AUDIT_LOG` - аудит действий админов

**Retention policy**: 30 дней или 10GB (что наступит раньше)

---

## 5. База данных (PostgreSQL)

### 5.1. Основные таблицы

```sql
-- Users & Auth
users
registration_requests
refresh_tokens

-- Corpus
authors
documents
document_versions

-- User Content
fragments
notes
chat_sessions
chat_messages

-- Vectors (pgvector)
document_embeddings
```

### 5.2. Индексы для производительности

```sql
-- Поиск фрагментов
CREATE INDEX idx_fragments_user ON fragments(user_id);
CREATE INDEX idx_fragments_document ON fragments(document_id);
CREATE INDEX idx_fragments_share_token ON fragments(share_token);

-- Векторный поиск
CREATE INDEX idx_embedding_search ON document_embeddings 
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Поиск по тегам
CREATE INDEX idx_documents_tags ON documents USING GIN (tags);

-- История чата
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_chat_sessions_user ON chat_sessions(user_id);
```

---

## 6. Кэширование (Redis)

### 6.1. Use cases

**Sessions**:
- `session:{user_id}` - JWT refresh tokens
- TTL: 7 дней

**Rate Limiting**:
- `ratelimit:{user_id}:{endpoint}` - счётчик запросов
- TTL: 1 минута

**Chat Context Cache**:
- `chat:context:{session_id}` - последние N сообщений для контекста
- TTL: 1 час

**Document Cache**:
- `document:{document_id}` - часто запрашиваемые документы
- TTL: 10 минут

---

## 7. Frontend (SvelteKit)

### 7.1. Структура проекта

```
frontend/
├── src/
│   ├── routes/
│   │   ├── +page.svelte                  # Главная
│   │   ├── login/
│   │   │   └── +page.svelte
│   │   ├── register/
│   │   │   └── +page.svelte
│   │   ├── corpus/
│   │   │   ├── +page.svelte              # Список документов
│   │   │   └── [id]/
│   │   │       └── +page.svelte          # Просмотр документа
│   │   ├── chat/
│   │   │   └── +page.svelte              # Чат с философом
│   │   ├── profile/
│   │   │   ├── fragments/
│   │   │   ├── notes/
│   │   │   └── history/
│   │   └── admin/
│   │       ├── requests/
│   │       ├── users/
│   │       └── documents/
│   ├── lib/
│   │   ├── components/
│   │   │   ├── layout/
│   │   │   │   ├── Header.svelte
│   │   │   │   ├── Sidebar.svelte
│   │   │   │   └── Footer.svelte
│   │   │   ├── corpus/
│   │   │   │   ├── DocumentCard.svelte
│   │   │   │   ├── DocumentViewer.svelte
│   │   │   │   ├── FragmentHighlighter.svelte
│   │   │   │   └── SearchBar.svelte
│   │   │   ├── chat/
│   │   │   │   ├── ChatInterface.svelte
│   │   │   │   ├── MessageList.svelte
│   │   │   │   ├── ChatInput.svelte
│   │   │   │   └── ModeToggle.svelte
│   │   │   └── admin/
│   │   ├── stores/
│   │   │   ├── auth.ts
│   │   │   ├── documents.ts
│   │   │   ├── chat.ts
│   │   │   └── user.ts
│   │   ├── api/
│   │   │   ├── client.ts              # Axios/Fetch wrapper
│   │   │   ├── auth.ts
│   │   │   ├── corpus.ts
│   │   │   ├── chat.ts
│   │   │   └── user.ts
│   │   ├── utils/
│   │   │   ├── markdown.ts
│   │   │   ├── websocket.ts
│   │   │   └── fragments.ts
│   │   └── types/
│   │       └── index.ts
│   ├── app.html
│   └── app.css
├── static/
├── svelte.config.js
├── vite.config.ts
└── package.json
```

### 7.2. Ключевые особенности

**SSR/SPA Hybrid**:
- Статические страницы (главная, о проекте) - SSG
- Динамический контент (документы, чат) - CSR
- SEO-критичные страницы - SSR

**WebSocket для чата**:
- Реактивное подключение через Svelte store
- Автоматический reconnect
- Streaming ответов от LLM

**Svelte Stores для состояния**:
- `authStore` - текущий пользователь, токены
- `documentsStore` - список документов, кэш
- `chatStore` - активная сессия, сообщения
- `fragmentsStore` - сохранённые фрагменты

---

## 8. Ollama Integration

### 8.1. Модели

**LLM для чата**:
- `llama3.1:8b` (рекомендуемая, баланс скорость/качество)
- `mistral:7b` (альтернатива)
- `llama3.1:70b` (если есть мощное железо)

**Embeddings**:
- `nomic-embed-text` (768 dimensions, оптимизирована для RAG)
- Альтернатива: `all-minilm:l6-v2` (384 dimensions, быстрее)

### 8.2. Конфигурация

**Ollama Server**:
```yaml
# docker-compose.yml
ollama:
  image: ollama/ollama:latest
  ports:
    - "11434:11434"
  volumes:
    - ollama_models:/root/.ollama
  deploy:
    resources:
      reservations:
        devices:
          - driver: nvidia
            count: 1
            capabilities: [gpu]
```

**Preload моделей**:
```bash
docker exec ollama ollama pull llama3.1:8b
docker exec ollama ollama pull nomic-embed-text
```

### 8.3. API взаимодействие

**Генерация ответа** (Chat Service → Ollama):
```
POST http://ollama:11434/api/chat
{
  "model": "llama3.1:8b",
  "messages": [...],
  "stream": true,
  "options": {
    "temperature": 0.7,
    "top_p": 0.9,
    "num_ctx": 4096
  }
}
```

**Генерация эмбеддингов** (Indexing Service → Ollama):
```
POST http://ollama:11434/api/embeddings
{
  "model": "nomic-embed-text",
  "prompt": "текст для эмбеддинга"
}
```

---

## 9. Deployment

### 9.1. Docker Compose (Production)

```yaml
version: '3.8'

services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: raibecas
      POSTGRES_USER: raibecas
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready"]
      interval: 10s

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data

  nats:
    image: nats:latest
    command: 
      - "-js"
      - "-sd=/data"
    ports:
      - "4222:4222"
      - "8222:8222"  # monitoring
    volumes:
      - nats_data:/data

  meilisearch:
    image: getmeili/meilisearch:v1.5
    environment:
      MEILI_MASTER_KEY: ${MEILI_MASTER_KEY}
    volumes:
      - meilisearch_data:/meili_data

  ollama:
    image: ollama/ollama:latest
    volumes:
      - ollama_models:/root/.ollama
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]

  # Микросервисы
  api-gateway:
    build: ./services/api-gateway
    environment:
      NATS_URL: nats://nats:4222
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "8080:8080"
    depends_on:
      - nats
      - redis

  auth-service:
    build: ./services/auth-service
    environment:
      DATABASE_URL: postgres://raibecas:${DB_PASSWORD}@postgres:5432/raibecas
      NATS_URL: nats://nats:4222
    depends_on:
      - postgres
      - nats

  corpus-service:
    build: ./services/corpus-service
    environment:
      DATABASE_URL: postgres://raibecas:${DB_PASSWORD}@postgres:5432/raibecas
      NATS_URL: nats://nats:4222
      MEILISEARCH_URL: http://meilisearch:7700
      MEILISEARCH_KEY: ${MEILI_MASTER_KEY}
    depends_on:
      - postgres
      - nats
      - meilisearch

  user-service:
    build: ./services/user-service
    environment:
      DATABASE_URL: postgres://raibecas:${DB_PASSWORD}@postgres:5432/raibecas
      NATS_URL: nats://nats:4222
    depends_on:
      - postgres
      - nats

  indexing-service:
    build: ./services/indexing-service
    environment:
      DATABASE_URL: postgres://raibecas:${DB_PASSWORD}@postgres:5432/raibecas
      NATS_URL: nats://nats:4222
      OLLAMA_URL: http://ollama:11434
      MEILISEARCH_URL: http://meilisearch:7700
    depends_on:
      - postgres
      - nats
      - ollama
      - meilisearch

  chat-service:
    build: ./services/chat-service
    environment:
      DATABASE_URL: postgres://raibecas:${DB_PASSWORD}@postgres:5432/raibecas
      NATS_URL: nats://nats:4222
      OLLAMA_URL: http://ollama:11434
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379
    depends_on:
      - postgres
      - nats
      - ollama
      - redis

  frontend:
    build: ./frontend
    environment:
      PUBLIC_API_URL: http://api-gateway:8080
      PUBLIC_WS_URL: ws://api-gateway:8080
    depends_on:
      - api-gateway

  nginx:
    image: nginx:alpine
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - frontend

volumes:
  postgres_data:
  redis_data:
  nats_data:
  meilisearch_data:
  ollama_models:
```

### 9.2. Hardware Requirements

**Минимальные (для разработки)**:
- CPU: 4 cores
- RAM: 16GB
- GPU: 6GB VRAM (для llama3.1:8b)
- Disk: 50GB SSD

**Рекомендуемые (production)**:
- CPU: 8+ cores
- RAM: 32GB
- GPU: RTX 3090 / 4090 или A100 (24GB VRAM)
- Disk: 100GB NVMe SSD

---

## 10. Мониторинг и Observability

### 10.1. Prometheus метрики

**Каждый сервис экспортирует**:
- HTTP request duration (histogram)
- HTTP request count (counter)
- Active connections (gauge)
- NATS message processing time
- Database query duration
- Error rate

**Ollama специфичные**:
- Token generation speed (tokens/sec)
- Model loading time
- GPU utilization
- Context window usage

### 10.2. Grafana дашборды

1. **System Overview**: CPU, RAM, Disk, Network
2. **Services Health**: Request rate, errors, latency
3. **NATS Broker**: Message throughput, queue depth
4. **Database**: Query performance, connection pool
5. **Ollama**: Model performance, GPU metrics
6. **User Activity**: Active users, chat sessions, documents viewed

### 10.3. Distributed Tracing

**Опционально**: Jaeger или Tempo для трейсинга запросов через микросервисы

---

## 11. Безопасность

### 11.1. Аутентификация
- JWT access tokens (15 мин) + refresh tokens (7 дней)
- bcrypt для паролей (cost=12)
- Redis для хранения refresh tokens

### 11.2. Авторизация
- RBAC: `user` | `admin`
- Middleware в API Gateway
- Service-to-service authentication через NATS credentials

### 11.3. Network Security
- Все сервисы в приватной Docker сети
- Только Nginx и API Gateway exposed
- NATS authentication
- PostgreSQL SSL connections

### 11.4. Rate Limiting
- User: 100 req/min (API), 10 msg/min (chat)
- Admin: 1000 req/min
- IP-based для неавторизованных: 20 req/min

---

## 12. Масштабируемость

### 12.1. Horizontal Scaling

**Stateless сервисы** (легко масштабируются):
- Auth Service
- Corpus Service
- User Service
- API Gateway

**Stateful сервисы** (требуют координации):
- Chat Service (использует Redis для шаринга состояния)

### 12.2. Database Scaling

**Для будущего**:
- Read replicas для PostgreSQL
- Partitioning для больших таблиц (`chat_messages`)
- Separate DB для векторов (Qdrant/Weaviate)

### 12.3. Message Broker Scaling

NATS Cluster с JetStream для HA (3+ ноды)

---

## 13. Testing Strategy

### 13.1. Unit Tests
- Go: `go test` для бизнес-логики
- Python: `pytest` для ML pipeline
- Svelte: `vitest` для компонентов

### 13.2. Integration Tests
- Testcontainers для PostgreSQL, Redis, NATS
- Mock Ollama API

### 13.3. E2E Tests
- Playwright для критичных user flows
- CI/CD pipeline

---

## 14. Development Workflow

### 14.1. Local Development

```bash
# Поднять инфраструктуру
docker-compose -f docker-compose.dev.yml up -d

# Запустить сервисы в dev mode (с hot reload)
cd services/auth-service && air  # Go hot reload
cd services/chat-service && uvicorn main:app --reload
cd frontend && npm run dev
```

### 14.2. CI/CD

**GitHub Actions**:
1. Lint + Test на каждый PR
2. Build Docker images
3. Deploy to staging (on merge to `develop`)
4. Deploy to production (on tag)

---

## 15. Roadmap

### Phase 1: Инфраструктура и базовые сервисы (4-6 недель)
- ✅ Docker Compose setup
- ✅ PostgreSQL + pgvector schema
- ✅ NATS broker setup
- ✅ Auth Service (регистрация, логин)
- ✅ Corpus Service (CRUD документов)
- ✅ API Gateway
- ✅ SvelteKit frontend scaffold
- ✅ Basic UI (логин, просмотр документов)

### Phase 2: Core Features (4-6 недель)
- ✅ User Service (фрагменты, заметки)
- ✅ Fragment highlighting и sharing
- ✅ Meilisearch integration
- ✅ Admin panel (модерация регистраций)
- ✅ Адаптивный дизайн

### Phase 3: AI Integration (4-6 недель)
- ✅ Ollama setup
- ✅ Indexing Service (chunking, embeddings)
- ✅ Chat Service (RAG pipeline)
- ✅ WebSocket чат в UI
- ✅ Два режима ответов (промпты)
- ✅ История чата

### Phase 4: Polish & Production (2-3 недели)
- ✅ Оптимизация производительности
- ✅ Comprehensive testing
- ✅ Monitoring setup (Prometheus + Grafana)
- ✅ Документация
- ✅ Production deployment

**Итого: 14-21 недель (~3.5-5 месяцев)**

---

## 16. Расширяемость (Multi-Philosopher Support)

### 16.1. Абстракция Автора

**Database**:
```sql
-- Авторы с расширяемыми метаданными
CREATE TABLE authors (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    bio TEXT,
    metadata JSONB,  -- Кастомные поля для каждого автора
    created_at TIMESTAMP DEFAULT NOW()
);

-- Конфигурация чат-бота для автора
CREATE TABLE author_chat_configs (
    author_id UUID REFERENCES authors(id),
    system_prompt TEXT NOT NULL,
    temperature FLOAT DEFAULT 0.7,
    model_name VARCHAR(100),  -- Разные модели для разных авторов
    vector_collection VARCHAR(100),  -- Отдельная коллекция эмбеддингов
    PRIMARY KEY (author_id)
);
```

### 16.2. Динамическая маршрутизация

**Frontend** (SvelteKit):
```
routes/
├── philosophers/
│   ├── [slug]/
│   │   ├── +page.svelte         # Страница философа
│   │   ├── documents/
│   │   │   └── [id]/+page.svelte
│   │   └── chat/
│   │       └── +page.svelte     # Чат с конкретным философом
```

**URL Examples**:
- `/philosophers/raibecas` - страница Райбекаса
- `/philosophers/raibecas/chat` - чат с Райбекасом
- `/philosophers/kant/chat` - в будущем: чат с Кантом

### 16.3. Изолированные коллекции эмбеддингов

При добавлении нового автора:
1. Создаётся новый namespace в pgvector: `embeddings_{author_slug}`
2. Индексируются только его тексты
3. RAG search ограничивается его контекстом

---

## Приложения

### A. Рекомендуемые библиотеки

**Go**:
- `github.com/gofiber/fiber/v2` - веб-фреймворк
- `github.com/nats-io/nats.go` - NATS client
- `github.com/jmoiron/sqlx` - PostgreSQL
- `github.com/golang-jwt/jwt/v5` - JWT
- `github.com/go-redis/redis/v8` - Redis
- `github.com/gofiber/websocket/v2` - WebSocket

**Python**:
- `fastapi` - веб-фреймворк
- `nats-py` - NATS client
- `httpx` - HTTP client для Ollama
- `sentence-transformers` - локальные эмбеддинги (fallback)
- `sqlalchemy` - ORM
- `websockets` - WebSocket server

**Frontend (Svelte)**:
- `svelte` + `sveltekit` - фреймворк
- `tailwindcss` - стили
- `marked` - Markdown рендеринг
- `highlight.js` - подсветка кода
- `@sveltejs/adapter-node` - SSR deployment

### B. Альтернативные решения

**RabbitMQ вместо NATS**:
- Если нужны более сложные routing patterns
- Больше overhead, но богаче функционал

**Qdrant/Weaviate вместо pgvector**:
- Если векторная БД станет узким местом
- Специализированные решения с лучшей производительностью
- Легко мигрировать: NATS event → переиндексация

**Monorepo vs Polyrepo**:
- Monorepo (рекомендуется): проще синхронизировать изменения
- Polyrepo: если команды работают независимо

---

## Заключение

Данная архитектура обеспечивает:
- ✅ **Настоящую микросервисную архитектуру** с асинхронной коммуникацией
- ✅ **Масштабируемость** (горизонтальное масштабирование stateless сервисов)
- ✅ **Отказоустойчивость** (благодаря event-driven подходу и JetStream)
- ✅ **Расширяемость** (легко добавлять новых философов/авторов)
- ✅ **Полная локальность** (Ollama, без зависимости от внешних API)
- ✅ **Современный стек** (Go, Python, SvelteKit, NATS)

Готов к реализации! 🚀