# JWT Token System - Quick Start

## ✅ Что было сделано

Реализована современная система управления JWT токенами с поддержкой:
- **Token Fingerprint** - защита от XSS атак
- **JWT ID (jti)** - blacklist для access токенов
- **Refresh Token Rotation** - автоматическая ротация
- **Token Family** - обнаружение кражи токенов
- **Инверсия зависимостей** - чистая архитектура
- **Redis Pipeline** - оптимизация производительности

## 📁 Структура файлов

### Новые файлы:
```
services/auth/
├── pkg/jwt/
│   ├── types.go           ✅ Новые типы данных
│   ├── interfaces.go      ✅ Интерфейсы (DI)
│   ├── jwt.go             ✅ Обновлён Manager
│   └── manager_test.go    ✅ Расширенные тесты
├── internal/
│   ├── storeredis/
│   │   └── token_store.go ✅ Новая Redis реализация
│   └── service/
│       └── auth_service.go ✅ Обновлён для новых токенов
└── docs/
    ├── JWT_SUMMARY.md              ✅ Обзор изменений
    ├── JWT_MODERN_IMPLEMENTATION.md ✅ Полная документация
    └── JWT_MIGRATION_GUIDE.md       ✅ Руководство по миграции
```

## 🚀 Быстрый старт

### 1. Инициализация (обновите `cmd/auth/main.go`):

```go
package main

import (
    "log"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/artmexbet/raibecas/services/auth/pkg/jwt"
    "github.com/artmexbet/raibecas/services/auth/internal/storeredis"
    "github.com/artmexbet/raibecas/services/auth/internal/service"
)

func main() {
    // 1. Redis клиент
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 2. Token Store
    tokenStore := storeredis.NewTokenStoreRedis(redisClient, logger)

    // 3. JWT Manager
    jwtManager := jwt.NewManager(
        cfg.JWT.Secret,
        cfg.JWT.Issuer,
        15*time.Minute,  // access token TTL
        7*24*time.Hour,  // refresh token TTL
        tokenStore,
    )

    // 4. Auth Service
    authService := service.NewAuthService(userRepo, jwtManager)

    // 5. Запуск сервера...
}
```

### 2. Обновление handlers:

#### Login Handler:
```go
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    var req domain.LoginRequest
    // ... parse request
    
    result, err := h.authService.Login(r.Context(), req)
    if err != nil {
        // handle error
        return
    }
    
    // ВАЖНО: Устанавливаем fingerprint в HttpOnly cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "fp",
        Value:    result.Fingerprint,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   int(7 * 24 * time.Hour.Seconds()),
    })
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "access_token":  result.AccessToken,
        "refresh_token": result.RefreshToken,
        "token_id":      result.TokenID,
        "user_id":       result.UserID,
    })
}
```

#### Auth Middleware:
```go
func AuthMiddleware(jwtManager jwt.TokenManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Извлекаем токен из Authorization header
            token := extractBearerToken(r)
            
            // 2. Получаем fingerprint из cookie
            fpCookie, err := r.Cookie("fp")
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // 3. Валидируем токен с fingerprint
            result, err := jwtManager.ValidateAccessToken(r.Context(), token, fpCookie.Value)
            if err != nil || !result.Valid {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // 4. Добавляем данные в context
            ctx := context.WithValue(r.Context(), "user_id", result.Claims.UserID)
            ctx = context.WithValue(ctx, "role", result.Claims.Role)
            ctx = context.WithValue(ctx, "jti", result.Claims.JTI) // Для logout
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### Refresh Handler:
```go
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
    var req struct {
        RefreshToken string `json:"refresh_token"`
        TokenID      string `json:"token_id"`
        DeviceID     string `json:"device_id"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Получаем fingerprint из cookie
    fpCookie, err := r.Cookie("fp")
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    refreshReq := domain.RefreshRequest{
        RefreshToken: req.RefreshToken,
        TokenID:      req.TokenID,
        DeviceID:     req.DeviceID,
        UserAgent:    r.UserAgent(),
        IPAddress:    r.RemoteAddr,
    }
    
    result, err := h.authService.RefreshTokens(r.Context(), refreshReq, fpCookie.Value)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "access_token":  result.AccessToken,
        "refresh_token": result.RefreshToken,
        "token_id":      result.TokenID,
    })
}
```

#### Logout Handler:
```go
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
    var req struct {
        TokenID string `json:"token_id"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Получаем JTI из context (установлен в middleware)
    jti := r.Context().Value("jti").(string)
    
    err := h.authService.Logout(r.Context(), req.TokenID, jti)
    if err != nil {
        http.Error(w, "Failed to logout", http.StatusInternalServerError)
        return
    }
    
    // Удаляем fingerprint cookie
    http.SetCookie(w, &http.Cookie{
        Name:   "fp",
        Value:  "",
        MaxAge: -1,
    })
    
    w.WriteHeader(http.StatusNoContent)
}
```

### 3. Frontend изменения:

```typescript
// Login
const response = await fetch('/auth/login', {
    method: 'POST',
    credentials: 'include', // ВАЖНО: для cookies
    body: JSON.stringify({ email, password, device_id }),
});

const data = await response.json();
// Сохраняем токены
localStorage.setItem('access_token', data.access_token);
localStorage.setItem('refresh_token', data.refresh_token);
localStorage.setItem('token_id', data.token_id); // НОВОЕ

// Все запросы
fetch('/api/endpoint', {
    credentials: 'include', // ВАЖНО: для fingerprint cookie
    headers: {
        'Authorization': `Bearer ${accessToken}`,
    },
});

// Refresh
await fetch('/auth/refresh', {
    method: 'POST',
    credentials: 'include',
    body: JSON.stringify({
        refresh_token: localStorage.getItem('refresh_token'),
        token_id: localStorage.getItem('token_id'),
        device_id: getDeviceId(),
    }),
});
```

## 🧪 Тестирование

```bash
# Unit тесты
go test ./pkg/jwt/... -v

# Integration тесты
go test ./internal/service/... -v

# Benchmark
go test -bench=. ./pkg/jwt/...

# Покрытие
go test -cover ./pkg/jwt/...
```

## 📊 Проверка работы

### 1. Проверка Redis:
```bash
# Подключиться к Redis
redis-cli

# Проверить ключи
KEYS auth:*

# Посмотреть токены пользователя
SMEMBERS auth:user:{userID}:refresh

# Посмотреть семью токенов
SMEMBERS auth:family:{familyID}

# Проверить blacklist
EXISTS auth:blacklist:{jti}
```

### 2. Логи:
Все операции логируются через slog:
```bash
# Успешная ротация
INFO stored refresh token token_id=... user_id=... family=...

# Обнаружение кражи
WARN revoked entire token family (possible theft detected) family=... revoked_count=3

# Fingerprint mismatch
ERROR fingerprint mismatch token_id=...
```

## 🔐 Безопасность

### Checklist:
- [x] Fingerprint в HttpOnly cookie
- [x] Access token в Authorization header
- [x] Refresh token rotation
- [x] Token family для обнаружения кражи
- [x] Blacklist для access токенов
- [x] HTTPS только
- [x] SameSite=Strict для cookies
- [x] CORS с credentials

### CORS конфигурация:
```go
c := cors.New(cors.Options{
    AllowedOrigins:   []string{"https://your-frontend.com"},
    AllowCredentials: true,  // ВАЖНО для cookies
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
})
```

## 📖 Документация

- [JWT_SUMMARY.md](./JWT_SUMMARY.md) - Краткий обзор
- [JWT_MODERN_IMPLEMENTATION.md](./JWT_MODERN_IMPLEMENTATION.md) - Полная документация
- [JWT_MIGRATION_GUIDE.md](./JWT_MIGRATION_GUIDE.md) - Руководство по миграции

## ⚠️ Важные замечания

1. **HTTPS обязателен** - токены должны передаваться только по защищённому соединению
2. **Fingerprint в cookie** - без него система не работает
3. **credentials: 'include'** - обязательно для всех fetch запросов
4. **CORS** - правильная настройка с AllowCredentials
5. **TokenID** - сохранять на клиенте вместе с refresh token

## 🐛 Troubleshooting

### "Fingerprint cookie отсутствует"
- Проверьте что клиент отправляет `credentials: 'include'`
- Проверьте CORS настройки (AllowCredentials: true)

### "Token reuse detected"
- Кто-то пытается использовать уже использованный токен
- Вся семья токенов автоматически отозвана
- Пользователь должен перелогиниться

### "Fingerprint mismatch"
- Возможная XSS атака или кража токена
- Проверьте логи на подозрительную активность

## 📞 Поддержка

При возникновении проблем:
1. Проверьте логи сервиса
2. Проверьте Redis (ключи и память)
3. Проверьте CORS настройки
4. Проверьте что клиент отправляет cookies

## ✨ Следующие шаги

1. Обновить handlers
2. Обновить middleware
3. Обновить frontend
4. Тестирование на staging
5. Развертывание в production

---

**Готово к использованию!** Все тесты проходят, документация готова.
