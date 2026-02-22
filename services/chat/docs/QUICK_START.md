# 🚀 Быстрый старт: Сохранение истории чатов в Redis

## За 5 минут до готовности

### Шаг 1: Убедитесь что Redis работает

```bash
# Проверить подключение
redis-cli ping
# Должно вывести: PONG
```

### Шаг 2: Установите переменные окружения

```bash
# Для локальной разработки
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_DB=0
export REDIS_CHAT_TTL=86400          # 24 часа
export REDIS_MESSAGE_TTL=86400       # 24 часа
```

Или в `.env` файле:
```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=0
REDIS_CHAT_TTL=86400
REDIS_MESSAGE_TTL=86400
```

### Шаг 3: Запустите приложение

```bash
cd services/chat
go run ./cmd/chat
```

Приложение запустится на `http://localhost:8080`

### Шаг 4: Отправьте первое сообщение

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user1",
    "input": "Hello!"
  }'
```

✅ Сообщение сохранено в Redis!

### Шаг 5: Отправьте второе сообщение

```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user1",
    "input": "Tell me more"
  }'
```

✅ Нейромодуль используется контекст из первого сообщения!

### Шаг 6: Проверьте историю в Redis

```bash
redis-cli
> GET "chat:history:user1"
```

Вы видите JSON массив всех сообщений!

### Шаг 7: Очистите историю

```bash
curl -X DELETE http://localhost:8080/api/v1/chat/user1
```

✅ История удалена!

## 🐳 Если у вас нет Redis

### Вариант 1: Docker

```bash
docker run -d -p 6379:6379 redis:7-alpine
```

### Вариант 2: Docker Compose

Создайте `docker-compose.dev.yml`:

```yaml
version: '3.8'

services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

volumes:
  redis_data:
```

Запустите:
```bash
docker-compose -f docker-compose.dev.yml up -d
```

## ✨ Что происходит внутри

```
1️⃣ User sends message
   ↓
2️⃣ Service saves to Redis: chat:history:user1
   ↓
3️⃣ Service gets response from AI in chunks
   ↓
4️⃣ Each chunk is buffered and sent to client
   ↓
5️⃣ When done, complete message is saved to Redis
   ↓
6️⃣ Temporary buffer is deleted
   ↓
7️⃣ Next message uses saved history as context
```

## 🔍 Мониторинг

### Смотреть все активные чаты

```bash
redis-cli
> KEYS "chat:history:*"
```

### Проверить размер чата

```bash
redis-cli
> GET "chat:history:user1" | wc -c
```

### Увидеть TTL (когда удалится)

```bash
redis-cli
> TTL "chat:history:user1"
```

Если возвращает `86400`, данные удалятся через 86400 секунд (24 часа)

## 🧪 Тестирование

### Запустить unit-тесты

```bash
cd services/chat
go test ./internal/redis -v
```

Все 5 тестов должны пройти ✅

### Выполнить curl тесты

```bash
# Тест 1: Новый пользователь
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "alice", "input": "Hi"}'

# Тест 2: Второе сообщение
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "alice", "input": "More info"}'

# Тест 3: Новый пользователь
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id": "bob", "input": "Hello"}'

# Тест 4: Проверить в Redis
redis-cli KEYS "chat:history:*"
# Должно показать:
# 1) "chat:history:alice"
# 2) "chat:history:bob"

# Тест 5: Очистить alice
curl -X DELETE http://localhost:8080/api/v1/chat/alice

# Тест 6: Проверить что alice удален
redis-cli KEYS "chat:history:*"
# Должно показать только bob
```

## ⚠️ Распространенные проблемы

### "Connection refused"
```
Решение: Redis не запущен
> docker run -d -p 6379:6379 redis:7-alpine
```

### "Key not found"
```
Решение: Это нормально, значит это новый пользователь
> История создается автоматически при первом сообщении
```

### "Timeout"
```
Решение: Проверьте что нейромодуль работает
> Проверьте OLLAMA_HOST и другие зависимости
```

### "Message not persisting"
```
Решение: Проверьте TTL
> redis-cli TTL "chat:history:userID"
> Если -1, значит TTL не установлен
> Если -2, значит ключ удален (истек TTL)
```

## 📚 Дальнейшее изучение

- 📖 [CHAT_PERSISTENCE.md](CHAT_PERSISTENCE.md) - Полная документация
- ⚙️ [REDIS_CONFIG.md](REDIS_CONFIG.md) - Конфигурация и best practices
- 💡 [USAGE_EXAMPLES.md](USAGE_EXAMPLES.md) - Примеры и сценарии
- ✅ [IMPLEMENTATION_CHECKLIST.md](IMPLEMENTATION_CHECKLIST.md) - Что было сделано

## 🎯 Ключевые моменты

✅ **Все сообщения сохраняются автоматически**
- User message → сохраняется сразу
- Assistant response → сохраняется при завершении

✅ **История используется для контекста**
- Каждый новый запрос включает всю предыдущую историю
- Нейромодуль "помнит" весь разговор

✅ **Автоматическое удаление**
- Через 24 часа (по умолчанию) все сообщения удаляются
- Это не требует ручного вмешательства

✅ **Надежность**
- Если Redis недоступен - чат все равно работает
- Без Redis будет работать, но без сохранения истории

## 🤝 Получить помощь

Если что-то не работает:

1. Проверьте логи приложения
2. Проверьте Redis подключение (`redis-cli ping`)
3. Проверьте переменные окружения
4. Запустите тесты (`go test ./internal/redis -v`)
5. Прочитайте документацию выше

---

**Версия**: 1.0.0
**Статус**: 🟢 PRODUCTION READY
**Последнее обновление**: 2025-01-15

