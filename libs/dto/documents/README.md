# Documents DTO Package

Этот пакет содержит общие структуры данных (DTO) для взаимодействия между сервисами Gateway и Documents через NATS.

## Структура

- `models.go` - Основные структуры данных для документов
- `models_easyjson.go` - Сгенерированные easyjson методы для быстрой сериализации/десериализации

## Основные типы

### Запросы и Ответы

- **ListDocumentsQuery** - Параметры запроса для получения списка документов
- **ListDocumentsResponse** - Ответ со списком документов
- **CreateDocumentRequest** - Запрос на создание документа
- **CreateDocumentResponse** - Ответ с созданным документом
- **GetDocumentRequest** - Запрос на получение документа по ID
- **GetDocumentResponse** - Ответ с документом
- **GetDocumentContentRequest** - Запрос на получение содержимого документа
- **GetDocumentContentResponse** - Ответ с содержимым документа
- **UpdateDocumentRequest** - Запрос на обновление документа
- **UpdateDocumentResponse** - Ответ с обновленным документом
- **DeleteDocumentRequest** - Запрос на удаление документа
- **DeleteDocumentResponse** - Ответ об успешном удалении
- **ListDocumentVersionsRequest** - Запрос на получение версий документа
- **ListDocumentVersionsResponse** - Ответ со списком версий

### Модели данных

- **Document** - Научный документ
- **Author** - Автор научной работы
- **Category** - Категория документа
- **Tag** - Тег документа
- **DocumentVersion** - Версия документа

## Использование

### В сервисе Gateway

```go
import "github.com/artmexbet/raibecas/libs/dto/documents"

// Создание запроса
req := documents.CreateDocumentRequest{
    Title:           "Title",
    AuthorID:        authorID,
    CategoryID:      1,
    PublicationDate: time.Now(),
}

// Сериализация с easyjson
data, err := req.MarshalJSON()

// Десериализация с easyjson
var response documents.CreateDocumentResponse
err = response.UnmarshalJSON(data)
```

### В сервисе Documents

```go
import "github.com/artmexbet/raibecas/libs/dto/documents"

// Обработка запроса
func (h *Handler) HandleCreateDocument(msg *natsw.Message) error {
    var req documents.CreateDocumentRequest
    if err := req.UnmarshalJSON(msg.Data); err != nil {
        return err
    }
    
    // ... бизнес-логика ...
    
    response := documents.CreateDocumentResponse{
        Document: dtoDoc,
    }
    
    return msg.RespondEasyJSON(&response)
}
```

## Генерация easyjson

При изменении структур необходимо перегенерировать easyjson:

```bash
cd libs/dto/documents
easyjson -all models.go
```

## Преимущества использования easyjson

1. **Производительность** - в 3-5 раз быстрее стандартного `encoding/json`
2. **Меньше аллокаций** - меньше нагрузка на GC
3. **Типобезопасность** - генерируется код на этапе компиляции
4. **Совместимость** - совместимо со стандартным JSON

## Интеграция с другими сервисами

Для добавления поддержки документов в новый сервис:

1. Добавьте зависимость в `go.mod`:
```go
require github.com/artmexbet/raibecas/libs/dto v0.0.0
```

2. Импортируйте пакет:
```go
import "github.com/artmexbet/raibecas/libs/dto/documents"
```

3. Используйте методы `MarshalJSON()` и `UnmarshalJSON()` для сериализации
