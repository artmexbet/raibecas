# Обработчик обновления пользователей в Auth Service

## Реализация

Добавлен полный цикл синхронизации данных пользователей между `users` и `auth` сервисами при обновлении данных пользователя.

## Архитектура

```
Admin Panel → Gateway → Users Service → Outbox → NATS → Auth Service → Postgres
                            ↓
                      Update User
                            ↓
                    Create Outbox Event
                            ↓
                    Outbox Processor
                            ↓
                  Publish "users.user.updated"
                            ↓
                    Auth Consumer receives
                            ↓
                    Update local user data
```

## Изменения в Users Service

### 1. SQL запросы (users/internal/postgres/queries/users.sql)
Добавлен запрос для полного обновления пользователя:
```sql
-- name: UpdateUser :exec
UPDATE users 
SET username = $2, 
    email = $3, 
    role = $4, 
    is_active = $5, 
    updated_at = NOW() 
WHERE id = $1;
```

### 2. Repository (users/internal/postgres/user_repository.go)
Обновлен метод `UpdateUser` для создания outbox события:
- Использует транзакцию для атомарности
- Обновляет данные пользователя
- Создает событие `user.updated` в outbox
- Commit транзакции

### 3. Domain (users/internal/domain/outbox.go)
Добавлены:
- Константа `EventTypeUserUpdated = "user.updated"`
- Структура `UserUpdatedPayload` с полями:
  - `UserID` (uuid.UUID)
  - `Username` (string)
  - `Email` (string)
  - `Role` (string)
  - `IsActive` (bool)

### 4. Outbox Processor (users/internal/outbox/processor.go)
Добавлен маппинг события:
```go
case domain.EventTypeUserUpdated:
    return "users.user.updated"
```

## Изменения в Auth Service

### 1. SQL запросы (auth/internal/postgres/queries/users.sql)
Добавлен запрос для полного обновления пользователя:
```sql
-- name: UpdateUser :exec
UPDATE users 
SET username = $2, 
    email = $3, 
    role = $4, 
    is_active = $5, 
    updated_at = NOW() 
WHERE id = $1;
```

### 2. Repository (auth/internal/postgres/user.go)
Добавлен метод `UpdateUser`:
```go
func (p *Postgres) UpdateUser(ctx context.Context, userID uuid.UUID, username, email string, role domain.UserRole, isActive bool) error
```

### 3. Consumer (auth/internal/consumer/user_consumer.go)
Добавлено:
- Структура `UserUpdatedEvent` для десериализации события
- Подписка на `users.user.updated`
- Обработчик `handleUserUpdated`:
  - Парсинг user_id
  - Проверка существования пользователя
  - Обновление всех данных пользователя
  - Логирование изменений

## Формат события

### NATS Subject
```
users.user.updated
```

### Payload
```json
{
  "user_id": "uuid-string",
  "username": "string",
  "email": "string",
  "role": "User" | "Admin" | "SuperAdmin",
  "is_active": boolean
}
```

## Поток обработки

1. **Admin** редактирует пользователя в админ-панели
2. **Gateway** проксирует запрос к users service
3. **Users Service**:
   - Обновляет данные пользователя в БД
   - Создает событие в outbox таблице
4. **Outbox Processor**:
   - Читает необработанные события
   - Публикует в NATS subject `users.user.updated`
   - Отмечает событие как обработанное
5. **Auth Service**:
   - Consumer получает событие
   - Проверяет существование пользователя
   - Обновляет локальную копию данных
   - Логирует изменения

## Гарантии

### Atomicity
- Обновление пользователя и создание outbox события в одной транзакции
- Либо оба действия выполняются, либо оба откатываются

### At-Least-Once Delivery
- Outbox pattern гарантирует доставку события
- При сбое событие будет переотправлено (retry механизм)

### Idempotency
- Обновление пользователя в auth service идемпотентно
- Повторная обработка события не приведет к ошибке

## Логирование

### Users Service
```
INFO: user updated successfully
  - user_id: uuid
  - username: string
  - email: string
  - role: string
```

### Auth Service
```
INFO: received user updated event
  - user_id: uuid
  - email: string
  - username: string
  - role: string

DEBUG: updating user in auth service
  - old_username: string
  - new_username: string
  - old_email: string
  - new_email: string
  - old_role: string
  - new_role: string

INFO: user updated successfully
  - user_id: uuid
  - username: string
  - email: string
  - role: string
  - is_active: boolean
```

## Обработка ошибок

### Users Service
- Ошибка обновления → транзакция откатывается
- Ошибка создания outbox → транзакция откатывается
- HTTP 500 возвращается клиенту

### Auth Service
- Пользователь не найден → логируется ошибка, событие не обрабатывается
- Ошибка обновления → логируется ошибка, событие помечается для retry
- Невалидный user_id → логируется ошибка, событие отклоняется

## Тестирование

### Сценарий тестирования:
1. Создать пользователя через админ-панель
2. Изменить данные пользователя (username, email, role)
3. Проверить в users БД - данные обновлены
4. Проверить в auth БД - данные синхронизированы
5. Проверить логи outbox processor - событие опубликовано
6. Проверить логи auth consumer - событие получено и обработано

### Проверка данных:
```sql
-- Users DB
SELECT id, username, email, role, is_active, updated_at 
FROM users WHERE id = 'user-id';

-- Auth DB
SELECT id, username, email, role, is_active, updated_at 
FROM users WHERE id = 'user-id';
```

## Совместимость

- ✅ Users Service: версия с outbox pattern
- ✅ Auth Service: consumer для user.registered и user.updated
- ✅ NATS: subject `users.user.updated`
- ✅ Роли: User, Admin, SuperAdmin (PascalCase)

## Файлы изменений

### Users Service (4 файла)
- `internal/domain/outbox.go` - добавлена константа и payload
- `internal/postgres/queries/users.sql` - SQL запрос
- `internal/postgres/user_repository.go` - outbox событие
- `internal/outbox/processor.go` - маппинг subject

### Auth Service (3 файла)
- `internal/postgres/queries/users.sql` - SQL запрос
- `internal/postgres/user.go` - метод UpdateUser
- `internal/consumer/user_consumer.go` - обработчик события

## Сборка

✅ Users Service собран успешно
✅ Auth Service собран успешно
