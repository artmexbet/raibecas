# Index Python Service

Python-сервис для индексирования документов с использованием векторных представлений.

## Конфигурация

Сервис поддерживает загрузку конфигурации из переменных окружения с использованием `pydantic-settings`.

### Способы конфигурирования

1. **Переменные окружения** - установите переменные окружения с соответствующими префиксами
2. **Файл .env** - создайте файл `.env` в корне проекта (см. `.env.example`)
3. **Значения по умолчанию** - если переменные не заданы, используются значения по умолчанию

### Структура конфигурации

#### Ollama (префикс: `OLLAMA__`)
- `OLLAMA__URL` - URL сервиса Ollama (по умолчанию: `http://127.0.0.1:11434`)
- `OLLAMA__EMBEDDING_MODEL` - модель для генерации эмбеддингов (по умолчанию: `embeddinggemma`)
- `OLLAMA__TIMEOUT_SECONDS` - таймаут запросов в секундах (по умолчанию: `30.0`)

#### Qdrant (префикс: `QDRANT__`)
- `QDRANT__HOST` - хост Qdrant (по умолчанию: `localhost`)
- `QDRANT__PORT` - порт Qdrant (по умолчанию: `6333`)
- `QDRANT__COLLECTION_NAME` - имя коллекции (по умолчанию: `documents`)
- `QDRANT__DISTANCE` - метрика расстояния (по умолчанию: `Cosine`)
- `QDRANT__VECTOR_SIZE` - размерность векторов (по умолчанию: `768`)

#### Chunking (префикс: `CHUNK__`)
- `CHUNK__CHUNK_SIZE` - размер чанка (по умолчанию: `700`)
- `CHUNK__CHUNK_OVERLAP` - перекрытие между чанками (по умолчанию: `80`)
- `CHUNK__MAX_CHUNKS` - максимальное количество чанков, 0 = без ограничений (по умолчанию: `0`)

#### NATS (префикс: `NATS__`)
- `NATS__SERVERS` - список серверов NATS (по умолчанию: `["nats://127.0.0.1:4222"]`)
- `NATS__NAME` - имя клиента (по умолчанию: `None`)
- `NATS__ALLOW_RECONNECT` - разрешить переподключение (по умолчанию: `true`)
- `NATS__RECONNECT_TIME_WAIT` - время ожидания переподключения в секундах (по умолчанию: `2.0`)
- `NATS__MAX_RECONNECT_ATTEMPTS` - максимальное количество попыток переподключения, -1 = бесконечно (по умолчанию: `-1`)
- `NATS__CONNECT_TIMEOUT` - таймаут подключения в секундах (по умолчанию: `5.0`)
- `NATS__REQUEST_TIMEOUT` - таймаут запроса в секундах (по умолчанию: `5.0`)
- `NATS__PING_INTERVAL` - интервал пингов в секундах (по умолчанию: `10.0`)
- `NATS__DRAIN_TIMEOUT` - таймаут завершения в секундах (по умолчанию: `5.0`)

### Примеры использования

#### Использование .env файла

Создайте файл `.env` на основе `.env.example`:

```bash
cp .env.example .env
```

Отредактируйте `.env` под свои нужды и запустите сервис:

```bash
python main.py
```

#### Использование переменных окружения (PowerShell)

```powershell
$env:OLLAMA__URL="http://192.168.1.100:11434"
$env:QDRANT__HOST="192.168.1.101"
$env:QDRANT__PORT="6333"
python main.py
```

#### Использование переменных окружения (bash)

```bash
export OLLAMA__URL="http://192.168.1.100:11434"
export QDRANT__HOST="192.168.1.101"
export QDRANT__PORT="6333"
python main.py
```

#### Программное конфигурирование

```python
from pipeline.config import AppConfig, OllamaConfig, QdrantConfig, ChunkConfig
from broker.nats_connector import NATSConfig

# Кастомная конфигурация
custom_config = AppConfig(
    ollama=OllamaConfig(url="http://custom-ollama:11434"),
    qdrant=QdrantConfig(host="custom-qdrant", port=6334),
    chunk=ChunkConfig(chunk_size=1000, chunk_overlap=100)
)

# Конфигурация NATS
nats_config = NATSConfig(
    servers=["nats://nats-1:4222", "nats://nats-2:4222"],
    name="my-service"
)

# Использование
from app import App
app = App(config=custom_config, nats_cfg=nats_config)
```

## Установка

```bash
pip install -r requirements.txt
```

## Запуск

```bash
python main.py
```

## Зависимости

- `pydantic>=2.0.0` - валидация данных
- `pydantic-settings>=2.0.0` - загрузка конфигурации из переменных окружения
- `nats-py>=2.6.0` - клиент NATS
- `ollama>=0.1.0` - клиент Ollama
- `qdrant-client>=1.7.0` - клиент Qdrant

