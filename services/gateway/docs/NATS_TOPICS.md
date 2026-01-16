# NATS Topics для Gateway Service

Документация по NATS топикам, используемым Gateway для взаимодействия с микросервисами.

## Общая информация

- **Протокол**: Request-Reply pattern
- **Формат сообщений**: JSON
- **Timeout**: 10 секунд

---

## Auth Service Topics

### `auth.login`
Аутентификация пользователя

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "device_id": "optional-device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

### `auth.refresh`
Обновление access токена

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "device_id": "optional-device-uuid",
  "user_agent": "Mozilla/5.0...",
  "ip_address": "192.168.1.1"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 900
  }
}
```

### `auth.validate`
Валидация токена

**Request:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "role": "user"
  }
}
```

### `auth.logout`
Выход из текущего устройства

**Request:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Logged out successfully"
  }
}
```

### `auth.logout_all`
Выход со всех устройств

**Request:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Logged out from all devices successfully"
  }
}
```

### `auth.change_password`
Изменение пароля

**Request:**
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "old_password": "oldpassword123",
  "new_password": "newpassword456"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Password changed successfully"
  }
}
```

---

## Document Service Topics

### `documents.list`
Получение списка документов с фильтрацией и пагинацией

**Request:**
```json
{
  "page": 1,
  "limit": 20,
  "author_id": "123e4567-e89b-12d3-a456-426614174000",
  "category_id": 5,
  "tag_id": 10,
  "search": "keyword"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "documents": [
      {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "title": "Document Title",
        "description": "Description...",
        "author": {
          "id": "123e4567-e89b-12d3-a456-426614174001",
          "name": "Author Name"
        },
        "category": {
          "id": 5,
          "title": "Category Title"
        },
        "publication_date": "2026-01-15T00:00:00Z",
        "tags": [
          {"id": 1, "title": "Tag 1"},
          {"id": 2, "title": "Tag 2"}
        ],
        "created_at": "2026-01-15T10:00:00Z",
        "updated_at": "2026-01-15T10:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20,
    "total_pages": 5
  }
}
```

### `documents.get`
Получение документа по ID

**Request:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "document": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Document Title",
      "description": "Description...",
      "author": {
        "id": "123e4567-e89b-12d3-a456-426614174001",
        "name": "Author Name"
      },
      "category": {
        "id": 5,
        "title": "Category Title"
      },
      "publication_date": "2026-01-15T00:00:00Z",
      "tags": [
        {"id": 1, "title": "Tag 1"}
      ],
      "created_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-15T10:00:00Z"
    }
  }
}
```

### `documents.create`
Создание нового документа

**Request:**
```json
{
  "title": "New Document",
  "description": "Description...",
  "author_id": "123e4567-e89b-12d3-a456-426614174000",
  "category_id": 5,
  "publication_date": "2026-01-15T00:00:00Z",
  "tag_ids": [1, 2, 3]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "document": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "New Document",
      ...
    }
  }
}
```

### `documents.update`
Обновление документа

**Request:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "title": "Updated Title",
  "description": "Updated description...",
  "author_id": "123e4567-e89b-12d3-a456-426614174000",
  "category_id": 6,
  "publication_date": "2026-01-16T00:00:00Z",
  "tag_ids": [2, 3, 4]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "document": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Updated Title",
      ...
    }
  }
}
```

### `documents.delete`
Удаление документа

**Request:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Response:**
```json
{
  "success": true,
  "data": null
}
```

---

## Обработка ошибок

При ошибке возвращается:

```json
{
  "success": false,
  "error": "Error message description"
}
```
