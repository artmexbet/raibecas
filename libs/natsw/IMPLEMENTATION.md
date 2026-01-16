# NATS Wrapper Implementation Summary

## Что было реализовано

Создана обёртка над NATS в `libs/natsw` с полной поддержкой:

### 1. Middleware система
- **LoggingMiddleware** - автоматическое логирование всех сообщений с duration
- **RecoverMiddleware** - защита от паник с stack trace
- **TimeoutMiddleware** - таймауты для обработчиков
- **RetryMiddleware** - повторные попытки при ошибках
- **MetadataMiddleware** - извлечение метаданных из headers в context
- **RateLimitMiddleware** - ограничение скорости обработки
- **ValidationMiddleware** - валидация сообщений
- **ChainMiddleware** - комбинирование middleware
- Возможность создания кастомных middleware

### 2. Context propagation
Каждый обработчик получает `context.Context`:
- Автоматическая передача контекста во все хендлеры
- Middleware могут обогащать контекст (user_id, request_id и т.д.)
- Timeout и cancellation через context

### 3. Distributed Tracing (OpenTelemetry)
Автоматическая пропагация trace context через NATS headers:
- **Publish** - инжектирует trace context в headers
- **Subscribe** - извлекает trace context из headers
- **Request-Reply** - сохраняет trace chain через запросы
- Поддержка OpenTelemetry propagators (W3C TraceContext, Baggage и др.)
- Автоматическое создание spans для publish/request операций

### 4. Type safety через Generics
```go
// Типизированная подписка
SubscribeTyped[T any](client, subject, handler func(ctx, *T) error)

// Типизированный request-reply
HandleRequestTyped[Req, Resp any](client, subject, handler func(ctx, *Req) (*Resp, error))
RequestTyped[Req, Resp any](client, ctx, subject, *Req) (*Resp, error)

// Типизированная публикация
PublishTyped[T any](client, ctx, subject, *T) error
```

## Как передаются метаданные через NATS

### Headers сообщений
OpenTelemetry propagator автоматически добавляет стандартные заголовки:
- `traceparent` - W3C trace context (trace-id, span-id, flags)
- `tracestate` - vendor-specific trace data
- Кастомные headers через middleware (X-User-Id, X-Request-Id и т.д.)

### Формат
```
Message:
  Subject: "user.events"
  Headers:
    traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
    X-User-Id: "user-123"
    X-Request-Id: "req-456"
  Data: {"user_id":"123","action":"login"} (JSON)
```

### Поток данных
```
Service A (Publisher)
  ↓
  1. context.Context с trace span
  2. Propagator.Inject() → добавляет traceparent в headers
  3. JSON.Marshal(data) → body сообщения
  4. NATS.Publish(msg)
  ↓
NATS Server
  ↓
Service B (Subscriber)
  5. NATS.Subscribe() получает msg
  6. Propagator.Extract() → восстанавливает trace context из headers
  7. JSON.Unmarshal(msg.Data) → восстанавливает данные
  8. handler(ctx, data) с полным trace context
```

## Преимущества подхода

1. **Прозрачность** - trace context передаётся автоматически, не требуя вмешательства в бизнес-логику
2. **Стандартизация** - использует W3C TraceContext стандарт через OpenTelemetry
3. **Совместимость** - работает с любыми OpenTelemetry-совместимыми системами (Jaeger, Zipkin, и т.д.)
4. **Extensibility** - легко добавлять свои headers через middleware
5. **Type safety** - generics обеспечивают compile-time проверку типов
6. **Observability** - полная прослеживаемость запросов между сервисами

## Использование в сервисах

```go
// Инициализация
nc, _ := nats.Connect(cfg.NATS.URL)
client := natsw.NewClient(nc,
    natsw.WithLogger(logger),
    natsw.WithRecover(),
    natsw.WithTimeout(30*time.Second),
)

// Подписка
natsw.SubscribeTyped(client, "events", func(ctx context.Context, event *Event) error {
    // ctx содержит trace span, можно создавать child spans
    span := trace.SpanFromContext(ctx)
    span.AddEvent("processing event")
    return processEvent(ctx, event)
})

// Публикация (trace автоматически пропагируется)
natsw.PublishTyped(client, ctx, "events", &Event{...})
```

## Файлы

- `client.go` - основной клиент с middleware и trace propagation
- `middleware.go` - встроенные middleware
- `typed.go` - generic функции для type safety
- `client_test.go` - тесты и примеры использования
- `example/main.go` - полноценный пример использования
- `README.md` - документация

## Интеграция с существующими сервисами

Все сервисы (auth, chat, gateway, index) могут постепенно переходить на использование natsw вместо прямого использования nats.go, получая:
- Автоматический distributed tracing
- Централизованное логирование
- Защиту от паник
- Типизированные обработчики
