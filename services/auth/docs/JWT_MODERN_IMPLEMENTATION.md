# JWT Token Management - Современная реализация

## Обзор

Реализована современная система управления JWT токенами с поддержкой всех актуальных стандартов безопасности.

## Основные возможности

### 1. **Enhanced Security Features**

#### Token Fingerprint (Защита от XSS)
- Каждый токен содержит уникальный fingerprint
- Fingerprint хранится в HttpOnly cookie на клиенте
- При валидации токена fingerprint должен совпадать
- Защищает от кражи токенов через XSS атаки

#### JWT ID (jti)
- Уникальный идентификатор для каждого access токена
- Позволяет добавлять токены в blacklist
- Необходим для немедленной инвалидации токенов

#### Refresh Token Rotation
- Автоматическая ротация refresh токенов при каждом обновлении
- Старый refresh token автоматически отзывается
- Предотвращает повторное использование токенов

#### Token Family
- Все refresh токены связаны в "семьи"
- При обнаружении повторного использования токена отзывается вся семья
- Защита от replay атак и кражи токенов

### 2. **Архитектурные улучшения**

#### Инверсия зависимостей
```go
// Интерфейсы вместо конкретных типов
type TokenStore interface {
    StoreRefreshToken(ctx context.Context, metadata *RefreshTokenMetadata, ttl time.Duration) error
    GetRefreshToken(ctx context.Context, tokenID string) (*RefreshTokenMetadata, error)
    RevokeRefreshToken(ctx context.Context, tokenID string) error
    // ... другие методы
}

type TokenManager interface {
    GenerateAccessToken(metadata *TokenMetadata) (string, *AccessTokenClaims, error)
    ValidateAccessToken(ctx context.Context, token string, fingerprint string) (*ValidationResult, error)
    RotateRefreshToken(ctx context.Context, oldTokenID string, metadata *TokenMetadata) (string, string, error)
    // ... другие методы
}
```

#### Separation of Concerns
- `jwt.Manager` - генерация и валидация токенов
- `TokenStore` - хранение и управление токенами в Redis
- `AuthService` - бизнес-логика аутентификации

### 3. **Redis Storage Patterns**

#### Структура ключей
```
auth:refresh:{tokenID}                      - Полные метаданные refresh токена
auth:user:{userID}:refresh                  - Set всех refresh токенов пользователя
auth:user:{userID}:device:{deviceID}:refresh - Токен конкретного устройства
auth:family:{familyID}                      - Set токенов в одной семье
auth:blacklist:{jti}                        - Blacklist для access токенов
```

#### Pipeline для атомарности
```go
pipe := s.client.Pipeline()
pipe.Set(ctx, tokenKey, data, ttl)
pipe.SAdd(ctx, userTokensKey, metadata.TokenID)
pipe.Expire(ctx, userTokensKey, ttl)
_, err := pipe.Exec(ctx)
```

## Примеры использования

### Инициализация

```go
import (
    "github.com/artmexbet/raibecas/services/auth/pkg/jwt"
    "github.com/artmexbet/raibecas/services/auth/internal/storeredis"
    "github.com/redis/go-redis/v9"
)

// 1. Создаём Redis клиент
redisClient := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// 2. Создаём token store
tokenStore := storeredis.NewTokenStoreRedis(redisClient, logger)

// 3. Создаём JWT manager
jwtManager := jwt.NewManager(
    "your-secret-key",
    "your-issuer",
    15*time.Minute,  // access token TTL
    7*24*time.Hour,  // refresh token TTL
    tokenStore,
)

// 4. Создаём auth service
authService := service.NewAuthService(userRepo, jwtManager)
```

### Login с fingerprint

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    var req domain.LoginRequest
    // ... parse request
    
    result, err := authService.Login(r.Context(), req)
    if err != nil {
        // handle error
        return
    }
    
    // Устанавливаем fingerprint в HttpOnly cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "fp",
        Value:    result.Fingerprint,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        MaxAge:   int(7 * 24 * time.Hour.Seconds()),
    })
    
    // Возвращаем токены клиенту
    json.NewEncoder(w).Encode(map[string]interface{}{
        "access_token":  result.AccessToken,
        "refresh_token": result.RefreshToken,
        "token_id":      result.TokenID,
        "user_id":       result.UserID,
    })
}
```

### Валидация access token

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Получаем токен из Authorization header
        authHeader := r.Header.Get("Authorization")
        token := strings.TrimPrefix(authHeader, "Bearer ")
        
        // Получаем fingerprint из cookie
        fpCookie, err := r.Cookie("fp")
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Валидируем токен
        claims, err := authService.ValidateAccessToken(r.Context(), token, fpCookie.Value)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Добавляем claims в context
        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "role", claims.Role)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Refresh tokens с ротацией

```go
func RefreshHandler(w http.ResponseWriter, r *http.Request) {
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
    
    result, err := authService.RefreshTokens(r.Context(), refreshReq, fpCookie.Value)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Возвращаем новые токены
    json.NewEncoder(w).Encode(map[string]interface{}{
        "access_token":  result.AccessToken,
        "refresh_token": result.RefreshToken,
        "token_id":      result.TokenID,
    })
}
```

### Logout с blacklist

```go
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
    // Получаем claims из context (из middleware)
    userID := r.Context().Value("user_id").(uuid.UUID)
    jti := r.Context().Value("jti").(string)
    
    var req struct {
        TokenID string `json:"token_id"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Logout отзывает refresh token и добавляет access token в blacklist
    err := authService.Logout(r.Context(), req.TokenID, jti)
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

### Logout со всех устройств

```go
func LogoutAllHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(uuid.UUID)
    
    err := authService.LogoutAll(r.Context(), userID)
    if err != nil {
        http.Error(w, "Failed to logout", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}
```

## Безопасность

### 1. Защита от XSS атак
- Fingerprint хранится в HttpOnly cookie
- Access token передаётся в Authorization header
- Fingerprint проверяется при каждом запросе

### 2. Защита от CSRF атак
- SameSite=Strict для cookies
- Дополнительно можно использовать CSRF tokens

### 3. Защита от Replay атак
- Refresh Token Rotation
- Token Family для обнаружения повторного использования
- Автоматическая отзыв всей семьи токенов при подозрении на кражу

### 4. Защита от Token Theft
- Fingerprint mismatch detection
- IP и User-Agent tracking
- Немедленная инвалидация через blacklist

### 5. Криптографическая стойкость
- Refresh tokens генерируются через crypto/rand
- HMAC-SHA256 для rotation hash
- RS256 можно легко добавить в будущем

## Мониторинг и аудит

Все операции логируются через structured logging (slog):

```go
s.logger.InfoContext(ctx, "stored refresh token",
    "token_id", metadata.TokenID,
    "user_id", metadata.UserID,
    "device_id", metadata.DeviceID,
    "family", metadata.TokenFamily,
    "ttl", ttl)

s.logger.WarnContext(ctx, "revoked entire token family (possible theft detected)",
    "family", tokenFamily,
    "revoked_count", len(tokenIDs))
```

## Миграция со старой системы

1. Развернуть новый код с поддержкой обоих версий
2. Генерировать новые токены по новой системе
3. Старые токены продолжают работать до истечения TTL
4. После истечения всех старых токенов удалить legacy код

## Performance

### Redis Pipeline
Использование pipeline сокращает количество network round-trips:
- Одна операция вместо 3-5 для сохранения токена
- Атомарность операций

### Индексы
- Быстрый поиск по user_id через Set
- Быстрый поиск по device_id
- Быстрый поиск по token_family

### TTL Management
- Автоматическое удаление истекших токенов
- Нет необходимости в cleanup jobs

## Тестирование

```go
// Пример unit теста
func TestTokenRotation(t *testing.T) {
    // Setup
    store := NewMockTokenStore()
    manager := jwt.NewManager("secret", "issuer", time.Minute, time.Hour, store)
    
    // Generate initial tokens
    metadata := &jwt.TokenMetadata{
        UserID: uuid.New(),
        Role: "user",
        // ...
    }
    
    accessToken1, refreshToken1, _ := manager.GenerateAccessToken(metadata)
    refreshToken1, refreshMeta1, _ := manager.GenerateRefreshToken(metadata)
    
    // Rotate
    accessToken2, refreshToken2, err := manager.RotateRefreshToken(
        context.Background(),
        refreshMeta1.TokenID,
        metadata,
    )
    
    assert.NoError(t, err)
    assert.NotEqual(t, accessToken1, accessToken2)
    assert.NotEqual(t, refreshToken1, refreshToken2)
    
    // Old token should be revoked
    _, err = manager.ValidateRefreshToken(context.Background(), refreshMeta1.TokenID, metadata.Fingerprint)
    assert.Error(t, err)
}
```

## Best Practices

1. **Всегда используйте HTTPS** - токены должны передаваться только по защищённому соединению
2. **Короткие TTL для access токенов** - 15 минут вполне достаточно
3. **Храните refresh токены в secure storage** - не в localStorage
4. **Используйте fingerprint** - обязательно для публичных API
5. **Мониторьте подозрительную активность** - множественные refresh с разных IP
6. **Регулярно ротируйте secrets** - используйте key rotation

## Troubleshooting

### "Token reuse detected"
- Кто-то пытается использовать уже использованный refresh token
- Вся семья токенов автоматически отозвана
- Пользователь должен перелогиниться

### "Fingerprint mismatch"
- Возможная XSS атака или кража токена
- Семья токенов отозвана
- Проверьте логи на предмет подозрительной активности

### "Token is blacklisted"
- Токен был явно отозван (logout)
- Пользователь должен получить новый токен через refresh

## Roadmap

- [ ] Добавить поддержку RS256 (асимметричная криптография)
- [ ] Реализовать JWKS endpoint для публичных ключей
- [ ] Добавить rate limiting для refresh операций
- [ ] Реализовать device fingerprinting
- [ ] Добавить geo-location tracking
- [ ] Интеграция с audit log системой
