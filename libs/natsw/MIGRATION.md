# Миграция на natsw

## Пример: auth сервис

### До (auth/internal/nats/subscriber.go)

```go
func (s *Subscriber) handleRegistrationApproved(msg *nats.Msg) {
    var event RegistrationApprovedEvent
    if err := json.Unmarshal(msg.Data, &event); err != nil {
        slog.Error("Failed to unmarshal registration approved event", "error", err)
        return
    }

    ctx := context.Background()

    user, err := s.regService.ApproveRegistration(ctx, event.RequestID, event.ApproverID)
    if err != nil {
        slog.Error("Failed to approve registration", "request_id", event.RequestID, "error", err)
        return
    }

    slog.Info("Registration approved, user created", "request_id", event.RequestID, "user_id", user.ID)
}
```

### После (с natsw)

```go
import "github.com/artmexbet/raibecas/libs/natsw"

type RegistrationApprovedEvent struct {
    RequestID  uuid.UUID `json:"request_id"`
    ApproverID uuid.UUID `json:"approver_id"`
}

// В инициализации сервиса
func setupNATS(nc *nats.Conn, regService IRegistrationService) error {
    client := natsw.NewClient(nc,
        natsw.WithLogger(slog.Default()),
        natsw.WithRecover(),
        natsw.WithTimeout(30*time.Second),
    )

    // Типизированная подписка с автоматическим trace context
    _, err := natsw.SubscribeTyped(client, "admin.registration.approved",
        func(ctx context.Context, event *RegistrationApprovedEvent) error {
            // ctx уже содержит trace context и автоматическое логирование
            span := trace.SpanFromContext(ctx)
            span.AddEvent("Processing registration approval")

            user, err := regService.ApproveRegistration(ctx, event.RequestID, event.ApproverID)
            if err != nil {
                // Ошибка автоматически залогируется middleware
                return fmt.Errorf("failed to approve registration: %w", err)
            }

            slog.InfoContext(ctx, "Registration approved",
                "request_id", event.RequestID,
                "user_id", user.ID,
            )
            return nil
        },
    )

    return err
}
```

## Преимущества миграции

### 1. Меньше boilerplate кода
- ❌ Ручная десериализация JSON
- ❌ Создание context.Background()
- ❌ Ручная обработка ошибок логирования
- ✅ Всё автоматически через middleware

### 2. Distributed tracing из коробки
```go
// Старый код - trace теряется
ctx := context.Background()
service.DoWork(ctx) // нет trace context

// Новый код - trace передаётся автоматически
natsw.SubscribeTyped(client, "topic", func(ctx context.Context, msg *Event) error {
    // ctx содержит полный trace context от publisher
    service.DoWork(ctx) // trace продолжается
    return nil
})
```

### 3. Type safety
```go
// Старый код - runtime ошибки
var event interface{}
json.Unmarshal(msg.Data, &event) // может упасть
data := event.(MyType) // может паниковать

// Новый код - compile-time проверка
natsw.SubscribeTyped(client, "topic", func(ctx context.Context, event *MyType) error {
    // event всегда *MyType, гарантированно
    return nil
})
```

### 4. Автоматический recover
```go
// Старый код - паника убьёт подписку
func handler(msg *nats.Msg) {
    // если паника - подписка умирает
    processMessage(msg)
}

// Новый код - паника перехватывается
client := natsw.NewClient(nc, natsw.WithRecover())
// паника залогируется, подписка продолжит работать
```

### 5. Request-Reply становится проще

#### До
```go
func handleRequest(msg *nats.Msg) {
    var req MyRequest
    if err := json.Unmarshal(msg.Data, &req); err != nil {
        respondError(msg, err)
        return
    }

    resp := processRequest(&req)
    
    respData, err := json.Marshal(resp)
    if err != nil {
        respondError(msg, err)
        return
    }

    msg.Respond(respData)
}
```

#### После
```go
natsw.HandleRequestTyped(client, "my.request",
    func(ctx context.Context, req *MyRequest) (*MyResponse, error) {
        // Просто возвращаем результат - всё остальное автоматически
        return processRequest(ctx, req)
    },
)
```

## План миграции сервисов

### Этап 1: Auth service
- [x] Создать natsw библиотеку
- [ ] Обновить subscriber.go
- [ ] Обновить publisher.go
- [ ] Добавить OpenTelemetry tracer
- [ ] Тесты

### Этап 2: Gateway service
- [ ] Мигрировать connector.go на natsw
- [ ] Использовать RequestTyped для синхронных запросов

### Этап 3: Index service
- [ ] Мигрировать nats_consumer.go
- [ ] Добавить трейсинг для pipeline

### Этап 4: Chat service
- [ ] Мигрировать на natsw
- [ ] Использовать для pub/sub чатов

## Обратная совместимость

natsw полностью совместим с обычным nats.go:

```go
client := natsw.NewClient(nc)

// Можно использовать raw nats.Conn если нужно
rawConn := client.Conn()
rawConn.Publish("topic", data) // старый API всё ещё работает

// Или использовать новый API
client.Subscribe("topic", func(ctx context.Context, msg *nats.Msg) error {
    // новый API с контекстом
    return nil
})
```

## Конфигурация OpenTelemetry

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
)

// Настройка трейсера (один раз при старте приложения)
func setupTracing() {
    exporter, _ := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://localhost:14268/api/traces"),
    ))
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("auth-service"),
        )),
    )
    
    otel.SetTracerProvider(tp)
}

// После этого natsw автоматически использует глобальный tracer
client := natsw.NewClient(nc)
// trace propagation работает автоматически
```
