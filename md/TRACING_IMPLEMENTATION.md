# Unified Tracing Implementation Guide

## Overview

Реализована единая система трейсинга (distributed tracing) для всех микросервисов проекта с использованием OpenTelemetry и OTLP (OpenTelemetry Protocol) для экспорта spans в Jaeger.

## Architecture

### Компоненты

1. **libs/telemetry** - Единый пакет инициализации tracer для всех сервисов
2. **NATS Wrapper (libs/natsw)** - Автоматическая пропагация контекста трасировки через сообщения NATS
3. **Сервисы** - Gateway, Auth, Users, Chat с инициализацией tracer provider

### Как работает трейсинг

```
Gateway (HTTP)
    ↓
NATS (с пропагацией trace context в headers)
    ↓
Auth/Users/Chat (получают и продолжают trace)
    ↓
OTLP Exporter
    ↓
Jaeger (localhost:6831)
```

## Инициализация Tracer

### Для новых сервисов

```go
import "github.com/artmexbet/raibecas/libs/telemetry"

// В функции инициализации приложения
tp, err := telemetry.InitTracer(telemetry.TracerConfig{
    ServiceName:    "your-service",
    ServiceVersion: "1.0.0",
    OTLPEndpoint:   "localhost:4317",
    Enabled:        true,
    ExportTimeout:  30 * time.Second,
    BatchTimeout:   5 * time.Second,
    MaxQueueSize:   2048,
    MaxExportBatch: 512,
})
if err != nil {
    return err
}

// При завершении приложения
defer telemetry.Shutdown(context.Background(), tp)
```

## Пропагация контекста через NATS

NATS wrapper автоматически пропагирует trace context:

```go
// При отправке сообщения
err := client.Publish(ctx, "subject", data)

// При подписке на сообщения
sub, err := client.Subscribe("subject", func(msg *natsw.Message) error {
    // msg.Ctx содержит trace context
    return handler(msg)
})
```

## Переменные окружения

Для конфигурации трейсинга используются переменные окружения (префикс `TELEMETRY_`):

```bash
TELEMETRY_ENABLED=true
TELEMETRY_SERVICE_NAME=auth
TELEMETRY_SERVICE_VERSION=1.0.0
TELEMETRY_OTLP_ENDPOINT=localhost:4317
TELEMETRY_EXPORT_TIMEOUT=30s
TELEMETRY_BATCH_TIMEOUT=5s
TELEMETRY_MAX_QUEUE_SIZE=2048
TELEMETRY_MAX_EXPORT_BATCH=512
```

## Запуск Jaeger локально

```bash
docker run -d \
  --name jaeger \
  -p 6831:6831/udp \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Jaeger UI доступен на http://localhost:16686

## Сервисы с трейсингом

### ✅ Gateway
- Инициализация: `services/gateway/internal/app/app.go`
- HTTP middleware: `otelfiber.Middleware` для автоматического трейсинга HTTP запросов
- NATS integration: Пропагация trace context во все исходящие запросы

### ✅ Auth
- Инициализация: `services/auth/internal/server/auth_server.go`
- NATS subscriptions: Автоматическое получение trace context из входящих сообщений
- Config-driven: Использует переменные окружения для конфигурации

### ✅ Users
- Инициализация: `services/users/internal/app/app.go`
- Database tracing: `otelpgx` для трейсинга PostgreSQL запросов
- Prometheus metrics: Интегрирована с metrics для полной observability

### ✅ Chat
- Инициализация: `services/chat/internal/app/app.go`
- Graceful shutdown: Корректный shutdown tracer provider при завершении

## Best Practices

1. **Всегда закрывайте tracer provider** при завершении приложения
2. **Используйте контекст из сообщений** при обработке NATS событий
3. **Пропагируйте контекст** при синхронных и асинхронных операциях
4. **Отключайте metrics** в middleware если не нужны для оптимизации: `otelfiber.WithoutMetrics(true)`

## Troubleshooting

### Spans не видны в Jaeger

1. Проверьте, что Jaeger запущен: `docker ps | grep jaeger`
2. Проверьте переменные окружения `TELEMETRY_*`
3. Убедитесь, что `TELEMETRY_ENABLED=true`
4. Проверьте логи сервиса: `OpenTelemetry tracer initialized`

### Разорванные traces (broken traces)

Если traces не связаны между сервисами:
1. Убедитесь, что используется `natsw.Client` для NATS операций
2. Проверьте, что trace context пропагируется в NATS headers
3. Убедитесь, что все сервисы используют одинаковую версию OpenTelemetry

### Высокая нагрузка на Jaeger

Отрегулируйте параметры в `telemetry.TracerConfig`:
- Увеличьте `BatchTimeout` для группирования spans
- Уменьшите `MaxQueueSize` для более быстрого экспорта
- Используйте выборочное логирование (sampling)
