# Error Handling в Users Service

## 📋 Структура ошибок

Все ошибки service layer вынесены в отдельный файл `internal/service/errors.go`:

```
services/users/internal/service/
├── errors.go          ← Все ошибки (centralized)
├── service.go         ← Использует ошибки из errors.go
├── handler.go         ← Проверяет ошибки через errors.Is()
└── ...
```

---

## 🔍 Определённые ошибки

### **User Management**

```go
ErrNotFound = errors.New("not found")
```
- Возвращается когда пользователь не найден
- Используется: `GetUserByID()`, `UpdateUser()`, `ApproveRegistrationRequest()`

```go
ErrInvalidUserID = errors.New("invalid user id")
```
- Возвращается когда UUID nil или некорректный
- Используется: `GetUserByID()`, `UpdateUser()`, `DeleteUser()`

### **Registration**

```go
ErrRegistrationRequestNil = errors.New("registration request cannot be nil")
```
- Возвращается когда registration request = nil
- Используется: `CreateRegistrationRequest()`

```go
ErrMissingRequiredFields = errors.New("missing required fields")
```
- Возвращается когда email, username, password пустые
- Используется: `CreateRegistrationRequest()`

```go
ErrInvalidRequestOrApproverID = errors.New("invalid request or approver id")
```
- Возвращается когда requestID или approverID = nil
- Используется: `ApproveRegistrationRequest()`, `RejectRegistrationRequest()`

### **General**

```go
ErrInvalidStatus = errors.New("invalid status")
```
- Зарезервирована для будущего использования статусов
- Используется: (на будущее)

---

## 💡 Преимущества

### **1. Centralized Error Definitions**
```go
// ✅ Все ошибки в одном месте
import "github.com/artmexbet/raibecas/services/users/internal/service"

if errors.Is(err, service.ErrNotFound) {
    // handle 404
}
```

### **2. Testability**
```go
func TestGetUserNotFound(t *testing.T) {
    // Теперь легко тестировать конкретные ошибки
    svc := NewService(repo, metrics)
    _, err := svc.GetUserByID(ctx, invalidID)
    
    if !errors.Is(err, service.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

### **3. Consistency**
```go
// Все ошибки с одинаковым форматом
var (
    ErrXXX = errors.New("description")
    ErrYYY = errors.New("description")
    // ...
)
```

### **4. Type Safety**
```go
// Нельзя случайно создать опечатку в имени ошибки
// IDE подскажет все доступные ошибки
if errors.Is(err, service.ErrIvalidUserID) {  // ← IDE выделит ошибку
```

---

## 🧪 Использование в тестах

### **До (плохо)**
```go
func TestGetUser(t *testing.T) {
    _, err := svc.GetUserByID(ctx, invalidID)
    if err == nil {
        t.Error("expected error")
    }
    // ❌ Не знаем какой тип ошибки
}
```

### **После (хорошо)**
```go
func TestGetUserNotFound(t *testing.T) {
    _, err := svc.GetUserByID(ctx, invalidID)
    if !errors.Is(err, service.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}

func TestGetUserInvalidID(t *testing.T) {
    _, err := svc.GetUserByID(ctx, uuid.Nil)
    if !errors.Is(err, service.ErrInvalidUserID) {
        t.Errorf("expected ErrInvalidUserID, got %v", err)
    }
}

func TestApproveRegistrationInvalidID(t *testing.T) {
    _, err := svc.ApproveRegistrationRequest(ctx, uuid.Nil, uuid.Nil)
    if !errors.Is(err, service.ErrInvalidRequestOrApproverID) {
        t.Errorf("expected ErrInvalidRequestOrApproverID, got %v", err)
    }
}
```

---

## 📝 Использование в handler'е

```go
package handler

import (
    "errors"
    "github.com/artmexbet/raibecas/services/users/internal/service"
)

func (h *Handler) HandleGetUser(msg *natsw.Message) error {
    var req GetUserRequest
    if err := msg.UnmarshalData(&req); err != nil {
        slog.ErrorContext(msg.Ctx, "invalid get user request", "error", err)
        return h.respondError(msg, "invalid_request")
    }

    user, err := h.service.GetUserByID(msg.Ctx, req.ID)
    if err != nil {
        // ✅ Проверяем конкретную ошибку
        if errors.Is(err, service.ErrNotFound) {
            slog.DebugContext(msg.Ctx, "user not found", "user_id", req.ID)
            return h.respondError(msg, "not_found")  // 404
        }
        if errors.Is(err, service.ErrInvalidUserID) {
            slog.DebugContext(msg.Ctx, "invalid user id")
            return h.respondError(msg, "invalid_request")  // 400
        }
        
        slog.ErrorContext(msg.Ctx, "failed to get user", "user_id", req.ID, "error", err)
        return h.respondError(msg, "internal_error")  // 500
    }

    return h.respond(msg, map[string]interface{}{"user": user})
}
```

---

## 🔧 Расширение ошибок

Если нужна новая ошибка:

```go
// errors.go
var (
    // ... existing errors ...
    
    // Add new error
    ErrUserAlreadyExists = errors.New("user already exists")
)

// service.go
func (s *Service) CreateUser(ctx context.Context, user *domain.User) error {
    exists, err := s.repo.ExistsUserByEmail(ctx, user.Email)
    if err != nil {
        return fmt.Errorf("failed to check user existence: %w", err)
    }
    if exists {
        return ErrUserAlreadyExists  // ← Используем новую ошибку
    }
    // ...
}
```

---

## ✅ Проверка в тестах

```bash
# Убедиться что все ошибки используются
grep -r "service\\.Err" services/users/internal/

# Убедиться что нет inline errors.New()
grep -r "errors\\.New" services/users/internal/service/service.go
# ✅ Должна быть пусто (только в errors.go)
```

---

## 📚 Best Practices

1. **Один файл для ошибок** - `errors.go` в пакете
2. **Понятные имена** - `ErrXXX` для всех ошибок
3. **Документация** - комментарии о когда возвращается ошибка
4. **Тестируемость** - используй `errors.Is()` в тестах
5. **Консистентность** - все ошибки через `errors.New()`

---

## 🎯 Итог

- ✅ Централизованное управление ошибками
- ✅ Легко добавлять новые ошибки
- ✅ Удобно тестировать конкретные случаи
- ✅ Улучшена читаемость кода
- ✅ Production-ready решение
