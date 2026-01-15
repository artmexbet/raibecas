# NATS Topics для Document Service

Этот документ описывает NATS-топики, используемые для взаимодействия Gateway с Document Service.

## Общая информация

- **Протокол**: Request-Reply pattern
- **Формат сообщений**: JSON
- **Timeout**: 5 секунд (по умолчанию)

## Топики

### 1. `documents.list`

**Описание**: Получение списка документов с фильтрацией и пагинацией

**Request**:
```json
{
  "page": 1,
  "limit": 20,
  "categoryId": 5,
  "authorId": "550e8400-e29b-41d4-a716-446655440000",
  "search": "search term",
  "sortBy": "title",
  "sortOrder": "asc"
}
```

**Request Fields**:
- `page` (int, optional): Номер страницы (по умолчанию: 1)
- `limit` (int, optional): Количество записей на странице (по умолчанию: 20, макс: 100)
- `categoryId` (int, optional): Фильтр по ID категории
- `authorId` (UUID, optional): Фильтр по ID автора
- `search` (string, optional): Поисковый запрос (макс: 200 символов)
- `sortBy` (string, optional): Поле для сортировки (`title`, `publicationDate`, `createdAt`)
- `sortOrder` (string, optional): Направление сортировки (`asc`, `desc`)

**Response**:
```json
{
  "documents": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Document Title",
      "description": "Document description",
      "author": {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "name": "Author Name"
      },
      "category": {
        "id": 1,
        "title": "Category Title"
      },
      "publicationDate": "2024-01-15T10:30:00Z",
      "tags": [
        {"id": 1, "title": "Tag 1"},
        {"id": 2, "title": "Tag 2"}
      ],
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-10T00:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "limit": 20,
  "totalPages": 5
}
```

---

### 2. `documents.get`

**Описание**: Получение одного документа по ID

**Request**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Document Title",
  "description": "Document description",
  "author": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Author Name"
  },
  "category": {
    "id": 1,
    "title": "Category Title"
  },
  "publicationDate": "2024-01-15T10:30:00Z",
  "tags": [
    {"id": 1, "title": "Tag 1"},
    {"id": 2, "title": "Tag 2"}
  ],
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-10T00:00:00Z"
}
```

**Error Response** (если документ не найден):
```json
{
  "error": "not_found",
  "message": "Document not found"
}
```

---

### 3. `documents.create`

**Описание**: Создание нового документа

**Request**:
```json
{
  "title": "New Document Title",
  "description": "Document description",
  "authorId": "550e8400-e29b-41d4-a716-446655440001",
  "categoryId": 1,
  "publicationDate": "2024-01-15T10:30:00Z",
  "tags": [1, 2, 3]
}
```

**Request Fields**:
- `title` (string, required): Название документа (1-500 символов)
- `description` (string, optional): Описание документа (макс: 2000 символов)
- `authorId` (UUID, required): ID автора
- `categoryId` (int, required): ID категории (мин: 1)
- `publicationDate` (timestamp, required): Дата публикации
- `tags` ([]int, optional): Массив ID тегов

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "New Document Title",
  "description": "Document description",
  "author": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "Author Name"
  },
  "category": {
    "id": 1,
    "title": "Category Title"
  },
  "publicationDate": "2024-01-15T10:30:00Z",
  "tags": [
    {"id": 1, "title": "Tag 1"},
    {"id": 2, "title": "Tag 2"}
  ],
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-01T00:00:00Z"
}
```

---

### 4. `documents.update`

**Описание**: Обновление существующего документа

**Request**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "updates": {
    "title": "Updated Title",
    "description": "Updated description",
    "authorId": "550e8400-e29b-41d4-a716-446655440002",
    "categoryId": 2,
    "publicationDate": "2024-02-15T10:30:00Z",
    "tags": [2, 3, 4]
  }
}
```

**Request Fields**:
- `id` (UUID, required): ID документа для обновления
- `updates` (object, required): Поля для обновления (все поля опциональны)
  - `title` (string, optional): Новое название (1-500 символов)
  - `description` (string, optional): Новое описание (макс: 2000 символов)
  - `authorId` (UUID, optional): Новый ID автора
  - `categoryId` (int, optional): Новый ID категории (мин: 1)
  - `publicationDate` (timestamp, optional): Новая дата публикации
  - `tags` ([]int, optional): Новый массив ID тегов

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Updated Title",
  "description": "Updated description",
  "author": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "name": "New Author Name"
  },
  "category": {
    "id": 2,
    "title": "New Category Title"
  },
  "publicationDate": "2024-02-15T10:30:00Z",
  "tags": [
    {"id": 2, "title": "Tag 2"},
    {"id": 3, "title": "Tag 3"}
  ],
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-15T12:00:00Z"
}
```

---

### 5. `documents.delete`

**Описание**: Удаление документа по ID

**Request**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response** (успешное удаление):
```json
{
  "success": true
}
```

**Error Response** (если документ не найден):
```json
{
  "error": "not_found",
  "message": "Document not found"
}
```

---

## Обработка ошибок

Все топики могут возвращать ошибки в следующем формате:

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field1": "validation error",
    "field2": "validation error"
  }
}
```

**Типы ошибок**:
- `bad_request` - Неверный формат запроса
- `validation_error` - Ошибка валидации данных
- `not_found` - Запрашиваемый ресурс не найден
- `internal_error` - Внутренняя ошибка сервера

