# Index Service - Quick Reference

## What's New ✨

Полностью переработанная архитектура с разделением хранения файлов и индексации:

### Ключевые изменения

1. **Storage Layer** - файлы сохраняются в постоянном хранилище
2. **Event-Driven** - через NATS передаются только события с путями к файлам
3. **Buffered Processing** - документы читаются через буфер, не загружаются в память полностью
4. **File Upload API** - HTTP endpoint для загрузки файлов через multipart/form-data

### Архитектура

```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌─────────┐
│  Client │────→│   HTTP  │────→│ Storage  │────→│  NATS   │
└─────────┘     └─────────┘     └──────────┘     └─────────┘
                                      │                │
                                      ↓                ↓
                                ┌──────────┐     ┌──────────┐
                                │   File   │     │  Event   │
                                │  System  │     │  Queue   │
                                └──────────┘     └──────────┘
                                      │                │
                                      └────────┬───────┘
                                               ↓
                                        ┌──────────┐
                                        │ Pipeline │
                                        └──────────┘
                                               ↓
                                        ┌──────────┐
                                        │  Qdrant  │
                                        └──────────┘
```

## Quick Start

```bash
# 1. Start dependencies
docker-compose up -d

# 2. Run service
go run cmd/index/main.go

# 3. Upload document
curl -X POST http://localhost:8082/api/v1/index \
  -F "file=@mydoc.txt" \
  -F "id=doc-1" \
  -F "title=My Document"
```

## Testing

```bash
# All tests
go test ./... -v

# With coverage
go test ./... -cover

# Coverage: 70%+ for core components
```

## Documentation

- **[README.md](README.md)** - полная документация
- **[USAGE_EXAMPLES.md](USAGE_EXAMPLES.md)** - примеры использования
- **[IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)** - описание реализации

## Features

✅ File-based document storage  
✅ HTTP multipart file upload  
✅ NATS event processing  
✅ Buffered file reading  
✅ Path traversal protection  
✅ Comprehensive tests (22 test cases)  
✅ Docker support  
✅ Production-ready  

## Tech Stack

- Go 1.23
- Fiber (HTTP framework)
- NATS JetStream (messaging)
- Qdrant (vector database)
- Ollama (embeddings)

