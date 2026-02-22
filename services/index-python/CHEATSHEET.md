# Конфигурация через переменные окружения - Шпаргалка

## Быстрый старт

### 1. Установка
```bash
pip install -r requirements.txt
```

### 2. Создание .env файла
```bash
cp .env.example .env
```

### 3. Запуск
```bash
python main.py
```

## Переменные окружения

### Ollama (префикс: `OLLAMA__`)
| Переменная | Тип | По умолчанию | Описание |
|-----------|-----|--------------|----------|
| `OLLAMA__URL` | string | `http://127.0.0.1:11434` | URL сервиса Ollama |
| `OLLAMA__EMBEDDING_MODEL` | string | `embeddinggemma` | Модель для эмбеддингов |
| `OLLAMA__TIMEOUT_SECONDS` | float | `30.0` | Таймаут запросов (сек) |

### Qdrant (префикс: `QDRANT__`)
| Переменная | Тип | По умолчанию | Описание |
|-----------|-----|--------------|----------|
| `QDRANT__HOST` | string | `localhost` | Хост Qdrant |
| `QDRANT__PORT` | int | `6333` | Порт Qdrant |
| `QDRANT__COLLECTION_NAME` | string | `documents` | Имя коллекции |
| `QDRANT__DISTANCE` | string | `Cosine` | Метрика расстояния |
| `QDRANT__VECTOR_SIZE` | int | `768` | Размерность векторов |

### Chunking (префикс: `CHUNK__`)
| Переменная | Тип | По умолчанию | Описание |
|-----------|-----|--------------|----------|
| `CHUNK__CHUNK_SIZE` | int | `700` | Размер чанка |
| `CHUNK__CHUNK_OVERLAP` | int | `80` | Перекрытие между чанками |
| `CHUNK__MAX_CHUNKS` | int | `0` | Макс. кол-во чанков (0=∞) |

### NATS (префикс: `NATS__`)
| Переменная | Тип | По умолчанию | Описание |
|-----------|-----|--------------|----------|
| `NATS__SERVERS` | list | `["nats://127.0.0.1:4222"]` | Список серверов NATS |
| `NATS__NAME` | string | `None` | Имя клиента |
| `NATS__ALLOW_RECONNECT` | bool | `true` | Разрешить переподключение |
| `NATS__RECONNECT_TIME_WAIT` | float | `2.0` | Время ожидания (сек) |
| `NATS__MAX_RECONNECT_ATTEMPTS` | int | `-1` | Макс. попыток (-1=∞) |
| `NATS__CONNECT_TIMEOUT` | float | `5.0` | Таймаут подключения (сек) |
| `NATS__REQUEST_TIMEOUT` | float | `5.0` | Таймаут запроса (сек) |
| `NATS__PING_INTERVAL` | float | `10.0` | Интервал пингов (сек) |
| `NATS__DRAIN_TIMEOUT` | float | `5.0` | Таймаут завершения (сек) |

## Примеры использования

### PowerShell
```powershell
$env:OLLAMA__URL="http://192.168.1.100:11434"
$env:QDRANT__HOST="192.168.1.101"
python main.py
```

### Bash
```bash
export OLLAMA__URL="http://192.168.1.100:11434"
export QDRANT__HOST="192.168.1.101"
python main.py
```

### Docker Compose
```yaml
environment:
  - OLLAMA__URL=http://ollama:11434
  - QDRANT__HOST=qdrant
  - CHUNK__CHUNK_SIZE=1000
```

### .env файл
```env
OLLAMA__URL=http://localhost:11434
QDRANT__HOST=localhost
CHUNK__CHUNK_SIZE=1000
```

## Программное использование

### Значения по умолчанию
```python
from pipeline.config import AppConfig
config = AppConfig()  # Использует значения по умолчанию
```

### Из переменных окружения
```python
import os
os.environ["OLLAMA__URL"] = "http://custom:11434"

from pipeline.config import AppConfig
config = AppConfig()  # Автоматически загружает из ENV
```

### Программная настройка
```python
from pipeline.config import AppConfig, OllamaConfig

config = AppConfig(
    ollama=OllamaConfig(url="http://custom:11434")
)
```

### Смешанный подход
```python
# ENV переменные + программная настройка
import os
os.environ["OLLAMA__URL"] = "http://custom:11434"

from pipeline.config import AppConfig, QdrantConfig

config = AppConfig(
    qdrant=QdrantConfig(host="custom-qdrant", port=6334)
)
# ollama загружен из ENV, qdrant - программно
```

## Проверка конфигурации

```bash
# Запустить пример
python example_config.py

# Запустить тесты
python test_config.py
```

## Важные заметки

1. ⚠️ Используйте **двойное подчеркивание** `__` в префиксах
2. 📝 `.env` файл автоматически загружается (не коммитьте его!)
3. 🔄 ENV переменные имеют приоритет над значениями по умолчанию
4. ✅ Все изменения обратно совместимы с предыдущей версией

## Дополнительные ресурсы

- 📄 [README.md](README.md) - Полная документация
- 🔄 [MIGRATION.md](MIGRATION.md) - Руководство по миграции
- 📋 [.env.example](.env.example) - Пример файла конфигурации

