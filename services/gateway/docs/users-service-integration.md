# Users Service Integration

## Описание

Gateway интегрирован с Users Service через NATS для управления пользователями. Все операции с пользователями (CRUD) отправляются в отдельный микросервис users через NATS messaging.

## NATS Topics

### users.list
**Описание:** Получение списка пользователей с фильтрацией и пагинацией

**Request:**
```json
{
  "page": 1,
  "page_size": 10,
  "search": "john",
  "is_active": true
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "uuid",
        "email": "user@example.com",
        "username": "username",
        "full_name": "Full Name",
        "registered_at": "2024-01-01T00:00:00Z",
        "last_login_at": "2024-01-01T00:00:00Z",
        "is_active": true
      }
    ],
    "total_count": 100,
    "page": 1,
    "page_size": 10
  }
}
```

### users.get
**Описание:** Получение информации о конкретном пользователе

**Request:**
```json
{
  "id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "user@example.com",
      "username": "username",
      "full_name": "Full Name",
      "registered_at": "2024-01-01T00:00:00Z",
      "last_login_at": "2024-01-01T00:00:00Z",
      "is_active": true
    }
  }
}
```

### users.update
**Описание:** Обновление информации о пользователе

**Request:**
```json
{
  "id": "uuid",
  "updates": {
    "email": "newemail@example.com",
    "username": "newusername",
    "full_name": "New Full Name",
    "is_active": false
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid",
      "email": "newemail@example.com",
      "username": "newusername",
      "full_name": "New Full Name",
      "registered_at": "2024-01-01T00:00:00Z",
      "last_login_at": "2024-01-01T00:00:00Z",
      "is_active": false
    }
  }
}
```

### users.delete
**Описание:** Удаление пользователя

**Request:**
```json
{
  "id": "uuid"
}
```

**Response:**
```json
{
  "success": true
}
```

## REST API Endpoints

### GET /api/v1/users
Получение списка пользователей

**Query Parameters:**
- `page` (int, optional, default=1) - номер страницы
- `page_size` (int, optional, default=10, max=100) - размер страницы
- `search` (string, optional) - поиск по имени/email
- `is_active` (bool, optional) - фильтр по активности

**Response:** 200 OK
```json
{
  "users": [...],
  "total_count": 100,
  "page": 1,
  "page_size": 10
}
```

### GET /api/v1/users/:id
Получение информации о пользователе

**Response:** 200 OK
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "username": "username",
    "full_name": "Full Name",
    "registered_at": "2024-01-01T00:00:00Z",
    "last_login_at": "2024-01-01T00:00:00Z",
    "is_active": true
  }
}
```

### PATCH /api/v1/users/:id
Обновление информации о пользователе

**Request Body:**
```json
{
  "email": "newemail@example.com",
  "username": "newusername",
  "full_name": "New Full Name",
  "is_active": false
}
```

**Response:** 200 OK
```json
{
  "user": {
    "id": "uuid",
    "email": "newemail@example.com",
    ...
  }
}
```

### DELETE /api/v1/users/:id
Удаление пользователя

**Response:** 200 OK
```json
{
  "success": true,
  "message": "User deleted successfully"
}
```

## Архитектура

```
HTTP Request → Gateway (REST API)
                  ↓
            NATS Connector (users_connector.go)
                  ↓
            NATS Wrapper Client (natsw.Client) [shared]
                  ↓
            NATS Message Bus
                  ↓
            Users Service (будет реализован отдельно)
```

**Важно:** Все коннекторы (auth, documents, users) используют единый экземпляр `natsw.Client`, который создается в `App` и передается во все коннекторы. Это обеспечивает:
- Эффективное использование ресурсов (один экземпляр вместо нескольких)
- Единообразную пропагацию trace context
- Упрощенное управление NATS соединением

## Компоненты

### UserServiceConnector Interface
Интерфейс для взаимодействия с users service. Определяет контракт для всех операций с пользователями.

**Location:** `internal/server/user_connector.go`

### NATSUserConnector
Реализация UserServiceConnector через NATS messaging. Принимает готовый `natsw.Client` от App, что обеспечивает автоматическую пропагацию trace context и эффективное использование ресурсов.

**Location:** `internal/connector/users_connector.go`

**Важно:** Коннектор НЕ создает собственный экземпляр `natsw.Client`, а использует тот, что передан из App.

### Domain Models
Модели данных для работы с users service.

**Location:** `internal/domain/users.go`

**Models:**
- `ListUsersQuery` - параметры запроса списка пользователей
- `ListUsersResponse` - ответ со списком пользователей
- `UpdateUserRequest` - данные для обновления пользователя
- `UpdateUserResponse` - ответ после обновления
- `GetUserRequest` - запрос конкретного пользователя
- `GetUserResponse` - ответ с данными пользователя

## Требования к Users Service

Users Service должен подписаться на следующие NATS топики и обрабатывать запросы:
- `users.list` - список пользователей
- `users.get` - получение пользователя
- `users.update` - обновление пользователя
- `users.delete` - удаление пользователя

Формат ответов должен соответствовать структуре:
```json
{
  "success": bool,
  "data": { ... },
  "error": "error message (if success=false)"
}
```

## Tracing

Все NATS запросы автоматически включают OpenTelemetry trace context через библиотеку `natsw.Client`, что позволяет отслеживать запросы через всю цепочку микросервисов.
