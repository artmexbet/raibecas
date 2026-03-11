# Рефакторинг Gateway и Chat сервисов

## Дата: 15.02.2026

## Цель
Приведение Gateway и Chat сервисов к единым архитектурным паттернам, установленным в Auth и Users сервисах.

---

## Изменения в Gateway сервисе

### 1. main.go - Улучшенная обработка ошибок
**Было:**
```go
func main() {
    gateway := app.New()
    if err := gateway.Run(); err != nil {
        panic(err)
    }
}
```

**Стало:**
```go
func main() {
    a, err := app.New()
    if err != nil {
        slog.Error("failed to initialize app", "error", err)
        os.Exit(1)
    }

    if err := a.Run(); err != nil {
        slog.Error("app error", "error", err)
        os.Exit(1)
    }
}
```

**Улучшения:**
- Замена `panic()` на структурированное логирование с `slog.Error()`
- Корректные exit codes через `os.Exit(1)`
- Явная обработка ошибок инициализации

### 2. config.go - Интеграция cleanenv
**Было:**
- Ручное определение конфигурации в app.go
- `getEnvOrDefault()` функции

**Стало:**
```go
func Load() (*Config, error) {
    var cfg Config
    if err := cleanenv.ReadEnv(&cfg); err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    return &cfg, nil
}
```

**Улучшения:**
- Использование `cleanenv` для загрузки конфигурации
- Структурированные env-prefix'ы для разных компонентов
- Валидация и значения по умолчанию через теги
- Добавлены недостающие секции: TelemetryConfig, CORSConfig

### 3. app.go - Dependency Injection и Graceful Shutdown
**Было:**
```go
func New() *App {
    // Прямая инициализация без возврата ошибок
    return &App{cfg: cfg}
}

func (a *App) Run() error {
    // Инициализация здесь
}
```

**Стало:**
```go
func New() (*App, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load configuration: %w", err)
    }
    // ... инициализация зависимостей с проверкой ошибок
    return &App{...}, nil
}

func (a *App) Run() error {
    // Graceful shutdown с таймаутами
}
```

**Улучшения:**
- `New()` возвращает `(*App, error)` вместо `*App`
- Централизованная загрузка конфигурации
- Корректная очистка ресурсов при ошибках (например, `natsConn.Close()` при ошибке трассировки)
- Использование `slog.ErrorContext()` вместо `slog.Error()`
- Конфигурируемый shutdown timeout

### 4. server.go - Конфигурируемый CORS
**Было:**
```go
router.Use(cors.New(cors.Config{
    AllowOrigins: "http://localhost:3000", // Хардкод
}))
```

**Стало:**
```go
func New(cfg *config.HTTPConfig, corsCfg config.CORSConfig, ...) {
    router.Use(cors.New(cors.Config{
        AllowOrigins: corsCfg.AllowOrigins,
    }))
}
```

---

## Изменения в Chat сервисе

### 1. main.go - Унифицированный подход
**Было:**
```go
func main() {
    _app := app.New()
    err := _app.Run()
    if err != nil {
        panic(err)
    }
}
```

**Стало:**
```go
func main() {
    a, err := app.New()
    if err != nil {
        slog.Error("failed to initialize app", "error", err)
        os.Exit(1)
    }

    if err := a.Run(); err != nil {
        slog.Error("app error", "error", err)
        os.Exit(1)
    }
}
```

### 2. app.go - Полный рефакторинг
**Было:**
```go
type App struct {
    tracerProvider *trace.TracerProvider
}

func New() *App {
    return &App{}
}

func (a *App) Run() error {
    // Вся инициализация здесь
    // Нет graceful cleanup
}
```

**Стало:**
```go
type App struct {
    cfg            *config.Config
    qdrantClient   *qdrant.Client
    redisClient    *redis.Client
    tracerProvider *trace.TracerProvider
    svc            *service.Chat
    api            *http.Handler
}

func New() (*App, error) {
    // Инициализация всех зависимостей
    // Корректная очистка при ошибках
    return &App{...}, nil
}

func (a *App) Run() error {
    // Graceful shutdown с очисткой всех ресурсов
}
```

**Улучшения:**
- Сохранение всех зависимостей в структуре App
- Таймаут при проверке соединений (10 секунд)
- Корректная очистка ресурсов при ошибках инициализации
- Graceful shutdown для всех компонентов:
  - HTTP сервер
  - Tracer provider
  - Redis клиент
  - Qdrant клиент

---

## Общие улучшения

### Архитектурные паттерны
1. **Dependency Injection**: Все зависимости создаются в `New()` и передаются в конструкторы
2. **Error Wrapping**: Использование `fmt.Errorf("...: %w", err)` для сохранения контекста ошибок
3. **Graceful Shutdown**: Корректная остановка всех компонентов с таймаутами
4. **Context-aware Logging**: Использование `slog.ErrorContext()` и `slog.InfoContext()`
5. **Structured Configuration**: Централизованная конфигурация через `cleanenv`

### Согласованность с другими сервисами
| Сервис | `main.go` паттерн | `New()` возврат | Graceful Shutdown |
|--------|-------------------|-----------------|-------------------|
| auth   | ✅ slog + os.Exit | ✅ (*App, error) | ✅ |
| users  | ✅ slog + os.Exit | ✅ (*App, error) | ✅ |
| gateway| ✅ slog + os.Exit | ✅ (*App, error) | ✅ |
| chat   | ✅ slog + os.Exit | ✅ (*App, error) | ✅ |

---

## Тестирование

```bash
# Gateway сервис
cd services/gateway
go build ./...      # ✅
go vet ./...        # ✅
go mod tidy         # ✅

# Chat сервис
cd services/chat
go build ./...      # ✅
go vet ./...        # ✅
go mod tidy         # ✅
```

---

## Следующие шаги
1. Добавить unit-тесты для app.go в обоих сервисах
2. Рассмотреть добавление health check endpoints
3. Унифицировать конфигурацию telemetry между всеми сервисами
