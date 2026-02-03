# Улучшения Outbox Pattern - Резюме

## ✅ Реализовано

### 1. Выбор ролей при регистрации (User, Admin, SuperAdmin)

**Users Service:**
- ✅ Обновлена миграция БД - добавлена enum типа для role
- ✅ Добавлены константы ролей: `RoleUser`, `RoleAdmin`, `RoleSuperAdmin`
- ✅ Функция валидации `IsValidRole(role string) bool`
- ✅ `ApproveRegistrationRequest` теперь принимает параметр `role`
- ✅ DTO `ApproveRegistrationRequest` содержит поле `Role`
- ✅ Роль передается в outbox событие `user.registered`

**Auth Service:**
- ✅ Обновлена БД миграция с enum ролями
- ✅ Добавлена константа `RoleSuperAdmin`
- ✅ `ApproveRegistration` теперь принимает параметр `role`
- ✅ Consumer получает роль из события и применяет её
- ✅ Интерфейс `IRegistrationService` обновлен

### 2. Редактирование пользователей с фронтенда

**Users Service:**
- ✅ Обновлен `UpdateUserParams` - добавлено поле `Role`
- ✅ SQL запрос `UpdateUser` теперь обновляет role
- ✅ `HandleUpdateUser` валидирует роль перед обновлением
- ✅ Сервис валидирует роль в `UpdateUser`
- ✅ DTO `UpdateUserPayload` содержит поле `Role`

### 3. События об изменении статуса пользователя

**Обе услуги:**
- ✅ Добавлен event type: `user.status_changed`
- ✅ Создана payload модель `UserStatusChangedPayload`
- ✅ Методы для создания outbox событий при изменении статуса

**Auth Service:**
- ✅ `UpdateUserRoleWithOutbox` - изменение роли + event
- ✅ `UpdateUserIsActiveWithOutbox` - изменение статуса + event

### 4. Транзакционный Outbox с блокировкой строк

**Users Service:**
- ✅ Миграция добавила поле `processing_started_at`
- ✅ Добавлены индексы для эффективного поиска:
  - `idx_outbox_unprocessed` - необработанные события
  - `idx_outbox_stale_locks` - "зависшие" события
- ✅ `GetUnprocessedEventsTx` - SELECT FOR UPDATE с локировкой
- ✅ `CleanupStaleLocks` - очистка зависших блокировок (timeout 30 сек)
- ✅ Processor работает с транзакциями - атомарная обработка
- ✅ Изолированость от других инстансов через `FOR UPDATE SKIP LOCKED`

## 📋 API Эндпоинты

### Одобрение регистрации с выбором роли
```json
POST /users/approve-registration
{
  "requestId": "uuid",
  "approverId": "uuid",
  "role": "admin"  // "user", "admin", "super_admin"
}
```

### Редактирование пользователя
```json
PATCH /users/{id}
{
  "updates": {
    "role": "admin",      // опционально
    "isActive": true,     // опционально
    "email": "new@email.com",  // опционально
    "fullName": "New Name" // опционально
  }
}
```

## 🔐 Безопасность

- ✅ **Трансакционность**: роль + outbox событие в одной транзакции
- ✅ **Идемпотентность**: SELECT FOR UPDATE предотвращает дублирование
- ✅ **Распределенность**: SKIP LOCKED позволяет запускать несколько процессоров
- ✅ **Таймауты**: Cleanup Job очищает "зависшие" события (30 сек)
- ✅ **Retry механизм**: максимум 5 попыток перед DLQ

## 📊 NATS Subjects

```
users.user.registered        - при создании пользователя (с ролью)
users.user.status_changed    - при изменении роли/статуса
auth.user.registered         - от auth при успешном создании
```

## 🗂️ Структура Событий

### user.registered (из users)
```json
{
  "user_id": "uuid",
  "username": "string",
  "email": "string",
  "password_hash": "string",
  "role": "admin",           // новое!
  "is_active": true
}
```

### user.status_changed (из обоих сервисов)
```json
{
  "user_id": "uuid",
  "role": "admin",           // опционально
  "is_active": true          // опционально
}
```

## 🚀 Миграции

### Users Service
- `000004_add_role_enum.up.sql` - enum тип для role
- `000004_add_role_enum.down.sql` - откат

### Auth Service
- `004_add_role_enum.sql` - enum тип для role

## 🔄 Флоу обновления ролей

1. Frontend отправляет PATCH /users/{id} с новой ролью
2. Users сервис валидирует роль
3. UpdateUser создает outbox событие `user.status_changed`
4. Outbox Processor (5 сек интервал) отправляет событие в NATS
5. Auth Consumer получает событие и обновляет роль пользователя
6. Auth создает свой outbox event (синхронизация обратно)

## ✨ Ключевые улучшения

| Функция | До | После |
|---------|-------|-------|
| Выбор роли | Hardcoded 'user' | Выбор из 3 вариантов |
| Редактирование | ❌ | ✅ PATCH endpoint |
| Синхронизация статуса | ❌ | ✅ Двусторонняя |
| Распределенность | ❌ | ✅ FOR UPDATE SKIP LOCKED |
| Таймауты блокировок | ❌ | ✅ 30 сек cleanup |
| Отслеживание обработки | Processing flag | `processing_started_at` |

## 🧪 Тестирование

```bash
# 1. Создать регистрацию
POST /registrations
{
  "email": "newadmin@example.com",
  "username": "admin_user",
  "password": "SecurePass123"
}

# 2. Одобрить с ролью admin
PATCH /registrations/{id}/approve
{
  "approverId": "current_admin_uuid",
  "role": "admin"
}

# 3. Отредактировать роль пользователя
PATCH /users/{user_id}
{
  "updates": {
    "role": "super_admin"
  }
}

# 4. Проверить синхронизацию в обеих БД
SELECT * FROM users WHERE id = ?;
```

## 📝 Комментарии к коду

- `ForUpdate SKIP LOCKED` - позволяет нескольким процессорам работать параллельно
- `processing_started_at` - отмечает начало обработки, нужна для cleanup
- `CleanupStaleLocks` - запускается каждые 10 секунд, очищает блокировки старше 30 сек
- `Serializable isolation` - максимальная консистентность при обработке batch'а

## ⚠️ Известные ограничения

- Роли жестко закодированы в enum БД (нужна миграция для добавления новых)
- CleanupStaleLocks запускается в том же контексте процессора
- Нет метрик для отслеживания длительности обработки событий

---

**Готово к production!** ✅
