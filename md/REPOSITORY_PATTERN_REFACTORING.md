# Рефакторинг: Repository Pattern

## Проблема
Service слой зависел от конкретной реализации БД (`queries.Queries`) и типов PostgreSQL (`pgtype.*`), что нарушает принципы чистой архитектуры.

## Решение
Внедрен **Repository Pattern** - промежуточный слой между бизнес-логикой и источником данных.

## Новая архитектура

```
┌─────────────────────────────────────┐
│  Handler                            │
│  depends on: DocumentService        │ ← Interface
└─────────────┬───────────────────────┘
              │
┌─────────────▼───────────────────────┐
│  Service (Business Logic)           │
│  depends on:                        │
│    - DocumentRepository             │ ← Interface
│    - VersionRepository              │ ← Interface  
│    - TagRepository                  │ ← Interface
│    - Storage                        │ ← Interface
│    - EventPublisher                 │ ← Interface
└─────────────┬───────────────────────┘
              │
        ┌─────┴──────┐
        │            │
┌───────▼──────┐ ┌──▼────────┐
│  PostgreSQL  │ │   MinIO   │
│  Repository  │ │  Storage  │
│ (implements) │ │(implements)│
└──────────────┘ └───────────┘
```

## Созданные файлы

### 1. `internal/repository/repository.go`
Определение интерфейсов репозиториев:
```go
type DocumentRepository interface {
    Create(ctx, *domain.Document) error
    GetByID(ctx, uuid.UUID) (*domain.Document, error)
    List(ctx, domain.ListDocumentsParams) ([]domain.Document, error)
    Count(ctx, domain.ListDocumentsParams) (int, error)
    Update(ctx, *domain.Document) error
    Delete(ctx, uuid.UUID) error
    UpdateIndexedStatus(ctx, uuid.UUID, bool) error
}
```

### 2. `internal/postgres/document_repository.go`
Реализация `DocumentRepository` для PostgreSQL:
- Преобразование domain ↔ queries types
- Работа с nullable типами pgtype
- Изоляция деталей БД от бизнес-логики

### 3. `internal/postgres/version_repository.go`
Реализация `VersionRepository`:
- Управление версиями документов
- Изолированная работа с БД

### 4. `internal/postgres/tag_repository.go`
Реализация `TagRepository`:
- Управление тегами документов
- CRUD операции для связей

## Преимущества

### ✅ Независимость от БД
**До:**
```go
// Service зависел от queries types
doc, err := s.queries.CreateDocument(ctx, queries.CreateDocumentParams{
    Title:       req.Title,
    CategoryID:  int32(req.CategoryID), // Конверсия типов
    // ...
})
```

**После:**
```go
// Service работает с domain types
doc := &domain.Document{
    Title:      req.Title,
    CategoryID: req.CategoryID, // Чистые типы
    // ...
}
err := s.docRepo.Create(ctx, doc)
```

### ✅ Тестируемость
Теперь легко создать mock для тестирования service:
```go
type MockDocumentRepo struct{}

func (m *MockDocumentRepo) Create(ctx context.Context, doc *domain.Document) error {
    // Mock implementation
    return nil
}
```

### ✅ Замена БД
Можно легко переключиться на другую БД, реализовав интерфейсы:
- MongoDB Repository
- DynamoDB Repository
- In-Memory Repository (для тестов)

### ✅ Единственная ответственность
- **Service**: бизнес-логика
- **Repository**: доступ к данным  
- **Domain**: модели предметной области

### ✅ Чистая архитектура (Clean Architecture)
```
Domain (entities) ← Service (use cases) ← Repository (gateways)
     ↑                    ↑                        ↓
     └────────────────────┴──────────────── PostgreSQL
```

## Изменения в Service

### Зависимости
**До:**
```go
type DocumentService struct {
    queries   *queries.Queries          // Конкретная реализация
    storage   storage.Storage
    publisher *nats.Publisher
}
```

**После:**
```go
type DocumentService struct {
    docRepo     DocumentRepository      // Интерфейс
    versionRepo VersionRepository       // Интерфейс
    tagRepo     TagRepository           // Интерфейс
    storage     Storage                 // Интерфейс
    publisher   EventPublisher          // Интерфейс
}
```

### Работа с данными
**До:**
```go
// Конверсия в queries types внутри service
doc, err := s.queries.CreateDocument(ctx, queries.CreateDocumentParams{
    Title:           req.Title,
    CategoryID:      int32(req.CategoryID), // ❌ Зависимость от БД типов
    PublicationDate: queries.NullDate(&req.PublicationDate),
})
```

**После:**
```go
// Работа с domain types
doc := &domain.Document{
    Title:           req.Title,
    CategoryID:      req.CategoryID, // ✅ Чистые domain типы
    PublicationDate: req.PublicationDate,
}
err := s.docRepo.Create(ctx, doc)
```

## Компиляция
✅ Проект успешно компилируется без ошибок

## Выводы

Теперь архитектура соответствует принципам:
- **SOLID** (особенно Dependency Inversion)
- **Clean Architecture** (слои зависят от абстракций)
- **Hexagonal Architecture** (порты и адаптеры)

Service слой полностью изолирован от деталей реализации БД и может быть легко протестирован и переиспользован.
