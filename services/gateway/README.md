# Gateway Service

API Gateway для микросервисной архитектуры Raibecas. Предоставляет единую точку входа для всех клиентов и маршрутизирует запросы к соответствующим микросервисам через NATS.

## Архитектура

Gateway взаимодействует с микросервисами через NATS, используя паттерн Request-Reply:

```
Client → Gateway → NATS → Document Service
                        → Auth Service
                        → Other Services
```

## API Endpoints

Все endpoints используют префикс `/api/v1`.

### Аутентификация

- `POST /api/v1/auth/login` - Вход в систему
- `POST /api/v1/auth/refresh` - Обновление токенов
- `POST /api/v1/auth/validate` - Валидация токена
- `POST /api/v1/auth/logout` - Выход из текущего устройства
- `POST /api/v1/auth/logout-all` - Выход со всех устройств
- `POST /api/v1/auth/change-password` - Изменение пароля

### Документы

- `GET /api/v1/documents` - Получение списка документов с фильтрацией и пагинацией
- `POST /api/v1/documents` - Создание нового документа
- `GET /api/v1/documents/:id` - Получение документа по ID
- `PUT /api/v1/documents/:id` - Обновление документа
- `DELETE /api/v1/documents/:id` - Удаление документа

### Пользователи

- `GET /api/v1/users` - Получение списка пользователей
- `PATCH /api/v1/users/:id` - Обновление пользователя
- `DELETE /api/v1/users/:id` - Удаление пользователя

### Запросы на регистрацию

- `GET /api/v1/registration-requests` - Получение списка запросов на регистрацию
- `POST /api/v1/registration-requests/:id/approve` - Одобрение запроса на регистрацию
- `POST /api/v1/registration-requests/:id/reject` - Отклонение запроса на регистрацию

## Настройка

### Переменные окружения

```bash
# HTTP сервер
HTTP_HOST=0.0.0.0
HTTP_PORT=8080

# NATS
NATS_URL=nats://localhost:4222
NATS_REQUEST_TIMEOUT=5s
NATS_MAX_RECONNECTS=10
NATS_RECONNECT_WAIT=2s
```

## Запуск

### Локально

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/gateway/main.go
```

### Docker

```bash
docker build -t gateway:latest .
docker run -p 8080:8080 \
  -e NATS_URL=nats://nats:4222 \
  gateway:latest
```

## Разработка

### Структура проекта

```
.
├── cmd/
│   └── gateway/         # Точка входа приложения
├── internal/
│   ├── app/            # Инициализация приложения
│   ├── config/         # Конфигурация
│   ├── connector/      # NATS коннекторы к микросервисам
│   ├── domain/         # Модели и DTO
│   └── server/         # HTTP handlers и роутинг
└── docs/               # Документация
```

### Добавление нового обработчика

1. Определите DTO в `internal/domain/dto.go`
2. Добавьте методы в интерфейс коннектора `internal/server/server.go`
3. Реализуйте методы в `internal/connector/nats_connector.go`
4. Создайте HTTP handlers в `internal/server/`
5. Зарегистрируйте роуты в `setupRoutes()`

### Валидация

Используется `go-playground/validator/v10` для валидации входящих данных. Добавьте теги валидации к полям DTO:

```go
type CreateDocumentRequest struct {
    Title string `json:"title" validate:"required,min=1,max=500"`
}
```

## Интеграция с микросервисами

Для интеграции с новым микросервисом:

1. Создайте интерфейс коннектора в `internal/server/`
2. Реализуйте NATS-коннектор в `internal/connector/`
3. Опишите топики в `docs/NATS_TOPICS.md`
4. Добавьте коннектор в структуру Server

См. `docs/NATS_TOPICS.md` для описания протокола обмена сообщениями.

## Тестирование

```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```

## Health Check

Gateway предоставляет эндпоинт для проверки здоровья:

```bash
curl http://localhost:8080/livez
curl http://localhost:8080/readyz
```

