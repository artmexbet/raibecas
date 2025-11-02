# Сервис Аутентификации

Микросервис аутентификации для платформы Raibecas. Обрабатывает регистрацию пользователей, вход, выход и управление токенами с использованием JWT, PostgreSQL и Redis.

## Возможности

- **Регистрация с модерацией**: Пользователи отправляют заявки на регистрацию, требующие одобрения администратора
- **JWT аутентификация**: Безопасная токен-based аутентификация с access и refresh токенами
- **Управление сессиями**: Хранение refresh-токенов в Redis
- **Event-Driven архитектура**: NATS Pub/Sub для коммуникации с другими сервисами
- **Безопасность паролей**: Хеширование паролей bcrypt с настраиваемым уровнем сложности
- **Современные Go паттерны**: Чистая архитектура с внедрением зависимостей

## Архитектура

Сервис следует принципам чистой архитектуры:

```
auth/
├── cmd/
│   └── auth/           # Точка входа приложения
├── internal/
│   ├── config/         # Управление конфигурацией
│   ├── domain/         # Доменные модели и ошибки
│   ├── repository/     # Слой доступа к данным (PostgreSQL)
│   ├── storeredis/     # Хранилище токенов в Redis
│   ├── service/        # Бизнес-логика
│   ├── handler/        # NATS обработчики
│   ├── nats/          # NATS событийная pub/sub
│   └── server/        # Настройка сервера
├── pkg/
│   └── jwt/           # Управление JWT токенами
└── migrations/        # Миграции базы данных
```

## NATS топики

Сервис работает через NATS, подписываясь на следующие топики:

### Входящие запросы (Request/Reply)

- `auth.register` - Создать заявку на регистрацию
- `auth.login` - Аутентифицировать пользователя
- `auth.refresh` - Обновить токены
- `auth.validate` - Валидировать access token
- `auth.logout` - Выход с текущего устройства
- `auth.logout_all` - Выход со всех устройств
- `auth.change_password` - Изменить пароль

### Публикуемые события

- `auth.user.registered` - Когда новый пользователь создан (после одобрения)
- `auth.user.login` - Когда пользователь входит
- `auth.user.logout` - Когда пользователь выходит
- `auth.password.reset` - Когда пароль изменён
- `auth.registration.requested` - Когда создана заявка на регистрацию

### Подписки на события

- `admin.registration.approved` - Администратор одобрил заявку
- `admin.registration.rejected` - Администратор отклонил заявку

## Формат сообщений

### Регистрация (auth.register)

**Запрос:**
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePassword123",
  "metadata": {
    "reason": "Для исследований"
  }
}
```

**Ответ:**
```json
{
  "request_id": "uuid",
  "status": "pending",
  "message": "Заявка на регистрацию отправлена. Ожидается одобрение администратора."
}
```

### Вход (auth.login)

**Запрос:**
```json
{
  "email": "john@example.com",
  "password": "SecurePassword123",
  "device_id": "device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**Ответ:**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "uuid",
  "expires_in": 900
}
```

### Обновление токенов (auth.refresh)

**Запрос:**
```json
{
  "refresh_token": "uuid",
  "device_id": "device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**Ответ:**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "new-uuid",
  "expires_in": 900
}
```

### Валидация токена (auth.validate)

**Запрос:**
```json
{
  "token": "eyJhbGc..."
}
```

**Ответ:**
```json
{
  "valid": true,
  "user_id": "uuid",
  "role": "user"
}
```

### Выход (auth.logout)

**Запрос:**
```json
{
  "user_id": "uuid",
  "token": "eyJhbGc..."
}
```

**Ответ:**
```json
{
  "message": "Выход выполнен успешно"
}
```

### Изменение пароля (auth.change_password)

**Запрос:**
```json
{
  "user_id": "uuid",
  "token": "eyJhbGc...",
  "old_password": "OldPassword123",
  "new_password": "NewPassword456"
}
```

**Ответ:**
```json
{
  "message": "Пароль успешно изменён"
}
```

## Конфигурация

Конфигурация загружается из переменных окружения с использованием cleanenv:

### Конфигурация сервера
- `SERVER_PORT` - Порт сервера (по умолчанию: 8081)
- `SERVER_READ_TIMEOUT` - Timeout чтения (по умолчанию: 10s)
- `SERVER_WRITE_TIMEOUT` - Timeout записи (по умолчанию: 10s)
- `SERVER_SHUTDOWN_TIMEOUT` - Timeout остановки (по умолчанию: 5s)

### Конфигурация базы данных
- `DB_HOST` - Хост PostgreSQL (по умолчанию: localhost)
- `DB_PORT` - Порт PostgreSQL (по умолчанию: 5432)
- `DB_USER` - Пользователь PostgreSQL (по умолчанию: raibecas)
- `DB_PASSWORD` - Пароль PostgreSQL (обязательно)
- `DB_NAME` - Имя базы данных (по умолчанию: raibecas)
- `DB_SSL_MODE` - Режим SSL (по умолчанию: disable)
- `DB_MAX_CONNS` - Максимум соединений (по умолчанию: 25)
- `DB_MIN_CONNS` - Минимум соединений (по умолчанию: 5)

### Конфигурация Redis
- `REDIS_HOST` - Хост Redis (по умолчанию: localhost)
- `REDIS_PORT` - Порт Redis (по умолчанию: 6379)
- `REDIS_PASSWORD` - Пароль Redis (опционально)
- `REDIS_DB` - Номер БД Redis (по умолчанию: 0)

### Конфигурация NATS
- `NATS_URL` - URL NATS сервера (по умолчанию: nats://localhost:4222)
- `NATS_MAX_RECONNECTS` - Максимум попыток переподключения (по умолчанию: 10)
- `NATS_RECONNECT_WAIT` - Время ожидания переподключения (по умолчанию: 2s)

### Конфигурация JWT
- `JWT_SECRET` - Секретный ключ JWT (обязательно)
- `JWT_ACCESS_TTL` - TTL access токена (по умолчанию: 15m)
- `JWT_REFRESH_TTL` - TTL refresh токена (по умолчанию: 168h / 7 дней)
- `JWT_ISSUER` - Издатель токена (по умолчанию: raibecas-auth)

## Разработка

### Требования
- Go 1.25.1 или выше
- Docker и Docker Compose
- PostgreSQL 16 с расширением pgvector
- Redis 7
- NATS Server
- SQLC для генерации кода

### Установка SQLC

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### Генерация кода с SQLC

```bash
cd services/auth
sqlc generate
```

### Настройка

1. Запустите зависимости:
```bash
docker-compose -f docker-compose.dev.yml up -d postgres redis nats
```

2. Выполните миграции:
```bash
psql -h localhost -U raibecas -d raibecas -f migrations/001_create_users_table.sql
psql -h localhost -U raibecas -d raibecas -f migrations/002_create_registration_requests_table.sql
```

3. Установите переменные окружения:
```bash
export DB_PASSWORD=raibecas_dev
export JWT_SECRET=dev_secret_change_in_production
```

4. Запустите сервис:
```bash
go run cmd/auth/main.go
```

### Запуск с Docker Compose

```bash
docker-compose -f docker-compose.dev.yml up --build
```

## Тестирование

Запустите unit-тесты:
```bash
go test ./... -v
```

Запустите интеграционные тесты (требуется testcontainers):
```bash
go test ./... -v -tags=integration
```

## Безопасность

- Пароли хешируются с использованием bcrypt с cost 12
- JWT токены подписываются с использованием HS256
- Access токены истекают через 15 минут
- Refresh токены истекают через 7 дней
- Все защищённые эндпоинты требуют аутентификации
- Изменение пароля автоматически выполняет выход со всех устройств

## Схема базы данных

### Таблица users
- `id` - UUID первичный ключ
- `username` - Уникальное имя пользователя
- `email` - Уникальный email
- `password_hash` - Хеш пароля bcrypt
- `role` - Роль пользователя (user/admin)
- `is_active` - Статус активности аккаунта
- `created_at` - Метка времени создания
- `updated_at` - Метка времени обновления

### Таблица registration_requests
- `id` - UUID первичный ключ
- `username` - Запрошенное имя пользователя
- `email` - Запрошенный email
- `password` - Хешированный пароль
- `status` - Статус заявки (pending/approved/rejected)
- `metadata` - Дополнительные JSON метаданные
- `created_at` - Метка времени создания
- `updated_at` - Метка времени обновления
- `approved_by` - UUID утверждающего (внешний ключ к users)
- `approved_at` - Метка времени одобрения

## Лицензия

MIT
