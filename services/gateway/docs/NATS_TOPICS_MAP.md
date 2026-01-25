# NATS Topics - Полная карта микросервисов

## Обзор

Этот документ содержит полную карту всех NATS топиков, используемых в системе для взаимодействия между Gateway и микросервисами.

## Архитектура

```
┌─────────────┐
│   Gateway   │
│  (Fiber)    │
└──────┬──────┘
       │
       ├─ NATS Request/Reply ─┐
       │                      │
       ▼                      ▼
┌─────────────┐      ┌─────────────┐      ┌──────────────┐
│Auth Service │      │Users Service│      │Docs Service  │
│  (Go/NATS)  │      │  (Go/NATS)  │      │  (Go/NATS)   │
└─────────────┘      └─────────────┘      └──────────────┘
```

---

## Auth Service (auth.*)

Отвечает за аутентификацию и авторизацию пользователей.

| Топик | Описание | Request | Response |
|-------|----------|---------|----------|
| `auth.register` | Создание заявки на регистрацию | `{username, email, password, metadata}` | `{request_id, status, message}` |
| `auth.login` | Вход в систему | `{email, password, device_id, user_agent, ip}` | `{access_token, refresh_token, token_id, fingerprint, expires_in}` |
| `auth.refresh` | Обновление access token | `{refresh_token, token_id, fingerprint, device_id}` | `{access_token, refresh_token, expires_in}` |
| `auth.validate` | Валидация access token | `{token, fingerprint}` | `{valid, user_id, role, jti}` |
| `auth.logout` | Выход с текущего устройства | `{user_id, token}` | `{success}` |
| `auth.logout_all` | Выход со всех устройств | `{user_id, token}` | `{success}` |
| `auth.change_password` | Смена пароля | `{user_id, old_password, new_password}` | `{success}` |

**Используется Gateway для:**
- Аутентификации пользователей (login)
- Валидации токенов (middleware)
- Обновления токенов (refresh)
- Управления сессиями (logout)

---

## Users Service (users.*)

Отвечает за управление пользователями и заявками на регистрацию.

### Управление пользователями

| Топик | Описание | Request | Response |
|-------|----------|---------|----------|
| `users.list` | Список пользователей | `{page, page_size, search, is_active}` | `{users[], total_count, page, page_size}` |
| `users.get` | Получение пользователя | `{id}` | `{user}` |
| `users.update` | Обновление пользователя | `{id, updates}` | `{user}` |
| `users.delete` | Удаление пользователя | `{id}` | `{success}` |

### Управление заявками на регистрацию

| Топик | Описание | Request | Response |
|-------|----------|---------|----------|
| `users.registration.create` | Создание заявки | `{username, email, password, metadata}` | `{request_id, status, message}` |
| `users.registration.list` | Список заявок | `{page, page_size, status}` | `{requests[], total_count, page, page_size}` |
| `users.registration.approve` | Одобрение заявки | `{request_id, approver_id}` | `{success, message, user}` |
| `users.registration.reject` | Отклонение заявки | `{request_id, approver_id, reason}` | `{success, message}` |

**Используется Gateway для:**
- CRUD операций с пользователями
- Создания заявок на регистрацию (публичный endpoint)
- Управления заявками (approve/reject) - только для админов

---

## Documents Service (documents.*)

Отвечает за управление документами (научные статьи, публикации).

| Топик | Описание | Request | Response |
|-------|----------|---------|----------|
| `documents.list` | Список документов | `{page, page_size, search, category_id, author_id, tags[], date_from, date_to}` | `{documents[], total_count, page, page_size}` |
| `documents.get` | Получение документа | `{id}` | `{document}` |
| `documents.create` | Создание документа | `{title, description, author_id, category_id, publication_date, tags[]}` | `{document}` |
| `documents.update` | Обновление документа | `{id, updates}` | `{document}` |
| `documents.delete` | Удаление документа | `{id}` | `{success}` |

**Используется Gateway для:**
- CRUD операций с документами
- Поиска и фильтрации документов
- Управления метаданными (categories, tags)

---

## Общий формат ответов

Все сервисы используют единый формат ответов:

### Успешный ответ
```json
{
  "success": true,
  "data": { ... }
}
```

### Ошибка
```json
{
  "success": false,
  "error": "error_code"
}
```

---

## Gateway REST → NATS Mapping

### Auth Endpoints
```
POST   /api/v1/auth/login              → auth.login
POST   /api/v1/auth/refresh            → auth.refresh
POST   /api/v1/auth/validate           → auth.validate
POST   /api/v1/auth/logout             → auth.logout
POST   /api/v1/auth/logout-all         → auth.logout_all
POST   /api/v1/auth/change-password    → auth.change_password
```

### Users Endpoints
```
GET    /api/v1/users                   → users.list
GET    /api/v1/users/:id               → users.get
PATCH  /api/v1/users/:id               → users.update
DELETE /api/v1/users/:id               → users.delete
```

### Registration Requests Endpoints
```
POST   /api/v1/registration-requests           → users.registration.create (PUBLIC)
GET    /api/v1/registration-requests           → users.registration.list
POST   /api/v1/registration-requests/:id/approve → users.registration.approve
POST   /api/v1/registration-requests/:id/reject  → users.registration.reject
```

### Documents Endpoints
```
GET    /api/v1/documents               → documents.list
GET    /api/v1/documents/:id           → documents.get
POST   /api/v1/documents               → documents.create
PATCH  /api/v1/documents/:id           → documents.update
DELETE /api/v1/documents/:id           → documents.delete
```

---

## Trace Context Propagation

Все NATS запросы автоматически включают OpenTelemetry trace context через библиотеку `natsw.Client`. Это позволяет отслеживать запросы через всю цепочку микросервисов.

**Пример trace:**
```
Gateway → auth.validate → Auth Service → Redis/PostgreSQL
  ↓
Gateway → users.list → Users Service → PostgreSQL
  ↓
Gateway → documents.get → Documents Service → PostgreSQL
```

---

## Timeout Configuration

**Gateway NATS Request Timeout:** 5 секунд (по умолчанию)

Рекомендуется настроить таймауты для каждого топика:
- `auth.validate`: 1s (быстрая операция)
- `users.list`: 5s (может быть медленной с поиском)
- `documents.list`: 10s (сложные запросы с JOIN)
- `registration.approve`: 10s (транзакция + создание пользователя)

---

## Error Handling

### Gateway обрабатывает ошибки:
1. **Timeout** - если сервис не отвечает в течение 5 секунд
2. **Service Error** - если сервис вернул `success: false`
3. **NATS Error** - если NATS недоступен

**Пример обработки:**
```go
resp, err := s.userConnector.ListUsers(ctx, query)
if err != nil {
    // NATS error or timeout
    return c.Status(503).JSON(fiber.Map{
        "error": "service_unavailable",
        "message": "Users service is temporarily unavailable"
    })
}
```

---

## Security

### Auth Middleware
Gateway валидирует все защищенные запросы через `auth.validate` топик перед передачей в другие сервисы.

**Flow:**
```
1. Client → Gateway: GET /api/v1/users (+ Bearer token + fingerprint cookie)
2. Gateway Middleware → NATS: auth.validate {token, fingerprint}
3. Auth Service → Gateway: {valid: true, user_id, role}
4. Gateway → NATS: users.list {page, page_size}
5. Users Service → Gateway: {users[], total_count}
6. Gateway → Client: {users[], total_count}
```

### Публичные топики
Только `users.registration.create` доступен без авторизации.

---

## Monitoring & Metrics

Рекомендуется отслеживать:
- Количество запросов к каждому топику
- Время ответа каждого топика
- Количество ошибок по топикам
- Количество timeout'ов
- Размер payload запросов/ответов

**Prometheus metrics пример:**
```
nats_request_total{topic="users.list", status="success"}
nats_request_duration_seconds{topic="users.list", quantile="0.99"}
nats_request_errors_total{topic="users.list", error="timeout"}
```

---

## Testing

### Тестирование через NATS CLI

```bash
# Валидация токена
nats request auth.validate '{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "fingerprint": "abc123"
}'

# Список пользователей
nats request users.list '{
  "page": 1,
  "page_size": 10
}'

# Создание заявки на регистрацию
nats request users.registration.create '{
  "username": "testuser",
  "email": "test@example.com",
  "password": "SecurePassword123"
}'
```

---

## Документация

- [Auth Service](../../auth/README.md)
- [Users Service Spec](./USERS_SERVICE_SPEC.md)
- [Documents Service Spec](./DOCUMENTS_SERVICE_SPEC.md)
- [Gateway NATS Integration](./users-service-integration.md)

---

## Roadmap

Планируемые топики:
- `notifications.*` - сервис уведомлений
- `search.*` - полнотекстовый поиск
- `analytics.*` - аналитика и статистика
- `files.*` - загрузка и управление файлами
