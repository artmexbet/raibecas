# Distributed Tracing Documentation Index

## 📑 Документация по реализации трейсинга

Все файлы находятся в папке `md/` проекта.

### 🚀 Для быстрого старта (начните отсюда!)

**[`TRACING_QUICK_START.md`](TRACING_QUICK_START.md)** ⭐ НАЧНИТЕ С ЭТОГО
- Quick start за 5 минут
- Пошаговые инструкции запуска
- Практические примеры
- Быстрые ответы на вопросы "Что-то не работает?"

**Время на прочтение**: 5-10 минут  
**Для кого**: Все разработчики

---

### 📚 Полная документация

**[`TRACING_SETUP.md`](TRACING_SETUP.md)** - Гайд по запуску и конфигурации
- Детальный Quick Start с Jaeger
- Конфигурация каждого сервиса
- Проверка трейсинга
- Performance tips
- Troubleshooting с решениями

**Время на прочтение**: 15-20 минут  
**Для кого**: DevOps инженеры, системные администраторы

---

**[`TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md)** - Техническая документация
- Полная архитектура системы
- Как работает трейсинг
- Инициализация tracer для новых сервисов
- Пропагация контекста через NATS
- Переменные окружения
- Best practices
- Troubleshooting с диагностикой

**Время на прочтение**: 20-30 минут  
**Для кого**: Backend разработчики, архитекторы

---

### 📋 Справочные материалы

**[`TRACING_COMPLETE_SUMMARY.md`](TRACING_COMPLETE_SUMMARY.md)** - Финальный summary
- Что было проблемой
- Полное решение (все компоненты)
- Ожидаемая архитектура
- Результаты "до" и "после"
- Для дальнейшего улучшения

**Время на прочтение**: 10-15 минут  
**Для кого**: Менеджеры, техлиды, архитекторы

---

**[`TRACING_CHECKLIST.md`](TRACING_CHECKLIST.md)** - Чек-лист реализации
- Полный список что реализовано
- Проверки сборки всех сервисов
- Ожидаемое поведение
- Known issues и их статус
- Таблица "До/После"

**Время на прочтение**: 5 минут  
**Для кого**: QA инженеры, код-ревьюеры

---

**[`TRACING_FILES_REFERENCE.md`](TRACING_FILES_REFERENCE.md)** - Справочник по файлам
- Список новых файлов
- Список измененных файлов
- Что было добавлено/удалено в каждом файле
- Связанные компоненты
- Типовые паттерны

**Время на прочтение**: 5 минут  
**Для кого**: Разработчики, которые хотят понять что изменилось

---

## 📊 Таблица: Как выбрать документ?

| Вопрос | Ответ | Документ |
|--------|--------|----------|
| "Как быстро запустить?" | 5 минут | TRACING_QUICK_START.md |
| "Как настроить Jaeger?" | 15 минут | TRACING_SETUP.md |
| "Как это работает?" | 30 минут | TRACING_IMPLEMENTATION.md |
| "Что было изменено?" | 5 минут | TRACING_FILES_REFERENCE.md |
| "Всё ли готово?" | 5 минут | TRACING_CHECKLIST.md |
| "Что в итоге?" | 15 минут | TRACING_COMPLETE_SUMMARY.md |

---

## 🎯 Сценарии использования

### Сценарий 1: Я новичок в проекте, хочу разобраться

1. Прочитайте [`TRACING_QUICK_START.md`](TRACING_QUICK_START.md) - 5 минут
2. Запустите локально по инструкциям - 10 минут
3. Посмотрите traces в Jaeger UI - 5 минут
4. **Итого**: 20 минут до понимания что это работает ✅

### Сценарий 2: Я хочу добавить трейсинг в новый сервис

1. Прочитайте раздел "Инициализация Tracer" в [`TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md)
2. Скопируйте код инициализации из существующего сервиса
3. Адаптируйте под свой сервис
4. **Итого**: 10-15 минут

### Сценарий 3: Я хочу понять архитектуру

1. Посмотрите диаграмму в [`TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md)
2. Прочитайте раздел "Как работает трейсинг"
3. Посмотрите в [`TRACING_COMPLETE_SUMMARY.md`](TRACING_COMPLETE_SUMMARY.md) "Ожидаемая архитектура трейсинга"
4. **Итого**: 15-20 минут

### Сценарий 4: Что-то не работает

1. Посмотрите раздел "Troubleshooting" в [`TRACING_QUICK_START.md`](TRACING_QUICK_START.md)
2. Если не помогло, посмотрите расширенное troubleshooting в [`TRACING_SETUP.md`](TRACING_SETUP.md)
3. Если еще не помогло, посмотрите диагностику в [`TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md)
4. **Итого**: 10-30 минут до решения

---

## 🔗 Быстрые ссылки

### Команды

```bash
# Запустить Jaeger
docker run -d --name jaeger -p 6831:6831/udp -p 16686:16686 jaegertracing/all-in-one

# Включить трейсинг
export TELEMETRY_ENABLED=true

# Запустить сервис
go run cmd/{service}/main.go

# Посмотреть traces
open http://localhost:16686
```

### Файлы в проекте

```
libs/telemetry/
├── tracer.go       ← Единая инициализация tracer
└── go.mod         ← Зависимости

services/gateway/internal/app/app.go   ← Gateway трейсинг
services/auth/internal/server/auth_server.go  ← Auth трейсинг
services/users/internal/app/app.go     ← Users трейсинг
services/chat/internal/app/app.go      ← Chat трейсинг

md/
├── TRACING_QUICK_START.md      ← НАЧНИТЕ ОТСЮДА
├── TRACING_SETUP.md             ← Гайд по запуску
├── TRACING_IMPLEMENTATION.md    ← Техническая документация
├── TRACING_COMPLETE_SUMMARY.md  ← Финальный summary
├── TRACING_CHECKLIST.md         ← Чек-лист
└── TRACING_FILES_REFERENCE.md   ← Справочник
```

---

## 📞 Часто задаваемые вопросы

### Q: С какого документа начать?
**A**: С [`TRACING_QUICK_START.md`](TRACING_QUICK_START.md) - это займет максимум 5 минут

### Q: Нужен ли мне Docker?
**A**: Да, для запуска Jaeger нужен Docker (инструкции в [`TRACING_SETUP.md`](TRACING_SETUP.md))

### Q: Сможет ли мой ноутбук это потянуть?
**A**: Да, Jaeger требует ~200MB памяти, сервисы ~50-100MB каждый

### Q: Что если я не хочу трейсинг?
**A**: Просто установите `TELEMETRY_ENABLED=false` и перезапустите сервис

### Q: Можно ли использовать в production?
**A**: Да, но рекомендуется добавить sampling (см. [`TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md))

---

## ✅ Что было сделано

- ✅ Единая система инициализации tracer
- ✅ Трейсинг для всех 4 сервисов (Gateway, Auth, Users, Chat)
- ✅ Автоматическая пропагация trace context через NATS
- ✅ Graceful shutdown tracer provider
- ✅ Полная документация (6 файлов)
- ✅ Все сервисы собираются без ошибок

---

## 🚀 Что дальше?

1. **Запустите локально** (5 минут) - следуйте [`TRACING_QUICK_START.md`](TRACING_QUICK_START.md)
2. **Посмотрите traces в Jaeger** (5 минут)
3. **Прочитайте доп. документацию** если интересно (опционально)
4. **Добавьте свои spans** если нужно (см. [`TRACING_IMPLEMENTATION.md`](TRACING_IMPLEMENTATION.md))

---

**Версия**: 1.0  
**Статус**: ✅ Ready for Production  
**Последнее обновление**: 2025-02-02
