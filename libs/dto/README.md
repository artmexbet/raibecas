# DTO Package

Общий модуль для Data Transfer Objects (DTO), используемых для взаимодействия между сервисами через NATS.

## Структура модуля

```
dto/
├── documents/          # DTO для работы с документами
│   ├── models.go       # Структуры данных
│   ├── models_easyjson.go  # Сгенерированные easyjson методы
│   └── README.md       # Документация по документам
├── registration.go     # DTO для регистрации пользователей
├── user.go            # DTO для управления пользователями
├── response.go        # Общие структуры ответов
└── go.mod
```

## Принципы организации

### Организация по доменам

DTO организованы по доменам в отдельные папки:
- `documents/` - всё, что касается документов
- В будущем можно добавить `chat/`, `analytics/` и т.д.

### Базовые типы на уровне корня

Общие структуры, используемые всеми сервисами, находятся в корне:
- `response.go` - стандартные форматы ответов и коды ошибок
- `user.go` - базовые структуры пользователей
- `registration.go` - структуры регистрации

## Использование easyjson

Все DTO используют easyjson для быстрой сериализации/десериализации:

```go
// Сериализация
data, err := dto.MarshalJSON()

// Десериализация  
var dto MyDTO
err := dto.UnmarshalJSON(data)
```

### Преимущества easyjson

1. **Производительность**: в 3-5 раз быстрее стандартного `encoding/json`
2. **Меньше аллокаций**: снижает нагрузку на сборщик мусора
3. **Типобезопасность**: код генерируется на этапе компиляции
4. **Совместимость**: полностью совместим с стандартным JSON

## Добавление новых DTO

### 1. Для существующего домена

Добавьте структуры в соответствующий файл `models.go`:

```go
//easyjson:json
type MyNewRequest struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}
```

Перегенерируйте easyjson:

```bash
cd libs/dto/documents
easyjson -all models.go
```

### 2. Для нового домена

Создайте новую папку и файлы:

```bash
mkdir libs/dto/mynewdomain
cd libs/dto/mynewdomain
```

Создайте `models.go`:

```go
package mynewdomain

import "github.com/google/uuid"

//go:generate easyjson -all models.go

//easyjson:json
type MyRequest struct {
    ID uuid.UUID `json:"id"`
}
```

Сгенерируйте easyjson:

```bash
easyjson -all models.go
```

Создайте README.md с документацией.

## Интеграция в сервисы

### В go.mod сервиса

```go
require github.com/artmexbet/raibecas/libs/dto v0.0.0
```

### Импорт

```go
// Базовые типы
import "github.com/artmexbet/raibecas/libs/dto"

// Документы
import "github.com/artmexbet/raibecas/libs/dto/documents"
```

### Использование в handlers

```go
func (h *Handler) HandleRequest(msg *natsw.Message) error {
    var req documents.CreateDocumentRequest
    if err := req.UnmarshalJSON(msg.Data); err != nil {
        return h.respondError(msg, dto.ErrCodeInvalidRequest)
    }
    
    // ... обработка ...
    
    response := documents.CreateDocumentResponse{
        Document: doc,
    }
    
    return msg.RespondEasyJSON(&response)
}
```

## Стандартные коды ошибок

Определены в `response.go`:

- `ErrCodeInvalidRequest` - неверный формат запроса
- `ErrCodeNotFound` - ресурс не найден
- `ErrCodeInternal` - внутренняя ошибка сервера
- `ErrCodeUnauthorized` - не авторизован
- `ErrCodeForbidden` - доступ запрещен

## Best Practices

1. **Всегда используйте easyjson**: Не используйте `json.Marshal/Unmarshal` напрямую
2. **Добавляйте валидацию**: Используйте теги `validate` где необходимо
3. **Документируйте структуры**: Добавляйте комментарии к каждой структуре
4. **Используйте указатели для optional полей**: `*string` для необязательных полей
5. **Группируйте по доменам**: Держите связанные типы вместе

## Миграция существующего кода

При миграции с `json.Marshal/Unmarshal` на easyjson:

```go
// Было
data, err := json.Marshal(obj)

// Стало
data, err := obj.MarshalJSON()

// Было
var obj MyType
err := json.Unmarshal(data, &obj)

// Стало
var obj MyType
err := obj.UnmarshalJSON(data)
```

## Производительность

Результаты бенчмарков показывают значительное улучшение:

- Сериализация: ~3-5x быстрее
- Десериализация: ~3-5x быстрее
- Аллокации памяти: снижение на 50-70%

Это особенно важно для высоконагруженных сервисов с большим количеством сообщений NATS.
