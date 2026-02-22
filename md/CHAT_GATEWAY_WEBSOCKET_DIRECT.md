# Прямое WebSocket соединение Gateway <-> Chat Service

## Обзор

Chat сервис теперь связан с Gateway через прямое WebSocket соединение без использования NATS для стриминга сообщений.

## Архитектура

```
┌─────────┐      WebSocket       ┌──────────┐      WebSocket       ┌──────────┐
│ Client  │◄────────────────────►│ Gateway  │◄────────────────────►│   Chat   │
│         │  /ws/chat/:userID    │          │  ws://chat:8081/ws   │  Service │
│         │                      │          │      /chat           │          │
└─────────┘                      └──────────┘                      └──────────┘
```

## Flow

1. Клиент устанавливает WebSocket соединение с Gateway: `/ws/chat/{userID}`
2. Gateway сразу же устанавливает WebSocket соединение с Chat сервисом
3. Сообщения проксируются напрямую между клиентом и Chat сервисом
4. Gateway выступает как прозрачный прокси

## Преимущества прямого WebSocket

- **Меньше задержки**: Нет промежуточного хопа через NATS
- **Меньше нагрузки на NATS**: Не гоняем трафик чата через message broker
- **Проще отладка**: Прямое соединение проще мониторить
- **Потоковая передача**: WebSocket идеален для стриминга ответов

## WebSocket API

### Gateway Endpoint
```
WS /ws/chat/{userID}
```

### Chat Service Endpoint
```
WS /ws/chat?userID={userID}
```

### Формат сообщений

Запрос от клиента:
```json
{
  "input": "Текст запроса",
  "user_id": "user123"
}
```

Ответ от Chat сервиса (стриминг):
```json
{
  "done": false,
  "message": {
    "role": "assistant",
    "content": "Часть ответа..."
  }
}
```

Последнее сообщение:
```json
{
  "done": true,
  "message": {
    "role": "assistant",
    "content": "Полный ответ"
  }
}
```

## Конфигурация

### Gateway

Переменные окружения:
```env
GATEWAY_HTTP_HOST=0.0.0.0
GATEWAY_HTTP_PORT=8080
CHAT_WS_URL=ws://localhost:8081/ws/chat
```

### Chat Service

Переменные окружения:
```env
HTTP_HOST=0.0.0.0
HTTP_PORT=8081
```

## Файлы

### Gateway
- `internal/connector/chat_ws_connector.go` - WebSocket connector с прямым соединением
- `internal/server/chat.go` - WebSocket handler
- `internal/config/config.go` - добавлена конфигурация ChatServiceConfig

### Chat Service
- `internal/handler/http/chat.go` - WebSocket handler
- `internal/handler/http/handler.go` - регистрация WebSocket маршрута

## Безопасность

- Gateway проверяет аутентификацию перед WebSocket upgrade
- UserID передается через URL параметр и header X-User-ID
- Соединения закрываются при disconnect любой из сторон

## Graceful Shutdown

- При shutdown Gateway закрывает все WebSocket соединения
- Chat сервис корректно завершает HTTP/WebSocket сервер
