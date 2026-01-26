# NATS Wrapper (natsw)

Обёртка над NATS с поддержкой middleware, контекста и distributed tracing.

## Основные возможности

- **Middleware**: логирование, recover, метрики, кастомные middleware
- **Context propagation**: автоматическая передача context.Context в обработчики
- **Distributed tracing**: пропагация trace context через NATS headers (OpenTelemetry)
- **Type safety**: типизированные хендлеры с автоматической (де)сериализацией

## Быстрый старт

### Создание клиента

```go
import "github.com/artmexbet/raibecas/libs/natsw"

// Подключение к NATS
nc, _ := nats.Connect(nats.DefaultURL)

// Создание клиента с middleware
client := natsw.NewClient(nc,
    natsw.WithLogger(slog.Default()),
    natsw.WithRecover(),
)
```

### Subscribe (обработка событий)

```go
type UserEvent struct {
    UserID string `json:"user_id"`
    Action string `json:"action"`
}

// Подписка - каждый сервис сам парсит JSON
_, err := client.Subscribe("user.events", func(msg *natsw.Message) error {
    var event UserEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        return err
    }
    
    // msg.Ctx содержит trace context и другие метаданные
    slog.InfoContext(msg.Ctx, "Event received", "user_id", event.UserID)
    return nil
})
```

### Publish (отправка событий)

```go
event := &UserEvent{
    UserID: "123",
    Action: "login",
}

data, _ := json.Marshal(event)

// Publish с автоматической пропагацией trace context
err := client.Publish(ctx, "user.events", data)
```

### Request-Reply

```go
// Сервер
_, _ = client.Subscribe("auth.login", func(msg *natsw.Message) error {
    var req LoginRequest
    json.Unmarshal(msg.Data, &req)
    
    // Обработка запроса
    resp := LoginResponse{Token: "jwt-token"}
    respData, _ := json.Marshal(resp)
    
    return msg.Respond(respData)
})

// Клиент
req := LoginRequest{Username: "user", Password: "pass"}
reqData, _ := json.Marshal(req)

reqMsg := nats.NewMsg("auth.login")
reqMsg.Data = reqData

respMsg, _ := client.RequestMsg(ctx, reqMsg)

var resp LoginResponse
json.Unmarshal(respMsg.Data, &resp)
```

## Передача метаданных через NATS

### Заголовки сообщений

Все метаданные передаются через NATS headers:

- `X-Trace-Id` - trace ID для distributed tracing (OpenTelemetry)
- `X-Span-Id` - span ID текущего span
- `X-Parent-Span-Id` - span ID родительского span
- `X-Trace-Flags` - флаги трейсинга (sampled/not sampled)
- `X-Request-Id` - опциональный request ID
- `X-User-Id` - опциональный user ID (из контекста)

### Формат сообщений

Все сообщения передаются в JSON формате в теле (Data) NATS сообщения.

## Middleware

### Встроенные middleware

```go
// Логирование всех сообщений
natsw.WithLogger(logger)

// Recover от паник
natsw.WithRecover()

// Метрики (требует OpenTelemetry meter)
natsw.WithMetrics(meter)

// Таймауты
natsw.WithTimeout(5 * time.Second)
```

### Кастомные middleware

```go
func authMiddleware(next natsw.HandlerFunc) natsw.HandlerFunc {
    return func(ctx context.Context, msg *nats.Msg) error {
        // Проверка авторизации
        userID := msg.Header.Get("X-User-Id")
        if userID == "" {
            return errors.New("unauthorized")
        }
        
        // Добавление userID в контекст
        ctx = context.WithValue(ctx, "user_id", userID)
        
        return next(ctx, msg)
    }
}

client := natsw.NewClient(nc, natsw.WithMiddleware(authMiddleware))
```

## Distributed Tracing

Библиотека автоматически интегрируется с OpenTelemetry:

```go
// При Publish trace context автоматически передаётся
client.Publish(ctx, "topic", data)

// При Subscribe trace context автоматически восстанавливается
client.Subscribe("topic", func(msg *natsw.Message) error {
    // msg.Ctx содержит восстановленный trace context
    span := trace.SpanFromContext(msg.Ctx)
    span.AddEvent("Processing message")
    return nil
})
```

## Примеры использования

Смотрите `example/main.go` для полного примера использования.
