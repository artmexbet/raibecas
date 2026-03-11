# План реализации обложек для документов

## Обзор

Добавить поддержку обложек (cover images) для документов: загрузка в MinIO, получение presigned URL, отображение на фронтенде.

---

## Затронутые компоненты

| Слой | Файл/Компонент | Изменение |
|------|---------------|-----------|
| БД | `migrations/` | Новая миграция: поле `cover_path` в таблице `documents` |
| Domain (documents) | `internal/domain/document.go` | Поле `CoverPath *string` в `Document`, `CreateDocumentRequest`, `UpdateDocumentRequest` |
| Storage (documents) | `internal/storage/minio.go` | Методы `SaveCover`, `GetCoverURL` (presigned), `DeleteCover` |
| Service interfaces | `internal/service/interfaces.go` | Метод `GetPresignedURL` в интерфейсе `Storage` |
| Service (documents) | `internal/service/document_service.go` | Логика сохранения/удаления обложки, обогащение ответа ссылкой |
| Handler (documents) | `internal/server/document_handler.go` | Новый хендлер `HandleUploadCover` (отдельный NATS subject) |
| Server NATS routing | `internal/server/server.go` | Регистрация нового subject `documents.cover.upload` |
| DTO lib | `libs/dto/documents/models.go` | Поле `CoverURL *string` в `Document`; новые типы `UploadCoverRequest/Response` |
| Gateway domain | `services/gateway/internal/domain/documents.go` | Поле `CoverURL *string` в `Document`; `CreateDocumentRequest` принимает multipart |
| Gateway connector | `services/gateway/internal/connector/document_connector.go` | Метод `UploadCover` |
| Gateway connector interface | `services/gateway/internal/server/document_connector.go` | `UploadCover(ctx, id, data, contentType, size)` |
| Gateway server | `services/gateway/internal/server/documents.go` | Маршрут `POST /documents/:id/cover` (multipart), обогащение ответа `coverUrl` |
| Gateway router | `services/gateway/internal/server/server.go` | Регистрация маршрута |
| Admin-panel types | `frontend/apps/admin-panel/src/types/document.ts` | Поле `cover_url?: string` в `Document` |
| Admin-panel service | `frontend/apps/admin-panel/src/services/document.service.ts` | Метод `uploadCover(id, file)` |
| Admin-panel pages | `DocumentCreatePage.tsx`, `DocumentEditPage.tsx` | Upload-поле для обложки |
| User-app types | `frontend/apps/user-app/src/types/document.ts` | Поле `cover_url?: string` в `Document` |
| User-app component | `DocumentCard.tsx` | Отображение обложки (img/placeholder) |
| User-app page | `DocumentViewPage.tsx` | Отображение обложки вверху страницы |

---

## Пошаговый план реализации

### Шаг 1 — Миграция БД
- Новый файл `000005_add_cover_path.up.sql`:
  ```sql
  ALTER TABLE documents ADD COLUMN IF NOT EXISTS cover_path VARCHAR(500);
  ```
- Соответствующий `.down.sql`.

### Шаг 2 — Backend: documents service

1. **`domain/document.go`** — добавить `CoverPath *string` в `Document`, `CreateDocumentRequest`, `UpdateDocumentRequest`.
2. **`storage/minio.go`** — добавить:
   - `SaveCover(ctx, documentID, data []byte, contentType string) (path string, err error)` — сохраняет файл по пути `covers/<id>.<ext>`.
   - `GetCoverPresignedURL(ctx, path string, ttl time.Duration) (string, error)` — presigned GET URL на 24 часа.
   - `DeleteCover(ctx, path string) error`.
3. **`service/interfaces.go`** — добавить методы в интерфейс `Storage`.
4. **`service/document_service.go`** — при `GetDocument`/`ListDocuments` если `CoverPath != nil` — генерировать presigned URL и класть в `CoverURL`.
5. **`server/document_handler.go`** — добавить `HandleUploadCover`:
   - Принимает `UploadCoverRequest{ID uuid.UUID, Data []byte, ContentType string}`.
   - Вызывает сервис, возвращает `UploadCoverResponse{CoverURL string}`.
6. **`server/server.go`** — зарегистрировать `documents.cover.upload`.

### Шаг 3 — Shared DTO lib

В `libs/dto/documents/models.go`:
- `Document` — добавить `CoverURL *string`.
- Новые типы:
  ```go
  type UploadCoverRequest struct {
      ID          uuid.UUID `json:"id"`
      Data        []byte    `json:"data"`
      ContentType string    `json:"content_type"`
  }
  type UploadCoverResponse struct {
      CoverURL string `json:"cover_url"`
  }
  ```
- Перегенерировать easyjson.

### Шаг 4 — Gateway service

1. **`domain/documents.go`** — `Document` + `CoverURL *string`, `UpdateDocumentRequest` + поле.
2. **`connector/`** — добавить метод `UploadCover(ctx, id uuid.UUID, data []byte, contentType string) (string, error)`.
3. **`server/document_connector.go`** (интерфейс) — добавить метод.
4. **`server/documents.go`** — новый хендлер `uploadCover`:
   - `POST /documents/:id/cover`
   - Принимает `multipart/form-data`, поле `cover` (изображение).
   - Ограничение: max 5 МБ, типы `image/jpeg`, `image/png`, `image/webp`.
   - Передаёт байты в коннектор → NATS → documents service.
5. **`server/server.go`** — регистрация маршрута (только для admin).

### Шаг 5 — Frontend: Admin Panel

1. **`types/document.ts`** — `cover_url?: string`.
2. **`services/document.service.ts`** — метод `uploadCover(id: string, file: File): Promise<string>` (FormData + multipart).
3. **`DocumentCreatePage.tsx`** — поле Upload для обложки:
   - `beforeUpload` — проверка типа/размера.
   - После создания документа — отдельный вызов `uploadCover`.
4. **`DocumentEditPage.tsx`** — аналогично + показ текущей обложки.

### Шаг 6 — Frontend: User App

1. **`types/document.ts`** (если отдельный) — `cover_url?: string`.
2. **`DocumentCard.tsx`** — если есть `cover_url`, рендерить `<img>` в заголовке карточки (с `object-fit: cover`, высота ~160px), иначе placeholder-градиент.
3. **`DocumentViewPage.tsx`** — hero-баннер с обложкой вверху страницы.

---

## Архитектурные решения

| Вопрос | Решение |
|--------|---------|
| Хранилище | MinIO, отдельная папка `covers/` в том же bucket |
| URL | Presigned GET URL с TTL 24ч (генерируется при каждом запросе документа) |
| Формат файла | JPEG, PNG, WebP; ограничение 5 МБ |
| NATS transport | Бинарные данные изображения передаются как `[]byte` в JSON (base64 via easyjson) — допустимо для ≤5 МБ |
| Миграция БД | Аддитивная: `cover_path` nullable, существующие документы без обложки работают как прежде |
| Удаление обложки | При удалении документа — удаляем файл обложки из MinIO |

---

## Порядок выполнения

1. Миграция БД → 2. DTO lib → 3. Documents service (domain + storage + service + handler) → 4. Gateway → 5. Admin panel → 6. User app

