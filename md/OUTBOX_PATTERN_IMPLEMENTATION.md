# Outbox Pattern Implementation

## Обзор

Реализован **Outbox Pattern** для надежной синхронизации данных между сервисами `users` и `auth` при регистрации новых пользователей.

## Архитектура

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Gateway   │────────▶│    Users     │────────▶│    Auth     │
│             │         │              │         │             │
└─────────────┘         └──────────────┘         └─────────────┘
                              │                        │
                              │ 1. Create User         │
                              │ 2. Create Outbox       │
                              │    Event               │
                              │    (transactional)     │
                              │                        │
                              ▼                        │
                        ┌──────────┐                   │
                        │ Outbox   │                   │
                        │ Table    │                   │
                        └──────────┘                   │
                              │                        │
                              │ 3. Poll Events         │
                              ▼                        │
                        ┌──────────┐                   │
                        │ Outbox   │                   │
                        │Processor │───────NATS───────▶│
                        └──────────┘                   │
                                                       │ 4. Create User
                                                       │    (idempotent)
                                                       ▼
```

## Компоненты

### 1. Users Service

#### Outbox Table
```sql
CREATE TABLE outbox (
    id UUID PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100),
    event_type VARCHAR(100),
    payload JSONB,
    created_at TIMESTAMP,
    processed_at TIMESTAMP,
    retry_count INT DEFAULT 0,
    last_error TEXT
);
```

#### Outbox Repository
- `CreateOutboxEvent` - создание события в транзакции
- `GetUnprocessedEvents` - получение необработанных событий
- `MarkEventAsProcessed` - отметка события как обработанного
- `MarkEventAsFailed` - обновление счетчика попыток

#### Outbox Processor
- Периодически (каждые 5 секунд) проверяет необработанные события
- Публикует события в NATS
- Отмечает успешно обработанные события
- Retry механизм с максимум 5 попытками

#### Модификация ApproveRegistrationRequest
При одобрении запроса на регистрацию:
1. Создается пользователь в БД
2. Создается outbox событие `user.registered` **в той же транзакции**
3. Транзакция коммитится

### 2. Auth Service

#### User Consumer
- Подписывается на события `users.user.registered`
- Создает пользователя с предоставленным UUID
- **Идемпотентность**: проверяет существование пользователя перед созданием

#### Модификация CreateUser
- Поддержка создания пользователя с заданным ID
- Проверка на существование (для идемпотентности)
- Использует `CreateUserWithID` SQL метод с `ON CONFLICT DO NOTHING`

## Гарантии

### 1. At-Least-Once Delivery
- События сохраняются в БД перед публикацией
- Retry механизм при сбоях публикации
- События не теряются даже при падении сервиса

### 2. Idempotency
- Auth проверяет существование пользователя перед созданием
- Повторная обработка события не создает дублей

### 3. Transactional Consistency
- Создание пользователя и outbox события в одной транзакции
- Либо оба операции успешны, либо обе откатываются

## NATS Subject

```
users.user.registered
```

## Event Payload

```json
{
  "user_id": "uuid",
  "username": "string",
  "email": "string",
  "password_hash": "string",
  "role": "user",
  "is_active": true
}
```

## Настройки

### Outbox Processor
- **Poll Interval**: 5 секунд
- **Batch Size**: 10 событий
- **Max Retry Count**: 5 попыток
- **Retry Strategy**: Exponential backoff (можно добавить)

## Миграции

### Users Service
- `000003_create_outbox_table.up.sql` - создание outbox таблицы
- `000003_create_outbox_table.down.sql` - откат миграции

### Auth Service
- Модификация `users.sql` - добавление `CreateUserWithID` метода

## Мониторинг

### Метрики (рекомендуется добавить)
- `outbox_events_total` - общее количество событий
- `outbox_events_processed` - обработанные события
- `outbox_events_failed` - неудачные попытки
- `outbox_processing_duration` - время обработки

### Логи
- Логирование всех событий в outbox processor
- Логирование создания пользователей в auth consumer
- Ошибки публикации и обработки

## Запуск

### 1. Применить миграции
```bash
# Users service
migrate -path services/users/migrations -database "postgresql://..." up

# Auth service (если изменения в схеме)
migrate -path services/auth/migrations -database "postgresql://..." up
```

### 2. Перегенерировать sqlc (для auth)
```bash
cd services/auth/internal/postgres
sqlc generate
```

### 3. Запустить сервисы
```bash
# Users service
cd services/users
go run cmd/users/main.go

# Auth service
cd services/auth
go run cmd/auth/main.go
```

## Тестирование

### Сценарий
1. Создать registration request через Gateway
2. Одобрить запрос (Admin panel)
3. Проверить создание пользователя в users DB
4. Проверить создание outbox события
5. Дождаться обработки события (max 5 сек)
6. Проверить создание пользователя в auth DB
7. Попробовать войти с новыми credentials

### Проверка idempotency
1. Остановить auth сервис
2. Одобрить несколько registration requests
3. Запустить auth сервис
4. Проверить, что пользователи не дублируются

## Улучшения (Future)

- [ ] Exponential backoff для retry
- [ ] Dead letter queue для проблемных событий
- [ ] Метрики Prometheus
- [ ] Graceful shutdown для outbox processor
- [ ] Конфигурация через environment variables
- [ ] Health check endpoint с информацией об outbox
- [ ] Cleanup старых обработанных событий
