# Gateway CORS Configuration

## Текущая конфигурация (Development)

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     "*",
    AllowCredentials: true,
    AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Device-ID",
    AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
}))
```

## Production конфигурация

⚠️ **ВАЖНО:** В production НЕ используйте `AllowOrigins: "*"` с `AllowCredentials: true`!

### Рекомендуемая конфигурация:

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     "https://your-frontend-domain.com",  // Конкретный домен
    AllowCredentials: true,                                 // Для cookies
    AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Device-ID",
    AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
    MaxAge:           3600,                                 // Cache preflight requests
}))
```

### Несколько доменов:

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     "https://app.example.com, https://admin.example.com",
    AllowCredentials: true,
    AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Device-ID",
    AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
    MaxAge:           3600,
}))
```

### Динамическая конфигурация через environment:

```go
// В config/config.go добавить:
type HTTPConfig struct {
    Host            string
    Port            int
    RPS             int
    AllowedOrigins  []string  // НОВОЕ
}

// В server.go:
router.Use(cors.New(cors.Config{
    AllowOrigins:     strings.Join(cfg.AllowedOrigins, ", "),
    AllowCredentials: true,
    AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Device-ID",
    AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
    MaxAge:           3600,
}))
```

### Environment variables:

```bash
# .env или docker-compose.yml
GATEWAY_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
```

## Безопасность

### ✅ Правильно:
```go
AllowOrigins:     "https://example.com"
AllowCredentials: true
```

### ❌ ОПАСНО (не использовать в production):
```go
AllowOrigins:     "*"
AllowCredentials: true
```

Такая комбинация **небезопасна** и может привести к CSRF атакам!

## Тестирование CORS

### cURL:
```bash
curl -H "Origin: https://example.com" \
     -H "Access-Control-Request-Method: POST" \
     -H "Access-Control-Request-Headers: Content-Type, Authorization" \
     -X OPTIONS \
     http://localhost:8080/api/v1/auth/login
```

### JavaScript (fetch):
```javascript
fetch('http://localhost:8080/api/v1/auth/login', {
  method: 'POST',
  credentials: 'include',  // Важно для cookies!
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password123'
  })
})
```

### Axios:
```javascript
const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  withCredentials: true  // Важно для cookies!
});
```

## Cookie настройки

Gateway автоматически устанавливает следующие флаги для cookies:

- `Secure: true` — только HTTPS (требуется в production)
- `HttpOnly: true` — недоступен для JavaScript
- `SameSite: Strict` — защита от CSRF
- `Path: /` — доступен для всех роутов
- `MaxAge: 2592000` — 30 дней (refresh token)

## Checklist для production

- [ ] Заменить `AllowOrigins: "*"` на конкретные домены
- [ ] Включить HTTPS на Gateway
- [ ] Настроить `Secure: true` для cookies (включено по умолчанию)
- [ ] Добавить `GATEWAY_ALLOWED_ORIGINS` в environment
- [ ] Проверить работу CORS с реальным фронтендом
- [ ] Добавить мониторинг CORS ошибок
- [ ] Настроить rate limiting для auth endpoints
