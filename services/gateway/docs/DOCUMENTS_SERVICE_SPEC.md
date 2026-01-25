# Documents Service - Спецификация NATS топиков

## Описание

Documents Service отвечает за управление документами (научные статьи, публикации). Сервис подписывается на NATS топики и обрабатывает запросы от Gateway.

## NATS Topics

### 1. documents.list
**Описание:** Получение списка документов с фильтрацией и пагинацией

**Request:**
```json
{
  "page": 1,
  "page_size": 10,
  "search": "machine learning",
  "category_id": 5,
  "author_id": "uuid",
  "tags": [1, 2, 3],
  "publication_date_from": "2023-01-01T00:00:00Z",
  "publication_date_to": "2024-01-01T00:00:00Z"
}
```

**Response:**
```json
{
  "documents": [
    {
      "id": "uuid",
      "title": "Introduction to Machine Learning",
      "description": "A comprehensive guide to ML",
      "author": {
        "id": "uuid",
        "name": "John Doe"
      },
      "category": {
        "id": 5,
        "title": "Computer Science"
      },
      "publication_date": "2024-01-01T00:00:00Z",
      "tags": [
        {"id": 1, "title": "AI"},
        {"id": 2, "title": "ML"}
      ],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total_count": 100,
  "page": 1,
  "page_size": 10
}
```

**Обработчик должен:**
- Парсить все параметры фильтрации
- Применять полнотекстовый поиск по title и description (если search указан)
- Фильтровать по category_id, author_id
- Фильтровать по tags (документы, содержащие хотя бы один из указанных тегов)
- Фильтровать по диапазону дат публикации
- Реализовать пагинацию с LIMIT и OFFSET
- Подгружать связанные данные (author, category, tags) через JOINs
- Возвращать общее количество документов (total_count)

**SQL пример:**
```sql
SELECT 
    d.*,
    json_build_object('id', a.id, 'name', a.name) as author,
    json_build_object('id', c.id, 'title', c.title) as category,
    COALESCE(json_agg(DISTINCT json_build_object('id', t.id, 'title', t.title)), '[]') as tags
FROM documents d
LEFT JOIN authors a ON d.author_id = a.id
LEFT JOIN categories c ON d.category_id = c.id
LEFT JOIN document_tags dt ON d.id = dt.document_id
LEFT JOIN tags t ON dt.tag_id = t.id
WHERE 
    ($1::text IS NULL OR d.title ILIKE '%' || $1 || '%' OR d.description ILIKE '%' || $1 || '%')
    AND ($2::int IS NULL OR d.category_id = $2)
    AND ($3::uuid IS NULL OR d.author_id = $3)
GROUP BY d.id, a.id, a.name, c.id, c.title
ORDER BY d.publication_date DESC
LIMIT $4 OFFSET $5;
```

---

### 2. documents.get
**Описание:** Получение полной информации о документе по ID

**Request:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "Introduction to Machine Learning",
  "description": "A comprehensive guide to ML",
  "author": {
    "id": "uuid",
    "name": "John Doe"
  },
  "category": {
    "id": 5,
    "title": "Computer Science"
  },
  "publication_date": "2024-01-01T00:00:00Z",
  "tags": [
    {"id": 1, "title": "AI"},
    {"id": 2, "title": "ML"}
  ],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Обработчик должен:**
- Валидировать UUID
- Искать документ в БД по ID
- Подгружать все связанные данные (author, category, tags)
- Вернуть ошибку "not_found" если документ не существует
- Опционально: увеличить счетчик просмотров документа

---

### 3. documents.create
**Описание:** Создание нового документа

**Request:**
```json
{
  "title": "New Research Paper",
  "description": "Abstract of the paper",
  "author_id": "uuid",
  "category_id": 5,
  "publication_date": "2024-01-01T00:00:00Z",
  "tags": [1, 2, 3]
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "New Research Paper",
  "description": "Abstract of the paper",
  "author": {
    "id": "uuid",
    "name": "John Doe"
  },
  "category": {
    "id": 5,
    "title": "Computer Science"
  },
  "publication_date": "2024-01-01T00:00:00Z",
  "tags": [
    {"id": 1, "title": "AI"},
    {"id": 2, "title": "ML"},
    {"id": 3, "title": "Deep Learning"}
  ],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Обработчик должен:**
1. Валидировать входные данные:
   - title: required, max 500 symbols
   - description: optional, max 5000 symbols
   - author_id: required, must exist
   - category_id: required, must exist
   - publication_date: required
   - tags: optional array of existing tag IDs
2. Проверить существование author_id и category_id
3. Создать документ в таблице documents
4. Создать связи в таблице document_tags (многие-ко-многим)
5. Вернуть созданный документ со всеми связанными данными
6. Опционально: отправить событие document.created

**SQL транзакция:**
```sql
BEGIN;
-- Создать документ
INSERT INTO documents (title, description, author_id, category_id, publication_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- Создать связи с тегами
INSERT INTO document_tags (document_id, tag_id)
SELECT $doc_id, unnest($tags::int[]);
COMMIT;
```

---

### 4. documents.update
**Описание:** Обновление существующего документа

**Request:**
```json
{
  "id": "uuid",
  "updates": {
    "title": "Updated Title",
    "description": "Updated description",
    "category_id": 6,
    "tags": [2, 3, 4]
  }
}
```

**Response:**
```json
{
  "id": "uuid",
  "title": "Updated Title",
  "description": "Updated description",
  "author": {
    "id": "uuid",
    "name": "John Doe"
  },
  "category": {
    "id": 6,
    "title": "Mathematics"
  },
  "publication_date": "2024-01-01T00:00:00Z",
  "tags": [
    {"id": 2, "title": "ML"},
    {"id": 3, "title": "Deep Learning"},
    {"id": 4, "title": "Neural Networks"}
  ],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-10T00:00:00Z"
}
```

**Обработчик должен:**
1. Проверить существование документа
2. Валидировать переданные поля
3. Обновить только переданные поля (partial update)
4. Если переданы tags - пересоздать все связи:
   - Удалить старые связи из document_tags
   - Создать новые связи
5. Обновить поле updated_at
6. Вернуть обновленный документ
7. Опционально: отправить событие document.updated

**Важно:** author_id и publication_date обычно нельзя изменять после создания

---

### 5. documents.delete
**Описание:** Удаление документа

**Request:**
```json
{
  "id": "uuid"
}
```

**Response:**
```json
{
  "success": true
}
```

**Обработчик должен:**
1. Проверить существование документа
2. Удалить связи из document_tags (CASCADE или вручную)
3. Удалить документ из documents
4. Опционально: отправить событие document.deleted
5. Опционально: удалить связанные файлы (PDF, изображения)

**SQL с CASCADE:**
```sql
-- При создании таблицы
CREATE TABLE document_tags (
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id),
    PRIMARY KEY (document_id, tag_id)
);

-- Удаление
DELETE FROM documents WHERE id = $1;
-- Связи удалятся автоматически
```

---

## Вспомогательные структуры данных

### Таблица documents
```sql
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    author_id UUID NOT NULL REFERENCES authors(id),
    category_id INT NOT NULL REFERENCES categories(id),
    publication_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_author ON documents(author_id);
CREATE INDEX idx_documents_category ON documents(category_id);
CREATE INDEX idx_documents_publication_date ON documents(publication_date);
CREATE INDEX idx_documents_title ON documents USING gin(to_tsvector('english', title));
CREATE INDEX idx_documents_description ON documents USING gin(to_tsvector('english', description));
```

### Таблица authors
```sql
CREATE TABLE authors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Таблица categories
```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    title VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Таблица tags
```sql
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    title VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Таблица document_tags (многие-ко-многим)
```sql
CREATE TABLE document_tags (
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (document_id, tag_id)
);

CREATE INDEX idx_document_tags_tag ON document_tags(tag_id);
```

---

## Общий формат ошибок

```json
{
  "error": "error_code",
  "message": "Human readable message"
}
```

**Коды ошибок:**
- `invalid_request` - неверный формат запроса
- `validation_failed` - провалена валидация
- `not_found` - документ не найден
- `author_not_found` - автор не найден
- `category_not_found` - категория не найдена
- `tag_not_found` - тег не найден
- `internal_error` - внутренняя ошибка

---

## Примеры реализации (Go)

### Handler структура
```go
type DocumentHandler struct {
    docService DocumentService
    publisher  EventPublisher
}

func (h *DocumentHandler) HandleListDocuments(msg *natsw.Message) error {
    var req ListDocumentsQuery
    if err := msg.UnmarshalData(&req); err != nil {
        return h.respondError(msg, domain.ErrorResponse{
            Error:   "invalid_request",
            Message: "Invalid request format",
        })
    }
    
    docs, total, err := h.docService.ListDocuments(msg.Ctx, req)
    if err != nil {
        return h.respondError(msg, domain.ErrorResponse{
            Error:   "internal_error",
            Message: err.Error(),
        })
    }
    
    response := ListDocumentsResponse{
        Documents:  docs,
        TotalCount: total,
        Page:       req.Page,
        PageSize:   req.PageSize,
    }
    
    return h.respond(msg, response)
}

func (h *DocumentHandler) HandleCreateDocument(msg *natsw.Message) error {
    var req CreateDocumentRequest
    if err := msg.UnmarshalData(&req); err != nil {
        return h.respondError(msg, domain.ErrorResponse{
            Error:   "invalid_request",
            Message: "Invalid request format",
        })
    }
    
    // Validate
    if err := validate.Struct(req); err != nil {
        return h.respondError(msg, domain.ErrorResponse{
            Error:   "validation_failed",
            Message: err.Error(),
        })
    }
    
    doc, err := h.docService.CreateDocument(msg.Ctx, req)
    if err != nil {
        if errors.Is(err, ErrAuthorNotFound) {
            return h.respondError(msg, domain.ErrorResponse{
                Error:   "author_not_found",
                Message: "Author does not exist",
            })
        }
        return h.respondError(msg, domain.ErrorResponse{
            Error:   "internal_error",
            Message: err.Error(),
        })
    }
    
    // Publish event
    h.publisher.PublishDocumentCreated(msg.Ctx, DocumentCreatedEvent{
        DocumentID: doc.ID,
        Title:      doc.Title,
        AuthorID:   doc.Author.ID,
        Timestamp:  time.Now(),
    })
    
    return h.respond(msg, doc)
}
```

### Регистрация подписок
```go
func (s *Server) setupSubscriptions() error {
    s.nc.Subscribe("documents.list", s.docHandler.HandleListDocuments)
    s.nc.Subscribe("documents.get", s.docHandler.HandleGetDocument)
    s.nc.Subscribe("documents.create", s.docHandler.HandleCreateDocument)
    s.nc.Subscribe("documents.update", s.docHandler.HandleUpdateDocument)
    s.nc.Subscribe("documents.delete", s.docHandler.HandleDeleteDocument)
    
    return nil
}
```

---

## События (опционально)

Documents Service может публиковать события:

- `documents.document.created` - новый документ создан
- `documents.document.updated` - документ обновлен
- `documents.document.deleted` - документ удален

**Пример события:**
```json
{
  "event_type": "document.created",
  "document_id": "uuid",
  "title": "New Research Paper",
  "author_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

---

## Полнотекстовый поиск

Для эффективного поиска по title и description рекомендуется использовать PostgreSQL Full-Text Search:

```sql
-- Создание индексов
CREATE INDEX idx_documents_search ON documents 
USING gin(to_tsvector('english', title || ' ' || COALESCE(description, '')));

-- Поиск
SELECT * FROM documents
WHERE to_tsvector('english', title || ' ' || COALESCE(description, '')) 
      @@ plainto_tsquery('english', $1)
ORDER BY ts_rank(to_tsvector('english', title || ' ' || COALESCE(description, '')), 
                  plainto_tsquery('english', $1)) DESC;
```

---

## Безопасность

1. **Валидация** - проверять все входные данные
2. **SQL Injection** - использовать prepared statements
3. **Существование связей** - проверять author_id, category_id перед вставкой
4. **Лимиты** - ограничивать page_size (max 100)
5. **Права доступа** - проверять, может ли пользователь редактировать/удалять документ

---

## Оптимизация

1. **Индексы** - создать индексы на author_id, category_id, publication_date
2. **JOIN оптимизация** - использовать LEFT JOIN для связанных данных
3. **Кэширование** - кэшировать categories и tags (редко меняются)
4. **Пагинация** - всегда использовать LIMIT и OFFSET
5. **COUNT оптимизация** - использовать отдельный запрос для подсчета или window functions

---

## Метрики

Рекомендуется отслеживать:
- Количество документов по категориям
- Популярные теги
- Время обработки поисковых запросов
- Количество создаваемых документов в день
- Самые просматриваемые документы
