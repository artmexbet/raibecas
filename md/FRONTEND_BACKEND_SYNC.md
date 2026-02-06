# Синхронизация типов данных Frontend ↔ Gateway

## Обзор

Типы данных документов в frontend синхронизированы с backend типами из gateway для обеспечения типобезопасности и корректного взаимодействия.

## Маппинг типов

### Backend → Frontend

| Backend (Go) | Frontend (TypeScript) | Описание |
|--------------|----------------------|----------|
| `uuid.UUID` | `string` | UUID в формате строки |
| `*string` | `string \| null \| undefined` | Опциональная строка |
| `time.Time` | `string` | ISO 8601 timestamp |
| `[]Tag` | `Tag[]` | Массив тегов |

## Структуры данных

### Document

**Backend (`domain.Document`):**
```go
type Document struct {
    ID          uuid.UUID
    Title       string
    Description *string
    Author      Author
    Category    Category
    PublicationDate time.Time
    Tags        []Tag
    Content     *string      // опционально, загружается отдельным запросом
    Additional  Additional   // содержит created_at, updated_at
}
```

**Frontend (`Document`):**
```typescript
interface Document {
    id: string;
    title: string;
    description?: string | null;
    author: Author;
    category: Category;
    publication_date: string; // ISO 8601
    tags: Tag[];
    content?: string | null;   // Markdown content
    created_at: string; // ISO 8601
    updated_at: string; // ISO 8601
}
```

### Author

**Backend:**
```go
type Author struct {
    ID   uuid.UUID
    Name string
}
```

**Frontend:**
```typescript
interface Author {
    id: string;
    name: string;
}
```

### Category

**Backend:**
```go
type Category struct {
    ID    int
    Title string
}
```

**Frontend:**
```typescript
interface Category {
    id: number;
    title: string;
}
```

### Tag

**Backend:**
```go
type Tag struct {
    ID    int
    Title string
}
```

**Frontend:**
```typescript
interface Tag {
    id: number;
    title: string;
}
```

## Request/Response типы

### CreateDocumentRequest

**Backend (`domain.CreateDocumentRequest`):**
```go
type CreateDocumentRequest struct {
    Title           string
    Description     *string
    AuthorID        uuid.UUID
    CategoryID      int
    PublicationDate time.Time
    TagIDs          []int
}
```

**Frontend:**
```typescript
interface CreateDocumentRequest {
    title: string;
    description?: string | null;
    authorId: string;
    categoryId: number;
    publicationDate: string; // ISO 8601
    tagIds?: number[];
}
```

### UpdateDocumentRequest

**Backend (`domain.UpdateDocumentRequest`):**
```go
type UpdateDocumentRequest struct {
    Title           *string
    Description     *string
    AuthorID        *uuid.UUID
    CategoryID      *int
    PublicationDate *time.Time
    TagIDs          []int
}
```

**Frontend:**
```typescript
interface UpdateDocumentRequest {
    title?: string;
    description?: string | null;
    authorId?: string;
    categoryId?: number;
    publicationDate?: string; // ISO 8601
    tagIds?: number[];
}
```

### ListDocumentsQuery

**Backend (`domain.ListDocumentsQuery`):**
```go
type ListDocumentsQuery struct {
    Page       int
    Limit      int
    AuthorID   uuid.UUID
    CategoryID int
    TagID      int
    Search     string
}
```

**Frontend:**
```typescript
interface ListDocumentsQuery {
    page?: number;
    limit?: number;
    authorId?: string;
    categoryId?: number;
    tagId?: number;
    search?: string;
}
```

### ListDocumentsResponse

**Backend (`domain.ListDocumentsResponse`):**
```go
type ListDocumentsResponse struct {
    Documents  []Document
    Total      int
    Page       int
    Limit      int
    TotalPages int
}
```

**Frontend:**
```typescript
interface ListDocumentsResponse {
    documents: Document[];
    total: number;
    page: number;
    limit: number;
    totalPages: number;
}
```

## Изменения в API сервисе

### До синхронизации

```typescript
// Прямой возврат Document[]
async getAll(): Promise<Document[]>

// Прямой возврат Document
async getById(id: string): Promise<Document>
```

### После синхронизации

```typescript
// Возврат структурированного ответа
async getAll(query?: ListDocumentsQuery): Promise<ListDocumentsResponse>

// Разворачиваем document из response
async getById(id: string): Promise<Document>
```

## Ключевые различия

### 1. Именование полей

- **Backend**: `snake_case` в JSON тегах (`publication_date`, `created_at`)
- **Frontend**: соответствует JSON (`publication_date`, `created_at`)

### 2. Опциональные поля

- **Backend**: `*string` для nullable полей
- **Frontend**: `string | null | undefined`

### 3. Timestamp формат

- **Backend**: `time.Time` → сериализуется в ISO 8601
- **Frontend**: `string` с комментарием о формате

### 4. UUID

- **Backend**: `uuid.UUID` тип
- **Frontend**: `string` (UUID в строковом формате)

## Обновлённые файлы

### Типы
- ✅ `frontend/apps/admin-panel/src/types/document.ts`
  - Добавлены недостающие Response типы
  - Изменены имена полей: `publicationDate` → `publication_date`
  - Изменены имена полей: `createdAt` → `created_at`, `updatedAt` → `updated_at`
  - Удалены поля `content`, `views`, `notesCount` (не в domain.Document)

### Сервисы
- ✅ `frontend/apps/admin-panel/src/services/document.service.ts`
  - Обновлены сигнатуры методов
  - Используются типизированные Request/Response
  - Добавлена поддержка `ListDocumentsQuery`

### Mock данные
- ✅ `frontend/apps/admin-panel/src/mocks/data/documents.ts`
  - Обновлены имена полей в mock документах
  - Удалены несуществующие поля

### Mock handlers
- ✅ `frontend/apps/admin-panel/src/mocks/handlers/documents.ts`
  - Обновлены имена полей в логике обработчиков
  - Удалены обработки несуществующих полей

## Проверка совместимости

### Checklist

- [x] Все поля Document соответствуют backend
- [x] Request типы совпадают с gateway
- [x] Response типы совпадают с gateway
- [x] Имена полей в snake_case
- [x] Опциональность полей соответствует backend
- [x] Mock данные используют правильные имена полей
- [x] API сервис возвращает правильные типы

## Пример использования

### Создание документа

```typescript
import { documentService } from '@/services/document.service';
import type { CreateDocumentRequest } from '@/types/document';

const request: CreateDocumentRequest = {
    title: 'Новый документ',
    description: 'Описание документа',
    authorId: '550e8400-e29b-41d4-a716-446655440001',
    categoryId: 1,
    publicationDate: new Date().toISOString(),
    tagIds: [1, 2, 3]
};

const document = await documentService.create(request);
console.log(document.publication_date); // ISO 8601 string
console.log(document.created_at); // ISO 8601 string
```

### Получение списка документов

```typescript
import { documentService } from '@/services/document.service';
import type { ListDocumentsQuery } from '@/types/document';

const query: ListDocumentsQuery = {
    page: 1,
    limit: 20,
    categoryId: 1,
    search: 'философия'
};

const response = await documentService.getAll(query);
console.log(response.documents); // Document[]
console.log(response.total); // number
console.log(response.totalPages); // number
```

## Рекомендации

1. **Всегда используйте типы** из `@/types/document` вместо `any`
2. **Используйте ISO 8601** для дат при отправке на backend
3. **Проверяйте опциональность** полей перед использованием
4. **Следите за обновлениями** backend типов и синхронизируйте frontend

## Связанные файлы

### Backend
- `services/gateway/internal/domain/models.go`
- `services/gateway/internal/domain/documents.go`
- `libs/dto/documents/models.go`

### Frontend
- `frontend/apps/admin-panel/src/types/document.ts`
- `frontend/apps/admin-panel/src/services/document.service.ts`
- `frontend/apps/admin-panel/src/mocks/data/documents.ts`
- `frontend/apps/admin-panel/src/mocks/handlers/documents.ts`
