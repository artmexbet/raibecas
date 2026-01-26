# Auth Middleware Implementation - Summary

## ✅ Реализовано

### 1. Authentication Middleware (`internal/server/middleware.go`)
- Валидация access token из Authorization header
- Проверка fingerprint из HttpOnly cookie
- Интеграция с Auth Service через NATS (`auth.validate`)
- Сохранение AuthUser в Fiber Context
- Детальное логирование (Debug/Warn/Error)

### 2. Разделение роутов (`internal/server/server.go`)

**Публичные роуты:**
- `POST /api/v1/auth/login`
- `POST /api/v1/registration-requests`

**Защищённые роуты:**
- Все Auth роуты (кроме login)
- Все Documents роуты
- Все Users роуты  
- Все Registration management роуты

### 3. Удалены дублирующие методы
- ❌ `setupAuthRoutes()` → переехало в `setupPublicRoutes()` + `setupProtectedRoutes()`
- ❌ `setupDocumentRoutes()` → переехало в `setupProtectedRoutes()`
- ❌ `setupUsersRoutes()` → переехало в `setupProtectedRoutes()`
- ❌ `setupRegistrationRequestRoutes()` → переехало в `setupPublicRoutes()` + `setupProtectedRoutes()`

## 🔐 Безопасность

**Защита от XSS:**
- Access token в header (не cookie)
- Fingerprint в HttpOnly cookie

**Защита от CSRF:**
- SameSite cookie policy
- Требуется token + fingerprint

**Защита от Token Theft:**
- Device fingerprinting
- Валидация fingerprint при каждом запросе

## 📊 AuthUser Context

```go
type AuthUser struct {
    ID   uuid.UUID  // из валидации токена
    Role string     // для RBAC
    JTI  string     // для операций с токеном
}
```

Доступно через: `getAuthUser(c)`

## 📁 Новые файлы

```
services/gateway/
├── internal/server/
│   └── middleware.go                    # 🆕 Auth middleware
└── docs/
    ├── authentication-middleware.md     # 🆕 Документация
    ├── testing-auth-middleware.md       # 🆕 Тестирование
    ├── auth-middleware-flow.mermaid     # 🆕 Sequence diagram
    └── routes-diagram.mermaid           # 🆕 Схема роутов
```

## 🚀 Быстрый тест

```bash
# 1. Попытка доступа без токена → 401
curl http://localhost:8080/api/v1/users

# 2. Login → получить токен
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}' \
  -c cookies.txt

# 3. Доступ с токеном → 200
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer {access_token}" \
  -b cookies.txt
```

## 🔄 Интеграция

**NATS Flow:**
```
Gateway Middleware
    ↓ NATS Request
auth.validate topic
    ↓
Auth Service
    ↓ Validates:
- Token signature
- Expiration  
- Fingerprint
- Blacklist
    ↓ NATS Response
Gateway Middleware
    ↓
Stores AuthUser in context
```

## ⚠️ Ошибки

**401 Unauthorized возвращается если:**
- Отсутствует Authorization header
- Неверный формат Bearer token
- Отсутствует fingerprint cookie
- Токен невалиден или истёк
- Fingerprint не совпадает с сохранённым

## 📚 Документация

См. подробную документацию:
- `docs/authentication-middleware.md` - полное описание
- `docs/testing-auth-middleware.md` - тестирование
- `docs/auth-middleware-flow.mermaid` - диаграмма процесса
- `docs/routes-diagram.mermaid` - схема роутов

## 🎯 Результат

✅ Все роуты защищены (кроме login и создания registration request)  
✅ Валидация через Auth Service по NATS  
✅ Device fingerprinting работает  
✅ Данные пользователя доступны в handlers  
✅ Безопасность на уровне industry standards  
