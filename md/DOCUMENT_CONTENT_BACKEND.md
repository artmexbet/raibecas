# Возврат отправки контента документов с backend

## Изменения

### Backend

#### 1. Gateway Domain Model
**`services/gateway/internal/domain/models.go`**
```go
type Document struct {
    // ...existing fields...
    Content *string `json:"content,omitempty"` // ← ДОБАВЛЕНО
    Additional
}
```
- ✅ Добавлено поле `Content *string` в структуру `Document`
- ✅ Перегенерирован easyjson

#### 2. Gateway Connector
**`services/gateway/internal/connector/nats_connector.go`**

**Добавлен subject:**
```go
const (
    // ...existing subjects...
    SubjectDocumentsGetContent = "documents.get_content" // ← ДОБАВЛЕНО
)
```

**Обновлён метод GetDocument:**
```go
func (c *NATSDocumentConnector) GetDocument(ctx context.Context, id uuid.UUID) (*domain.GetDocumentResponse, error) {
    // 1. Получаем метаданные документа
    // ...existing code...
    
    // 2. Получаем контент документа отдельным запросом
    contentReq := documents.GetDocumentContentRequest{ID: id}
    // ...запрос к documents.get_content...
    
    // 3. Объединяем данные
    doc := convertDocument(dtoResponse.Document)
    doc.Content = &contentResponse.Content
    
    return &domain.GetDocumentResponse{Document: doc}, nil
}
```

### Frontend

#### 1. TypeScript Types
**`frontend/apps/admin-panel/src/types/document.ts`**
```typescript
export interface Document {
    // ...existing fields...
    content?: string | null; // ← ДОБАВЛЕНО: Markdown content
    // ...
}
```

#### 2. Mock Data
**`frontend/apps/admin-panel/src/mocks/data/documents.ts`**
- ✅ Добавлено поле `content` во все mock документы
- ✅ Добавлено поле `content` в `createMockDocument()`

#### 3. Mock Handlers
**`frontend/apps/admin-panel/src/mocks/handlers/documents.ts`**
- ✅ Добавлена обработка поля `content` в методе `update()`

## Архитектура получения документа

```
Frontend
   ↓ GET /documents/:id
Gateway
   ↓ 1. documents.get (метаданные)
   ↓ 2. documents.get_content (контент)
Documents Service
   ↓ Response: Document + Content
Gateway
   ↓ Response: Document (с content)
Frontend
```

## Преимущества текущего подхода

1. **Разделение запросов** - метаданные и контент получаются отдельно
2. **Оптимизация списков** - при получении списка документов контент не загружается
3. **Гибкость** - можно легко вернуться к ленивой загрузке контента

## Что работает

### При получении списка документов (GET /documents)
- ✅ Возвращаются только метаданные (без content)
- ✅ Быстрая загрузка списка

### При получении одного документа (GET /documents/:id)
- ✅ Возвращаются метаданные + контент
- ✅ Контент доступен сразу для редактирования

### При редактировании (PUT /documents/:id)
- ✅ Можно отправлять контент в теле запроса
- ✅ Frontend получает полный документ с контентом

## Проверка

- ✅ Gateway компилируется без ошибок
- ✅ TypeScript типы обновлены
- ✅ Mock данные содержат content
- ✅ DocumentEditPage работает с content

## Использование

### Frontend - получение документа
```typescript
const document = await documentService.getById(id);
// document.content уже доступен
console.log(document.content); // "# Заголовок\n\nТекст..."
```

### Frontend - редактирование
```typescript
// DocumentEditPage.tsx уже работает с content
form.setFieldsValue({
    title: data.title,
    content: data.content, // ← теперь загружается с backend
    // ...
});
```

## Будущие улучшения

1. **Кэширование** - добавить кэширование контента в Redis
2. **Версионирование** - получать контент конкретной версии документа
3. **Потоковая передача** - для очень больших документов использовать streaming
4. **Сжатие** - сжимать контент при передаче по сети

## Связанные файлы

### Backend
- `services/gateway/internal/domain/models.go`
- `services/gateway/internal/connector/nats_connector.go`
- `services/documents/internal/server/document_handler.go` (уже поддерживает get_content)

### Frontend
- `frontend/apps/admin-panel/src/types/document.ts`
- `frontend/apps/admin-panel/src/mocks/data/documents.ts`
- `frontend/apps/admin-panel/src/mocks/handlers/documents.ts`
- `frontend/apps/admin-panel/src/pages/DocumentEditPage.tsx`

---

**Статус**: ✅ Реализовано  
**Дата**: 2026-02-06
