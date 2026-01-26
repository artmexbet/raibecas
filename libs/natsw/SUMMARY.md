# natsw - Упрощённая обёртка над NATS

## Что получилось

Создана тонкая, простая обёртка над NATS с тремя основными фичами:

### 1. **Message с контекстом**
```go
type Message struct {
    *nats.Msg      // встроенное NATS сообщение
    Ctx context.Context  // контекст с trace info
}

type HandlerFunc func(*Message) error
```

Обработчики получают `*Message` вместо `*nats.Msg`, что даёт доступ к контексту.

### 2. **Middleware**
```go
type Middleware func(next HandlerFunc) HandlerFunc

// Встроенные:
- LoggingMiddleware    // авто-логирование
- RecoverMiddleware    // защита от паник
- TimeoutMiddleware    // таймауты
- RetryMiddleware      // повторы
- MetadataMiddleware   // извлечение headers в context
```

Каждый middleware может модифицировать `msg.Ctx` перед передачей дальше.

### 3. **Trace Propagation (OpenTelemetry)**
Автоматическая пропагация trace context через NATS headers:
- `Publish(ctx, ...)` → инжектирует trace в headers
- `Subscribe(...)` → извлекает trace из headers в `msg.Ctx`

## Простота использования

**До (plain NATS)**:
```go
nc.Subscribe("topic", func(msg *nats.Msg) {
    // нет контекста
    // нет trace
    // нет middleware
    processMessage(msg.Data)
})
```

**После (natsw)**:
```go
client := natsw.NewClient(nc,
    natsw.WithLogger(logger),
    natsw.WithRecover(),
)

client.Subscribe("topic", func(msg *natsw.Message) error {
    // msg.Ctx - полный контекст с trace
    // middleware отрабатывают автоматически
    // паника не убьёт подписку
    
    var event MyEvent
    json.Unmarshal(msg.Data, &event)
    return processEvent(msg.Ctx, &event)
})
```

## Что НЕ делает библиотека

- ❌ Не навязывает типизацию - каждый сервис парсит JSON сам
- ❌ Не абстрагирует NATS - `*nats.Msg` встроен в `*Message`
- ❌ Не скрывает API - можно использовать `client.Conn()` для прямого доступа
- ❌ Не добавляет сложности - всего 3 файла кода

## Файлы

```
libs/natsw/
├── client.go          (237 строк)  - основной клиент
├── middleware.go      (203 строки) - встроенные middleware
├── client_test.go                  - тесты
├── example/main.go                 - пример
└── README.md                       - документация
```

## Использование в сервисах

Каждый сервис может:
1. Создать клиента с нужными middleware
2. Подписаться на топики
3. Парсить JSON так, как ему удобно
4. Получать автоматический tracing

Пример:
```go
// auth service
client.Subscribe("admin.registration.approved", func(msg *natsw.Message) error {
    var event RegistrationApprovedEvent
    json.Unmarshal(msg.Data, &event)
    
    // trace автоматически продолжается
    user, err := s.regService.ApproveRegistration(msg.Ctx, event.RequestID)
    return err
})
```

## Ключевые преимущества

1. **Баланс**: Не переусложнено, но решает реальные проблемы
2. **Trace out-of-the-box**: Distributed tracing работает сразу
3. **Защита**: Recover middleware не даёт паникам убивать подписки
4. **Гибкость**: Каждый сервис контролирует свою (де)сериализацию
5. **Observability**: Логи, трейсы, метрики через middleware

Библиотека готова к использованию! 🎉
