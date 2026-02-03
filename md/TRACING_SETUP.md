# Distributed Tracing Setup Guide

## Quick Start

### 1. Запустите Jaeger

```bash
docker run -d \
  --name jaeger \
  -p 6831:6831/udp \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Jaeger UI: http://localhost:16686

### 2. Установите переменные окружения

Убедитесь, что у всех сервисов установлены переменные:

```bash
export TELEMETRY_ENABLED=true
export TELEMETRY_OTLP_ENDPOINT=localhost:4317
```

### 3. Запустите сервисы

```bash
# В разных терминалах
cd services/gateway && go run cmd/gateway/main.go
cd services/auth && go run cmd/auth/main.go
cd services/users && go run cmd/users/main.go
cd services/chat && go run cmd/chat/main.go
```

### 4. Сделайте запрос

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'
```

### 5. Посмотрите traces в Jaeger

1. Откройте http://localhost:16686
2. Выберите сервис из dropdown (например, "gateway")
3. Нажмите "Find Traces"

## Архитектура трейсинга

### Компоненты

```
┌─────────────────────────────────────────┐
│         OpenTelemetry SDK               │
│  (libs/telemetry - единая инициализация)│
└──────────────┬──────────────────────────┘
               │
       ┌───────┴───────┐
       │               │
   ┌───▼────┐    ┌────▼───┐
   │HTTP    │    │NATS    │
   │(Fiber) │    │wrapper │
   └───┬────┘    └────┬───┘
       │              │
       │  Trace       │ Trace
       │  Context     │ Context
       │  Headers     │ in Headers
       │              │
       └──────┬───────┘
              │
      ┌───────▼─────────┐
      │ OTLP Exporter   │
      │ (gRPC)          │
      └────────┬────────┘
               │
        ┌──────▼──────┐
        │   Jaeger    │
        │ localhost   │
        │   :6831     │
        └─────────────┘
```

### Как работает пропагация контекста

1. **Gateway** получает HTTP запрос
   - `otelfiber.Middleware` автоматически создает span
   - Trace ID и Span ID генерируются

2. **Gateway → Auth/Users/Chat** (через NATS)
   - `natsw.Client` инжектирует trace context в NATS message headers
   - Используется W3C Trace Context стандарт

3. **Auth/Users/Chat** обрабатывают сообщение
   - При подписке контекст автоматически извлекается
   - Создаются новые spans, связанные с родительским trace

4. **Экспорт spans**
   - `BatchSpanProcessor` группирует spans
   - Отправляет в Jaeger каждые 5 секунд (по умолчанию)
   - Jaeger хранит и отображает полный trace

## Конфигурация сервисов

### Auth Service
```bash
cd services/auth
export TELEMETRY_ENABLED=true
export TELEMETRY_SERVICE_NAME=auth
export TELEMETRY_OTLP_ENDPOINT=localhost:4317
export DB_PASSWORD=yourpassword
go run cmd/auth/main.go
```

### Users Service
```bash
cd services/users
export TELEMETRY_ENABLED=true
export DB_PASSWORD=yourpassword
go run cmd/users/main.go
```

### Chat Service
```bash
cd services/chat
export TELEMETRY_ENABLED=true
go run cmd/chat/main.go
```

## Проверка трейсинга

### 1. Через логи

```bash
# Должно быть сообщение для каждого сервиса:
OpenTelemetry tracer initialized service=auth endpoint=localhost:4317
```

### 2. Через Jaeger UI

1. Откройте http://localhost:16686
2. Выберите сервис
3. Нажмите "Find Traces"
4. Кликните на trace для детального просмотра

### 3. Проверка пропагации контекста

В Jaeger вы должны увидеть:
- Span из Gateway (HTTP запрос)
- Span из NATS publish в Gateway
- Span из NATS subscribe в Auth/Users/Chat
- Все spans должны быть соединены одним Trace ID

## Отключение трейсинга

Если хотите отключить трейсинг для тестирования:

```bash
export TELEMETRY_ENABLED=false
```

Сервисы будут работать без экспорта spans, но код остается одинаковым.

## Проблемы и решения

### Spans не видны в Jaeger

**Проблема**: Сервисы запущены, но spans не появляются в Jaeger UI

**Решение**:
1. Проверьте, что Jaeger запущен: `docker ps | grep jaeger`
2. Проверьте логи: `docker logs jaeger`
3. Убедитесь, что порт 6831 открыт: `netstat -an | grep 6831`
4. Проверьте переменные окружения: `echo $TELEMETRY_ENABLED`

### Broken traces (разорванные traces)

**Проблема**: Spans от разных сервисов не связаны между собой

**Решение**:
1. Убедитесь, что используется `natsw.Client` для всех NATS операций
2. Проверьте, что все сервисы используют одинаковую версию OpenTelemetry
3. Добавьте логирование в `natsw.Client` для отладки headers

### Высокая нагрузка на Jaeger

**Проблема**: Jaeger потребляет много памяти или CPU

**Решение**:
1. Увеличьте `TELEMETRY_BATCH_TIMEOUT` (например, 10s вместо 5s)
2. Уменьшите `TELEMETRY_MAX_QUEUE_SIZE` (например, 512 вместо 2048)
3. Используйте probabilistic sampling (например, 0.1 вместо 1.0)

## Performance Tips

1. **Batch size** - Увеличивайте для экономии памяти, уменьшайте для низкой latency
2. **Timeout** - Больший timeout = более полные traces, но выше latency
3. **Queue size** - Больший размер = больше буферизации, но больше памяти
4. **Sampling** - Используйте для production (экспортируйте только 10-50% traces)

## Дополнительные ресурсы

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Jaeger Getting Started](https://www.jaegertracing.io/docs/getting-started/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [OTEL Go Documentation](https://pkg.go.dev/go.opentelemetry.io/otel)
