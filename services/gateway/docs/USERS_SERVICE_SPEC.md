# Users Service - Спецификация NATS топиков

## Описание

Users Service отвечает за управление пользователями и заявками на регистрацию. Сервис подписывается на NATS топики и обрабатывает запросы от Gateway.

## NATS Topics

### 1. users.list
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

**Обработчик должен:**
- Парсить параметры запроса (page, page_size, search, is_active)
- Применять фильтры в SQL запросе
- Реализовать пагинацию с LIMIT и OFFSET
- Считать общее количество записей (total_count)
- Возвращать массив пользователей

---

### 2. users.get
**Описание:** Получение информации о конкретном пользователе по ID

**Request:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
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

**Обработчик должен:**
- Валидировать UUID
- Искать пользователя в БД по ID
- Вернуть 404 если пользователь не найден (success=true, но с ошибкой в error)
- Исключить чувствительные данные (password_hash)

---

### 3. users.update
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

**Обработчик должен:**
- Валидировать входные данные (email format, username length, etc.)
- Проверить существование пользователя
- Обновить только переданные поля (partial update)
- Обновить поле `updated_at`
- Вернуть обновленного пользователя

---

### 4. users.delete
**Описание:** Удаление пользователя (мягкое удаление - soft delete)

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

**Обработчик должен:**
- Валидировать UUID
- Проверить существование пользователя
- Выполнить soft delete (is_active = false) или hard delete
- Опционально: удалить связанные данные (сессии, токены)

---

## Registration Requests Topics

### 5. users.registration.create
**Описание:** Создание заявки на регистрацию

**Request:**
```json
{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "SecurePassword123",
  "metadata": {
    "reason": "I want to join",
    "organization": "ACME Corp"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "request_id": "uuid",
    "status": "pending",
    "message": "Registration request submitted successfully. Waiting for admin approval."
  }
}
```

**Обработчик должен:**
- Валидировать email (формат, уникальность)
- Валидировать username (длина, символы, уникальность)
- Валидировать пароль (минимальная длина, сложность)
- Хэшировать пароль с использованием bcrypt
- Сохранить заявку в БД со статусом "pending"
- Опционально: отправить событие в NATS (registration.requested)

**Таблица БД:**
```sql
CREATE TABLE registration_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMP
);
```

---

### 6. users.registration.list
**Описание:** Получение списка заявок на регистрацию

**Request:**
```json
{
  "page": 1,
  "page_size": 10,
  "status": "pending"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "requests": [
      {
        "id": "uuid",
        "username": "newuser",
        "email": "newuser@example.com",
        "status": "pending",
        "metadata": {...},
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z",
        "approved_by": null,
        "approved_at": null
      }
    ],
    "total_count": 5,
    "page": 1,
    "page_size": 10
  }
}
```

**Обработчик должен:**
- Применять фильтр по статусу (pending, approved, rejected)
- Реализовать пагинацию
- Сортировать по дате создания (новые сверху)
- Исключить password_hash из ответа

---

### 7. users.registration.approve
**Описание:** Одобрение заявки на регистрацию и создание пользователя

**Request:**
```json
{
  "request_id": "uuid",
  "approver_id": "uuid"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Registration approved successfully. User account created.",
    "user": {
      "id": "uuid",
      "email": "newuser@example.com",
      "username": "newuser",
      "full_name": "New User",
      "registered_at": "2024-01-01T00:00:00Z",
      "last_login_at": "2024-01-01T00:00:00Z",
      "is_active": true
    }
  }
}
```

**Обработчик должен:**
1. Получить заявку из БД по request_id
2. Проверить, что статус = "pending" (нельзя одобрить уже обработанную)
3. Создать пользователя в таблице users:
   - username, email из заявки
   - password_hash из заявки (уже хэширован)
   - role = "user" (по умолчанию)
   - is_active = true
4. Обновить заявку:
   - status = "approved"
   - approved_by = approver_id
   - approved_at = NOW()
5. Опционально: отправить событие user.registered
6. Опционально: отправить email пользователю
7. Вернуть созданного пользователя

**SQL транзакция:**
```sql
BEGIN;
-- Создать пользователя
INSERT INTO users (username, email, password_hash, role, is_active)
SELECT username, email, password, 'user', true
FROM registration_requests
WHERE id = $1 AND status = 'pending';

-- Обновить заявку
UPDATE registration_requests
SET status = 'approved', approved_by = $2, approved_at = NOW()
WHERE id = $1;
COMMIT;
```

---

### 8. users.registration.reject
**Описание:** Отклонение заявки на регистрацию

**Request:**
```json
{
  "request_id": "uuid",
  "approver_id": "uuid",
  "reason": "Invalid email domain"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "Registration request rejected."
  }
}
```

**Обработчик должен:**
1. Получить заявку из БД по request_id
2. Проверить, что статус = "pending"
3. Обновить заявку:
   - status = "rejected"
   - approved_by = approver_id (кто отклонил)
   - approved_at = NOW()
   - metadata.rejection_reason = reason
4. Опционально: отправить событие registration.rejected
5. Опционально: отправить email пользователю с причиной отклонения

---

## Общий формат ошибок

Все обработчики должны возвращать ошибки в формате:

```json
{
  "success": false,
  "error": "error_code"
}
```

**Примеры кодов ошибок:**
- `invalid_request` - неверный формат запроса
- `validation_failed` - провалена валидация
- `not_found` - ресурс не найден
- `already_exists` - дублирование (email, username)
- `internal_error` - внутренняя ошибка сервера
- `invalid_status` - неверный статус (например, заявка уже обработана)

---

## Примеры реализации (Go)

### Handler структура
```go
type UserHandler struct {
    userService UserService
    publisher   EventPublisher
}

func (h *UserHandler) HandleListUsers(msg *natsw.Message) error {
    var req ListUsersRequest
    if err := msg.UnmarshalData(&req); err != nil {
        return h.respondError(msg, "invalid_request")
    }
    
    users, total, err := h.userService.ListUsers(msg.Ctx, req)
    if err != nil {
        return h.respondError(msg, "internal_error")
    }
    
    response := ListUsersResponse{
        Users:      users,
        TotalCount: total,
        Page:       req.Page,
        PageSize:   req.PageSize,
    }
    
    return h.respond(msg, response)
}
```

### Регистрация подписок
```go
func (s *Server) setupSubscriptions() error {
    // Users
    s.nc.Subscribe("users.list", s.userHandler.HandleListUsers)
    s.nc.Subscribe("users.get", s.userHandler.HandleGetUser)
    s.nc.Subscribe("users.update", s.userHandler.HandleUpdateUser)
    s.nc.Subscribe("users.delete", s.userHandler.HandleDeleteUser)
    
    // Registration requests
    s.nc.Subscribe("users.registration.create", s.regHandler.HandleCreate)
    s.nc.Subscribe("users.registration.list", s.regHandler.HandleList)
    s.nc.Subscribe("users.registration.approve", s.regHandler.HandleApprove)
    s.nc.Subscribe("users.registration.reject", s.regHandler.HandleReject)
    
    return nil
}
```

---

## События (опционально)

Users Service может публиковать события для других сервисов:

- `users.user.created` - новый пользователь создан
- `users.user.updated` - пользователь обновлен
- `users.user.deleted` - пользователь удален
- `users.registration.requested` - новая заявка на регистрацию
- `users.registration.approved` - заявка одобрена
- `users.registration.rejected` - заявка отклонена

---

## Безопасность

1. **Валидация входных данных** - всегда валидировать перед обработкой
2. **Хэширование паролей** - использовать bcrypt с cost >= 12
3. **SQL Injection** - использовать prepared statements
4. **Уникальность** - проверять email и username на уникальность
5. **Лимиты** - ограничивать page_size (max 100)
6. **Логирование** - логировать важные операции (approve, reject, delete)

---

## Метрики

Рекомендуется отслеживать:
- Количество заявок по статусам (pending/approved/rejected)
- Время обработки запросов
- Количество ошибок валидации
- Количество созданных пользователей
