# План миграции на NATS архитектуру

## Текущее состояние

✅ Документация переведена на русский
✅ Cleanenv интегрирован для конфигурации
✅ SQLC настроен для генерации кода БД

## Что нужно сделать

### 1. Генерация SQLC кода

```bash
cd services/auth
sqlc generate
```

Это создаст файлы в `sqlc/generated/` с типобезопасными функциями для работы с БД.

### 2. Замена репозиториев на SQLC

- Заменить `internal/repository/user_repository.go` на использование SQLC кода
- Заменить `internal/repository/registration_repository.go` на использование SQLC кода
- Обновить интерфейсы в `internal/domain/repository.go` при необходимости

### 3. Создание NATS handlers вместо HTTP

Создать новую директорию `internal/handler/nats/` с файлами:

#### `auth_handler.go`
```go
type AuthHandler struct {
    authService *service.AuthService
    publisher   *nats.Publisher
}

func (h *AuthHandler) HandleLogin(msg *nats.Msg) {
    // Десериализовать LoginRequest из msg.Data
    // Вызвать authService.Login()
    // Сериализовать ответ
    // msg.Respond(response)
}

func (h *AuthHandler) HandleRegister(msg *nats.Msg) {
    // Аналогично
}

func (h *AuthHandler) HandleRefresh(msg *nats.Msg) {
    // Аналогично
}

func (h *AuthHandler) HandleValidate(msg *nats.Msg) {
    // Аналогично
}

func (h *AuthHandler) HandleLogout(msg *nats.Msg) {
    // Аналогично
}

func (h *AuthHandler) HandleLogoutAll(msg *nats.Msg) {
    // Аналогично
}

func (h *AuthHandler) HandleChangePassword(msg *nats.Msg) {
    // Аналогично
}
```

#### `registration_handler.go`
```go
type RegistrationHandler struct {
    regService *service.RegistrationService
    publisher  *nats.Publisher
}

func (h *RegistrationHandler) HandleRegister(msg *nats.Msg) {
    // Обработка регистрации
}
```

### 4. Обновление server.go

Заменить:
```go
// Старое (HTTP с Fiber)
app := fiber.New()
api := app.Group("/api/v1")
api.Post("/login", authHandler.Login)
```

На:
```go
// Новое (NATS Request/Reply)
nc, _ := nats.Connect(cfg.NATS.URL)

// Подписываемся на топики
nc.Subscribe("auth.login", authHandler.HandleLogin)
nc.Subscribe("auth.register", regHandler.HandleRegister)
nc.Subscribe("auth.refresh", authHandler.HandleRefresh)
nc.Subscribe("auth.validate", authHandler.HandleValidate)
nc.Subscribe("auth.logout", authHandler.HandleLogout)
nc.Subscribe("auth.logout_all", authHandler.HandleLogoutAll)
nc.Subscribe("auth.change_password", authHandler.HandleChangePassword)

// Существующие event subscriptions остаются
nc.Subscribe("admin.registration.approved", subscriber.handleRegistrationApproved)
nc.Subscribe("admin.registration.rejected", subscriber.handleRegistrationRejected)
```

### 5. Удалить HTTP зависимости

- Удалить Fiber из `go.mod`
- Удалить `internal/handler/auth_handler.go` (старый HTTP handler)
- Удалить `internal/handler/registration_handler.go` (старый HTTP handler)
- Удалить `internal/middleware/` (больше не нужен)
- Удалить `test_api.sh` (больше не релевантен)

### 6. Обновить тесты

- Заменить HTTP тесты на NATS тесты
- Использовать NATS тестовый сервер
- Тестировать Request/Reply паттерн

### 7. Обновить Docker и docker-compose

Убрать порт 8081 (HTTP больше не используется):
```yaml
auth-service:
  # Убрать ports:
  # - "8081:8081"
  environment:
    # Оставить только NATS_URL
    NATS_URL: nats://nats:4222
```

### 8. Обновить README.md

Обновить примеры использования с HTTP на NATS:

```bash
# Старое
curl -X POST http://localhost:8081/api/v1/login -d '{"email":"...","password":"..."}'

# Новое (используя nats CLI)
nats request auth.login '{"email":"...","password":"..."}'
```

## Порядок выполнения

1. Сгенерировать SQLC код
2. Обновить репозитории для использования SQLC
3. Создать NATS handlers
4. Обновить server.go
5. Удалить старый HTTP код
6. Обновить тесты
7. Обновить документацию
8. Тестировать всё вместе

## Проверка работы

После завершения, сервис должен:
- ✅ Подключаться к NATS
- ✅ Отвечать на Request/Reply топики
- ✅ Публиковать события
- ✅ Подписываться на события от админа
- ✅ Использовать SQLC для БД операций
- ✅ Использовать cleanenv для конфигурации
- ❌ НЕ иметь HTTP сервера

## Время выполнения

Оценка: 4-6 часов работы для полной миграции и тестирования.
