# Миграция на новую систему JWT токенов

## Что изменилось

### 1. Интерфейсы и зависимости

**Было:**
```go
type JWTManager interface {
    GenerateAccessToken(userID uuid.UUID, role string) (string, error)
    GenerateRefreshToken() (string, error)
    ValidateAccessToken(token string) (*jwt.Claims, error)
    GetRefreshTokenTTL() time.Duration
}

authService := service.NewAuthService(userRepo, tokenStore, jwtManager)
```

**Стало:**
```go
type TokenManager interface {
    GenerateAccessToken(metadata *TokenMetadata) (string, *AccessTokenClaims, error)
    GenerateRefreshToken(metadata *TokenMetadata) (string, *RefreshTokenMetadata, error)
    ValidateAccessToken(ctx context.Context, token string, fingerprint string) (*ValidationResult, error)
    RotateRefreshToken(ctx context.Context, oldTokenID string, metadata *TokenMetadata) (string, string, error)
    // ... и другие методы
}

// TokenStore теперь внутри JWT Manager
jwtManager := jwt.NewManager(secret, issuer, accessTTL, refreshTTL, tokenStore)
authService := service.NewAuthService(userRepo, jwtManager)
```

### 2. Структура токенов

**Access Token теперь содержит:**
- `jti` - уникальный ID токена для blacklist
- `fingerprint` - для защиты от XSS
- `device_id` - для multi-device support
- `token_type` - тип токена

**Refresh Token теперь содержит:**
- `token_id` - уникальный ID
- `token_family` - для обнаружения кражи
- `rotation_hash` - для проверки ротации
- `fingerprint` - для защиты от XSS
- `previous_jti` - ссылка на предыдущий токен

### 3. Redis структура

**Новые ключи:**
```
auth:refresh:{tokenID}                      - метаданные токена (было: refresh_token:data:{token})
auth:user:{userID}:refresh                  - Set токенов (было: refresh_token:user:{userID}:tokens)
auth:family:{familyID}                      - новый: семьи токенов
auth:blacklist:{jti}                        - новый: blacklist для access токенов
```

## Шаги миграции

### Шаг 1: Обновление кода (без breaking changes)

```bash
# 1. Скопировать новые файлы
cp jwt/types.go services/auth/pkg/jwt/
cp jwt/interfaces.go services/auth/pkg/jwt/
cp storeredis/token_store.go services/auth/internal/storeredis/

# 2. Обновить jwt.go
# Старые методы помечены как deprecated, но продолжают работать
```

### Шаг 2: Обновление инициализации

**Файл: `cmd/auth/main.go`**

```go
// Было:
jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
tokenStore := storeredis.NewTokenStore(redisClient)
authService := service.NewAuthService(userRepo, tokenStore, jwtManager)

// Стало:
tokenStore := storeredis.NewTokenStoreRedis(redisClient, logger)
jwtManager := jwt.NewManager(
    cfg.JWT.Secret,
    cfg.JWT.Issuer,
    cfg.JWT.AccessTTL,
    cfg.JWT.RefreshTTL,
    tokenStore,
)
authService := service.NewAuthService(userRepo, jwtManager)
```

### Шаг 3: Обновление handlers

**Login Handler:**

```go
// Было:
result, userID, err := s.authService.Login(ctx, req)
// Возвращали: TokenPair{AccessToken, RefreshToken}

// Стало:
result, err := s.authService.Login(ctx, req)
// Возвращаем: LoginResult{AccessToken, RefreshToken, TokenID, Fingerprint, UserID}

// ВАЖНО: Добавляем fingerprint в HttpOnly cookie
http.SetCookie(w, &http.Cookie{
    Name:     "fp",
    Value:    result.Fingerprint,
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
    MaxAge:   int(7 * 24 * time.Hour.Seconds()),
})
```

**Refresh Handler:**

```go
// Было:
req := domain.RefreshRequest{
    RefreshToken: body.RefreshToken,
    DeviceID:     body.DeviceID,
}
result, userID, err := s.authService.RefreshTokens(ctx, req)

// Стало:
// 1. Получаем fingerprint из cookie
fpCookie, err := r.Cookie("fp")
if err != nil {
    return errors.New("fingerprint missing")
}

req := domain.RefreshRequest{
    RefreshToken: body.RefreshToken,
    TokenID:      body.TokenID,  // НОВОЕ: добавляем token_id
    DeviceID:     body.DeviceID,
}
result, err := s.authService.RefreshTokens(ctx, req, fpCookie.Value)
```

**Validate Handler/Middleware:**

```go
// Было:
claims, err := s.authService.ValidateAccessToken(ctx, token)

// Стало:
// 1. Получаем fingerprint из cookie
fpCookie, err := r.Cookie("fp")
if err != nil {
    return errors.New("fingerprint missing")
}

claims, err := s.authService.ValidateAccessToken(ctx, token, fpCookie.Value)
```

**Logout Handler:**

```go
// Было:
err := s.authService.Logout(ctx, userID, refreshToken)

// Стало:
// Нужен JTI из access token (получаем из claims в middleware)
jti := r.Context().Value("jti").(string)
err := s.authService.Logout(ctx, tokenID, jti)

// Удаляем fingerprint cookie
http.SetCookie(w, &http.Cookie{
    Name:   "fp",
    Value:  "",
    MaxAge: -1,
})
```

### Шаг 4: Обновление middleware

```go
func AuthMiddleware(jwtManager jwt.TokenManager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Получаем токен
            token := extractBearerToken(r)
            
            // 2. Получаем fingerprint из cookie
            fpCookie, err := r.Cookie("fp")
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // 3. Валидируем с fingerprint
            result, err := jwtManager.ValidateAccessToken(r.Context(), token, fpCookie.Value)
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            if !result.Valid {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            // 4. Добавляем claims в context (включая JTI для logout)
            ctx := context.WithValue(r.Context(), "user_id", result.Claims.UserID)
            ctx = context.WithValue(ctx, "role", result.Claims.Role)
            ctx = context.WithValue(ctx, "jti", result.Claims.JTI)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Шаг 5: Обновление domain models

**Файл: `internal/domain/user.go`**

```go
// Добавляем TokenID в RefreshRequest
type RefreshRequest struct {
    RefreshToken string
    TokenID      string // НОВОЕ
    DeviceID     string
    UserAgent    string
    IPAddress    string
}
```

### Шаг 6: Обновление клиентской части

**Frontend/Mobile app:**

```typescript
// Login response
interface LoginResponse {
    access_token: string;
    refresh_token: string;
    token_id: string;      // НОВОЕ: сохранить для refresh
    user_id: string;
    // fingerprint автоматически в cookie
}

// Refresh request
async function refreshTokens() {
    const response = await fetch('/auth/refresh', {
        method: 'POST',
        credentials: 'include',  // ВАЖНО: отправляет cookies
        body: JSON.stringify({
            refresh_token: localStorage.getItem('refresh_token'),
            token_id: localStorage.getItem('token_id'),  // НОВОЕ
            device_id: getDeviceId(),
        }),
    });
}

// Все запросы
fetch('/api/endpoint', {
    credentials: 'include',  // ВАЖНО: отправляет fingerprint cookie
    headers: {
        'Authorization': `Bearer ${accessToken}`,
    },
});
```

## Порядок развёртывания

### 1. Подготовка (Day 0)
- [ ] Код review новой реализации
- [ ] Unit тесты
- [ ] Integration тесты
- [ ] Load тесты

### 2. Staging (Day 1-3)
- [ ] Развернуть на staging
- [ ] Проверить совместимость со старыми токенами
- [ ] Проверить новые токены
- [ ] QA тестирование

### 3. Production Rollout (Day 4-7)

**Phase 1: Backend deployment**
```bash
# 1. Развернуть новый код (backward compatible)
kubectl apply -f auth-service-v2.yaml

# 2. Проверить health checks
kubectl get pods -l app=auth-service

# 3. Мониторинг логов
kubectl logs -f deployment/auth-service | grep "token"
```

**Phase 2: Frontend deployment**
```bash
# 1. Развернуть новый frontend с поддержкой token_id
# 2. Старые клиенты продолжают работать
# 3. Новые клиенты используют новую систему
```

**Phase 3: Cleanup (через 7 дней)**
```bash
# Когда все старые токены истекли:
# 1. Удалить legacy код
# 2. Удалить старые Redis ключи
# 3. Обновить документацию
```

## Rollback Plan

Если что-то пошло не так:

```bash
# 1. Откатить deployment
kubectl rollout undo deployment/auth-service

# 2. Восстановить старую версию frontend
# 3. Токены, выпущенные новой версией, станут невалидными
# 4. Пользователям придётся перелогиниться
```

## Мониторинг после миграции

### Метрики для отслеживания

```
# Количество активных токенов
auth_active_tokens_total{type="refresh"} 

# Ротации токенов
auth_token_rotations_total

# Обнаружение кражи
auth_token_theft_detected_total

# Fingerprint mismatches
auth_fingerprint_mismatch_total

# Blacklist hits
auth_blacklist_hits_total
```

### Алерты

```yaml
- alert: HighTokenTheftDetectionRate
  expr: rate(auth_token_theft_detected_total[5m]) > 0.1
  annotations:
    summary: "High rate of token theft detection"

- alert: HighFingerprintMismatch
  expr: rate(auth_fingerprint_mismatch_total[5m]) > 1
  annotations:
    summary: "High rate of fingerprint mismatches"
```

## Проблемы и решения

### Проблема: "Fingerprint cookie отсутствует"

**Причина:** Клиент не отправляет cookies

**Решение:**
```typescript
// Добавить credentials: 'include' во все fetch запросы
fetch(url, { credentials: 'include' })
```

### Проблема: "CORS errors с cookies"

**Причина:** Неправильная CORS конфигурация

**Решение:**
```go
c := cors.New(cors.Options{
    AllowedOrigins:   []string{"https://your-frontend.com"},
    AllowCredentials: true,  // ВАЖНО
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
})
```

### Проблема: "Токены не работают в Safari"

**Причина:** Safari блокирует third-party cookies

**Решение:**
- Используйте same-site deployment
- Или SameSite=None; Secure

## Тестирование

### Manual Testing Checklist

- [ ] Login - получаем fingerprint cookie
- [ ] Validate - с правильным fingerprint
- [ ] Validate - с неправильным fingerprint (должен fail)
- [ ] Refresh - успешная ротация
- [ ] Refresh - повторное использование токена (должен fail + revoke family)
- [ ] Logout - токен в blacklist
- [ ] Logout all - все токены revoked
- [ ] Multi-device - независимые сессии

### Load Testing

```bash
# Apache Bench
ab -n 10000 -c 100 \
   -H "Authorization: Bearer $TOKEN" \
   -H "Cookie: fp=$FINGERPRINT" \
   https://api.example.com/validate

# Expected: < 50ms p95 latency
```

## Контакты для поддержки

При возникновении проблем:
1. Проверьте логи сервиса
2. Проверьте Redis (количество ключей, память)
3. Создайте issue в репозитории
