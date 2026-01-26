# OpenTelemetry Tracing в Auth Service

## Проблема

При работе с OpenTelemetry traces в auth сервисе возникала ошибка:
```
2026/01/26 13:42:14 traces export: context deadline exceeded: rpc error: code = DeadlineExceeded desc = context deadline exceeded
```

### Причины

1. **TracerProvider не был инициализирован** - auth сервис не настраивал OpenTelemetry, хотя natsw библиотека создавала spans
2. **Короткий таймаут экспорта** - по умолчанию BatchSpanProcessor имеет таймаут экспорта 5 секунд
3. **Контекст запроса завершался раньше экспорта** - spans создавались с контекстом NATS сообщения, который мог истечь до завершения экспорта

## Решение

### 1. Инициализация TracerProvider

Добавлен модуль `internal/telemetry/tracing.go` с функцией `InitTracer()`, которая:

- Создает OTLP gRPC exporter для отправки трейсов в Jaeger
- Настраивает BatchSpanProcessor с оптимальными параметрами
- Регистрирует TracerProvider глобально через `otel.SetTracerProvider()`

### 2. Конфигурация телеметрии

Добавлена секция `TelemetryConfig` в `config.Config`:

```go
type TelemetryConfig struct {
    Enabled         bool          // Включить/выключить трейсинг
    ServiceName     string        // Имя сервиса для трейсов
    ServiceVersion  string        // Версия сервиса
    OTLPEndpoint    string        // Адрес OTLP collector (Jaeger)
    ExportTimeout   time.Duration // Таймаут для экспорта (30s)
    BatchTimeout    time.Duration // Интервал батчинга (5s)
    MaxQueueSize    int           // Размер очереди spans (2048)
    MaxExportBatch  int           // Размер батча для экспорта (512)
}
```

### 3. Параметры BatchSpanProcessor

#### ExportTimeout (30 секунд)
- Время, которое процессор ждет завершения экспорта одного батча
- Увеличен с 5 до 30 секунд для надежной отправки в Jaeger
- Предотвращает ошибки "context deadline exceeded"

#### BatchTimeout (5 секунд)
- Максимальное время между экспортами
- Spans экспортируются либо по заполнению батча, либо по истечении времени
- Обеспечивает регулярную отправку трейсов

#### MaxQueueSize (2048)
- Размер буфера для хранения spans перед экспортом
- Предотвращает потерю spans при высокой нагрузке

#### MaxExportBatchSize (512)
- Количество spans, отправляемых за один раз
- Оптимизирует сетевые запросы

### 4. Корректное завершение

В `Shutdown()` добавлен вызов `telemetry.Shutdown()`, который:

1. Вызывает `ForceFlush()` - принудительно экспортирует все оставшиеся spans
2. Вызывает `Shutdown()` - корректно завершает все ресурсы
3. Использует контекст с таймаутом 30 секунд для завершения

## Переменные окружения

```bash
# Включить трейсинг
TELEMETRY_ENABLED=true

# Имя сервиса (отображается в Jaeger)
TELEMETRY_SERVICE_NAME=auth-service

# Версия сервиса
TELEMETRY_SERVICE_VERSION=1.0.0

# Адрес Jaeger OTLP collector
TELEMETRY_OTLP_ENDPOINT=localhost:4317

# Таймауты и размеры
TELEMETRY_EXPORT_TIMEOUT=30s
TELEMETRY_BATCH_TIMEOUT=5s
TELEMETRY_MAX_QUEUE_SIZE=2048
TELEMETRY_MAX_EXPORT_BATCH=512
```

## Архитектура трейсинга

```
NATS Message -> natsw.Client.Subscribe()
                    ↓
    extractContext() создает ctx из NATS headers
                    ↓
    Handler создает spans с этим контекстом
                    ↓
    Spans добавляются в BatchSpanProcessor
                    ↓
    BatchSpanProcessor (независимый процесс):
      - Собирает spans в батчи
      - Экспортирует батчи каждые 5 секунд
      - Использует отдельный контекст для экспорта
      - Таймаут экспорта: 30 секунд
                    ↓
    OTLP gRPC Exporter -> Jaeger (localhost:4317)
```

## Важные моменты

### BatchSpanProcessor работает асинхронно

- Spans добавляются в очередь без блокировки handler'а
- Экспорт происходит в фоновом режиме
- Контекст handler'а НЕ влияет на экспорт spans

### Context.Background() в extractContext()

- В `natsw/client.go:213` используется `context.Background()`
- Это правильно! Контекст для trace propagation не должен иметь таймаут
- Spans будут экспортированы независимо от жизненного цикла запроса

### ForceFlush() при shutdown

- Критически важно вызывать при завершении приложения
- Гарантирует экспорт всех накопленных spans
- Без него последние spans будут потеряны

## Мониторинг

### Проверка работоспособности

1. Запустить Jaeger: `docker-compose -f deploy/docker-compose.dev.yml up -d jaeger`
2. Открыть UI: http://localhost:16686
3. Выбрать сервис "auth-service"
4. Проверить наличие трейсов для операций:
   - `auth.login`
   - `auth.validate`
   - `auth.refresh`
   - `auth.logout`
   - `auth.register`

### Логи

При старте сервиса должно появиться:
```
OpenTelemetry tracer initialized service=auth-service endpoint=localhost:4317 export_timeout=30s batch_timeout=5s
```

При завершении:
```
Shutting down tracer provider...
Tracer provider shut down successfully
```

## Troubleshooting

### Трейсы не появляются в Jaeger

1. Проверить, что Jaeger запущен: `docker ps | grep jaeger`
2. Проверить порт 4317: `netstat -an | findstr 4317`
3. Проверить логи auth сервиса на наличие сообщения об инициализации tracer
4. Проверить `TELEMETRY_ENABLED=true`

### Ошибка "connection refused"

- Проверить `TELEMETRY_OTLP_ENDPOINT` - должен быть без схемы (не `http://`)
- Для Docker: использовать имя сервиса вместо localhost
- Для localhost: убедиться что Jaeger слушает на 4317

### Spans теряются

- Увеличить `TELEMETRY_MAX_QUEUE_SIZE`
- Уменьшить `TELEMETRY_BATCH_TIMEOUT` для более частого экспорта
- Проверить, что вызывается `ForceFlush()` при shutdown

### "context deadline exceeded" все еще появляется

- Увеличить `TELEMETRY_EXPORT_TIMEOUT`
- Проверить сетевую задержку до Jaeger
- Проверить, что Jaeger не перегружен
