# Миграция на конфигурацию через переменные окружения

## Что изменилось

### До миграции

Конфигурация использовала `@dataclass` и загружалась только программно:

```python
from dataclasses import dataclass

@dataclass
class OllamaConfig:
    url: str = "http://127.0.0.1:11434"
    embedding_model: str = "embeddinggemma"
    timeout_seconds: float = 30.0
```

### После миграции

Конфигурация использует `pydantic-settings` и поддерживает загрузку из переменных окружения:

```python
from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict

class OllamaConfig(BaseSettings):
    """Configuration for Ollama service"""
    model_config = SettingsConfigDict(env_prefix='OLLAMA_')
    
    url: str = Field(default="http://127.0.0.1:11434", description="Ollama service URL")
    embedding_model: str = Field(default="embeddinggemma", description="Embedding model name")
    timeout_seconds: float = Field(default=30.0, description="Request timeout in seconds")
```

## Обратная совместимость

✅ **Изменения полностью обратно совместимы!**

Все существующие способы использования конфигурации продолжают работать:

```python
# Способ 1: Значения по умолчанию (как раньше)
config = AppConfig()

# Способ 2: Программное создание (как раньше)
config = AppConfig(
    ollama=OllamaConfig(url="http://custom:11434"),
    qdrant=QdrantConfig(host="custom-host", port=6334)
)

# Способ 3: НОВЫЙ - загрузка из ENV
# Установите переменные окружения, и они будут автоматически применены
# export OLLAMA__URL="http://custom:11434"
config = AppConfig()  # автоматически загрузит из ENV
```

## Преимущества новой конфигурации

1. **Гибкость**: Конфигурация теперь может быть установлена через переменные окружения
2. **Docker/Kubernetes friendly**: Легко настраивать через ConfigMaps и Secrets
3. **Валидация**: Pydantic автоматически валидирует типы данных
4. **Документация**: Каждое поле имеет описание
5. **Безопасность**: Можно использовать .env файлы (добавлены в .gitignore)
6. **Единообразие**: Соответствует подходу в Go-сервисах (cleanenv)

## Структура переменных окружения

Используются префиксы для группировки конфигурации:

- `OLLAMA__*` - настройки Ollama
- `QDRANT__*` - настройки Qdrant
- `CHUNK__*` - настройки чанкинга
- `NATS__*` - настройки NATS

Двойное подчеркивание `__` используется как разделитель для вложенных структур.

## Примеры использования

### Docker Compose

```yaml
services:
  index-python:
    build: ./services/index-python
    environment:
      - OLLAMA__URL=http://ollama:11434
      - QDRANT__HOST=qdrant
      - QDRANT__PORT=6333
      - CHUNK__CHUNK_SIZE=1000
      - NATS__SERVERS=["nats://nats:4222"]
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: index-python-config
data:
  OLLAMA__URL: "http://ollama-service:11434"
  QDRANT__HOST: "qdrant-service"
  QDRANT__PORT: "6333"
  CHUNK__CHUNK_SIZE: "1000"
---
apiVersion: v1
kind: Pod
metadata:
  name: index-python
spec:
  containers:
  - name: index-python
    image: index-python:latest
    envFrom:
    - configMapRef:
        name: index-python-config
```

### Локальная разработка с .env

Создайте файл `.env` в корне `services/index-python/`:

```env
OLLAMA__URL=http://localhost:11434
QDRANT__HOST=localhost
QDRANT__PORT=6333
CHUNK__CHUNK_SIZE=700
```

`.env` файл автоматически загружается библиотекой `pydantic-settings`.

## Что нужно сделать в вашем проекте

### 1. Установить зависимости

```bash
pip install -r requirements.txt
```

### 2. (Опционально) Создать .env файл

```bash
cp .env.example .env
# Отредактируйте .env под ваши нужды
```

### 3. Запустить приложение

```bash
python main.py
```

## Проверка конфигурации

Запустите пример для проверки:

```bash
python example_config.py
```

## Тестирование

Запустите тесты конфигурации:

```bash
python test_config.py
```

## Поддержка

Если у вас возникли проблемы после миграции:

1. Проверьте, что `pydantic-settings` установлен: `pip show pydantic-settings`
2. Убедитесь, что переменные окружения установлены правильно
3. Проверьте синтаксис префиксов (используйте двойное подчеркивание `__`)
4. Запустите `example_config.py` для диагностики

## Дополнительная информация

- [Pydantic Settings Documentation](https://docs.pydantic.dev/latest/concepts/pydantic_settings/)
- [Pydantic Field Documentation](https://docs.pydantic.dev/latest/concepts/fields/)

