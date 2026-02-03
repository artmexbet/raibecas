# Quick Start: Distributed Tracing in 5 Minutes

## 🚀 Быстрый старт (5 минут)

### Шаг 1: Запустить Jaeger (1 минута)

```bash
docker run -d \
  --name jaeger \
  -p 6831:6831/udp \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Проверить: http://localhost:16686 должен открыться пустой Jaeger UI

### Шаг 2: Установить переменные окружения (30 секунд)

```bash
export TELEMETRY_ENABLED=true
export TELEMETRY_OTLP_ENDPOINT=localhost:4317
export DB_PASSWORD=yourpassword  # для auth и users
export REDIS_PASSWORD=           # если требуется
```

### Шаг 3: Запустить сервисы (2 минуты)

В разных терминалах:

```bash
# Terminal 1 - Gateway
cd services/gateway
go run cmd/gateway/main.go

# Terminal 2 - Auth
cd services/auth
go run cmd/auth/main.go

# Terminal 3 - Users
cd services/users
go run cmd/users/main.go

# Terminal 4 - Chat
cd services/chat
go run cmd/chat/main.go
```

Должны увидеть в логах для каждого сервиса:
```
OpenTelemetry tracer initialized
  service=<service-name>
  endpoint=localhost:4317
```

### Шаг 4: Сделать запрос (30 секунд)

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'
```

### Шаг 5: Посмотреть traces в Jaeger (30 секунд)

1. Откройте http://localhost:16686
2. В dropdown выберите "gateway" (или другой сервис)
3. Нажмите "Find Traces"
4. Кликните на любой trace

## 📊 Что вы увидите в Jaeger

Для каждого запроса будет полная цепочка (trace graph):

```
[gateway] HTTP GET /api/v1/auth/login
  ↓
[nats] publish auth.login (trace context in headers)
  ↓
[auth] subscribe auth.login (extract trace context)
  ↓
[auth] handler.login
  ├─ [db] postgres query
  ├─ [redis] operation
  └─ [nats] publish response
      ↓
[gateway] nats request reply
```

## 🔍 Практический примеры

### Пример 1: Посмотреть время выполнения запроса

1. В Jaeger откройте trace
2. Посмотрите duration у корневого span (gateway HTTP GET)
3. Разверните spans для деталей каждого сервиса

### Пример 2: Найти медленный запрос

1. В Jaeger выберите сервис
2. В фильтре введите: `duration > 1s`
3. Нажмите "Find Traces"
4. Посмотрите какой span самый медленный

### Пример 3: Отследить ошибку

1. Сделайте запрос с неправильными данными
2. В Jaeger найдите trace
3. Посмотрите какой span вернул ошибку
4. Кликните на span для деталей

## 💡 Совет: Включение/отключение трейсинга

Трейсинг можно быстро включить/отключить:

```bash
# Включить
export TELEMETRY_ENABLED=true

# Отключить
export TELEMETRY_ENABLED=false

# Перезапустить сервис - готово!
```

## 📚 Дальнейшее изучение

- Полная техническая документация: [`md/TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md)
- Расширенный гайд по запуску: [`md/TRACING_SETUP.md`](TRACING_SETUP.md)
- Чек-лист реализации: [`md/TRACING_CHECKLIST.md`](TRACING_CHECKLIST.md)
- Полный summary: [`md/TRACING_COMPLETE_SUMMARY.md`](TRACING_COMPLETE_SUMMARY.md)

## 🆘 Если что-то не работает

### Spans не видны в Jaeger

```bash
# 1. Проверьте, что Jaeger запущен
docker ps | grep jaeger

# 2. Проверьте логи Jaeger
docker logs jaeger | tail -20

# 3. Проверьте переменные окружения
echo $TELEMETRY_ENABLED
echo $TELEMETRY_OTLP_ENDPOINT

# 4. Проверьте, что сервисы работают
curl http://localhost:8080/livez  # gateway
```

### Ошибка подключения к Jaeger

```bash
# Проверьте, что Jaeger слушает на правильном порту
netstat -an | grep 6831

# Если портов нет, перезапустите Jaeger
docker stop jaeger
docker rm jaeger
docker run -d --name jaeger -p 6831:6831/udp -p 16686:16686 jaegertracing/all-in-one
```

### Разорванные traces (broken traces)

Если spans из разных сервисов не связаны:

```bash
# Убедитесь, что все сервисы используют одинаковую конфигурацию
export TELEMETRY_OTLP_ENDPOINT=localhost:4317

# Перезапустите все сервисы с новой переменной
```

## 🎓 Что дальше?

После освоения базовых возможностей:

1. **Custom spans** - Добавить свои spans в код для более детального анализа
2. **Sampling** - Настроить вероятностное sampling для production
3. **Metrics** - Связать Prometheus метрики с traces
4. **Logs** - Добавить trace ID в структурированные логи
5. **Alerts** - Настроить ��лерты на основе данных из Jaeger

## 📞 Контакты

Если нужна помощь:
- Проверьте логи в терминалах с сервисами
- Посмотрите документацию в папке `md/`
- Проверьте Jaeger UI для деталей spans

---

**Готово!** 🎉 Теперь у вас есть полностью рабочая система distributed tracing!
