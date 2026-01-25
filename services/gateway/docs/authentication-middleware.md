# Authentication Middleware

## Описание

Gateway использует middleware для защиты всех роутов, кроме публичных эндпоинтов (login и создание registration request). Middleware проверяет access token через auth service и извлекает информацию о пользователе.

## Архитектура авторизации

```
HTTP Request с Authorization Header
        ↓
Auth Middleware (middleware.go)
        ↓
Извлечение Bearer Token
        ↓
Извлечение Fingerprint из Cookie
        ↓
Валидация через Auth Service (NATS)
        ↓
Сохранение AuthUser в Context
        ↓
Обработчик роута
```

## Публичные роуты (без авторизации)

- `POST /api/v1/auth/login` - вход в систему
- `POST /api/v1/registration-requests` - создание заявки на регистрацию

## Защищённые роуты (требуют авторизации)

### Auth
- `POST /api/v1/auth/refresh` - обновление access token
- `POST /api/v1/auth/validate` - валидация токена
- `POST /api/v1/auth/logout` - выход с текущего устройства
- `POST /api/v1/auth/logout-all` - выход со всех устройств
- `POST /api/v1/auth/change-password` - смена пароля

### Documents
- `GET /api/v1/documents` - список документов
- `GET /api/v1/documents/:id` - получение документа
- `POST /api/v1/documents` - создание документа
- `PATCH /api/v1/documents/:id` - обновление документа
- `DELETE /api/v1/documents/:id` - удаление документа

### Users
- `GET /api/v1/users` - список пользователей
- `GET /api/v1/users/:id` - получение пользователя
- `PATCH /api/v1/users/:id` - обновление пользователя
- `DELETE /api/v1/users/:id` - удаление пользователя

### Registration Requests (управление заявками)
- `GET /api/v1/registration-requests` - список заявок
- `POST /api/v1/registration-requests/:id/approve` - одобрение заявки
- `POST /api/v1/registration-requests/:id/reject` - отклонение заявки

## Как работает авторизация

### 1. Клиент отправляет запрос

```http
GET /api/v1/users HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Cookie: fingerprint=abc123def456...
```

### 2. Middleware проверяет:

1. **Authorization Header**: Должен содержать `Bearer {token}`
2. **Fingerprint Cookie**: Должен присутствовать в HttpOnly cookie
3. **Валидация токена**: Через auth service по NATS (топик `auth.validate`)

### 3. Ответ от Auth Service

```json
{
  "success": true,
  "data": {
    "valid": true,
    "user_id": "uuid",
    "role": "admin",
    "jti": "token-id"
  }
}
```

### 4. Сохранение в Context

Middleware создает объект `AuthUser` и сохраняет его в Fiber Context:

```go
type AuthUser struct {
    ID   uuid.UUID // ID пользователя
    Role string    // Роль пользователя
    JTI  string    // JWT ID для операций
}
```

### 5. Использование в обработчиках

```go
func (s *Server) someProtectedHandler(c *fiber.Ctx) error {
    // Получение авторизованного пользователя
    authUser, ok := getAuthUser(c)
    if !ok {
        // Этого не должно происходить, если middleware работает
        return c.Status(fiber.StatusUnauthorized).JSON(...)
    }
    
    // Использование данных пользователя
    userID := authUser.ID
    userRole := authUser.Role
    
    // Бизнес-логика...
}
```

## Безопасность

### Защита от XSS
- Access token передается в `Authorization` header (не в cookie)
- Fingerprint хранится в **HttpOnly** cookie (недоступен для JavaScript)

### Защита от CSRF
- Fingerprint cookie имеет флаг **SameSite**
- Требуется наличие обоих: token в header + fingerprint в cookie

### Защита от Token Theft
- При валидации токена проверяется соответствие fingerprint
- Если fingerprint не совпадает → токен невалиден

### Device Fingerprinting
```
Token + Fingerprint = Привязка к устройству
```

Даже если токен утечет, без соответствующего fingerprint cookie злоумышленник не сможет его использовать.

## Коды ответов

### 200 OK
Запрос успешно обработан, пользователь авторизован.

### 401 Unauthorized
- Отсутствует Authorization header
- Неверный формат токена
- Отсутствует fingerprint cookie
- Токен невалиден или истек
- Fingerprint не совпадает

**Пример:**
```json
{
  "error": "unauthorized",
  "message": "Invalid or expired token"
}
```

## Примеры запросов

### Успешный запрос
```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Cookie: fingerprint=abc123..."
```

### Запрос без токена
```bash
curl -X GET http://localhost:8080/api/v1/users
# Response: 401 Unauthorized
# {"error":"unauthorized","message":"Authorization header required"}
```

### Запрос без fingerprint
```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
# Response: 401 Unauthorized
# {"error":"unauthorized","message":"Authentication fingerprint missing"}
```

## Логирование

Middleware логирует следующие события:

- **Debug**: Успешная аутентификация с user_id и role
- **Warn**: Отсутствие header/cookie, неверный формат
- **Error**: Ошибки валидации токена через auth service

## Интеграция с Auth Service

Middleware использует `authConnector.ValidateToken()` для проверки токена через NATS:

**NATS Topic:** `auth.validate`

**Request:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "fingerprint": "abc123def456..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "role": "admin",
    "jti": "unique-token-id"
  }
}
```

## Конфигурация

Настройки fingerprint cookie:

```go
const (
    CookieFingerprint  = "fingerprint"
    RefreshTokenMaxAge = 30 * 24 * 60 * 60 // 30 дней
    CookiePath         = "/"
)
```

**Production:**
- `Secure: true` (только HTTPS)
- `SameSite: Strict`
- `HTTPOnly: true`

**Development:**
- `Secure: false` (разрешает HTTP)
- `SameSite: Lax`
- `HTTPOnly: true`

## Дальнейшие улучшения

- [ ] Role-based access control (RBAC) middleware
- [ ] Rate limiting per user
- [ ] Audit logging для защищенных операций
- [ ] Refresh token rotation в middleware
