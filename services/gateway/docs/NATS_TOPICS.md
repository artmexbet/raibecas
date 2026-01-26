# NATS Topics для Gateway Service

Документация по NATS топикам, используемым Gateway для взаимодействия с микросервисами.

## Общая информация

- **Протокол**: Request-Reply pattern
- **Формат сообщений**: JSON
- **Timeout**: 10 секунд

---

## Auth Service Topics

### `auth.register`
Создание заявки на регистрацию (требует одобрения администратора)

**Request:**
```json
{
  "username": "john_doe",
  "email": "user@example.com",
  "password": "password123",
  "metadata": {
    "organization": "University",
    "purpose": "Research"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "request_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "pending",
    "message": "Registration request submitted successfully. Waiting for admin approval."
  }
}
```

### `auth.login`
Аутентификация пользователя

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "device_id": "optional-device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_id": "550e8400-e29b-41d4-a716-446655440000",
    "fingerprint": "abc123def456...",
    "expires_in": 900,
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "username": "john_doe",
      "email": "user@example.com",
      "role": "user",
      "created_at": "2026-01-15T10:00:00Z"
    }
  }
}
```

### `auth.refresh`
Обновление access токена

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_id": "550e8400-e29b-41d4-a716-446655440000",
  "fingerprint": "abc123def456...",
  "device_id": "optional-device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_id": "550e8400-e29b-41d4-a716-446655440000",
    "fingerprint": "abc123def456...",
    "expires_in": 900,
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "username": "john_doe",
      "email": "user@example.com",
      "role": "user",
      "created_at": "2026-01-15T10:00:00Z"
    }
  }
}
```

### `auth.validate`
Валидация токена (fingerprint обязателен для безопасности)

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
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "role": "user",
    "jti": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

**Response (Invalid Token):**
```json
{
  "success": true,
  "data": {
    "valid": false
  }
}
```

### `auth.logout`
Выход из текущего устройства

**Request:**
```json
{
  "token_id": "550e8400-e29b-41d4-a716-446655440000",
  "access_token_jti": "jwt-id-from-access-token",
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

### `auth.logout_all`
Выход со всех устройств

**Request:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Logged out from all devices successfully"
  }
}
```

### `auth.change_password`
Изменение пароля

**Request:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "old_password": "oldpassword123",
  "new_password": "newpassword456"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Password changed successfully"
  }
}
```

---

## Auth Service Events (Pub/Sub)

Эти события публикуются Auth Service и могут быть прослушаны другими сервисами.

### `auth.registration.requested`
Событие создания заявки на регистрацию

**Payload:**
```json
{
  "request_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "john_doe",
  "email": "user@example.com",
  "timestamp": "2026-01-15T10:00:00Z"
}
```

### `auth.user.registered`
Событие успешной регистрации пользователя (после одобрения админа)

**Payload:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "john_doe",
  "email": "user@example.com",
  "role": "user",
  "timestamp": "2026-01-15T10:00:00Z"
}
```

### `auth.user.login`
Событие входа пользователя

**Payload:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "username": "john_doe",
  "email": "user@example.com",
  "role": "user",
  "device_id": "optional-device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1",
  "timestamp": "2026-01-15T10:00:00Z"
}
```

### `auth.user.logout`
Событие выхода пользователя

**Payload:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "device_id": "optional-device-uuid",
  "reason": "user_initiated",
  "timestamp": "2026-01-15T10:00:00Z"
}
```

### `auth.password.reset`
Событие изменения пароля

**Payload:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "method": "self-service",
  "timestamp": "2026-01-15T10:00:00Z"
}
```

---


## Обработка ошибок

При ошибке возвращается:

```json
{
  "success": false,
  "error": "Error message description"
}
```
