# Руководство по работе с авторизацией на фронтенде

## Обзор изменений

Gateway теперь использует **современную архитектуру токенов** с разделением ответственности:

- **Access Token** — короткоживущий (15 мин), передается в JSON, используется для авторизации запросов
- **Refresh Token** — долгоживущий (30 дней), хранится в **HttpOnly cookie**, недоступен для JavaScript
- **Token ID & Fingerprint** — служебные данные, хранятся в **HttpOnly cookies**, недоступны для JavaScript

## Что изменилось в API

### 1. Login (POST `/api/v1/auth/login`)

**Запрос:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Ответ (JSON):**
```json
{
  "access_token": "eyJhbGc...",
  "expires_in": 900,
  "token_type": "Bearer",
  "user": {
    "id": "uuid",
    "role": "admin"
  }
}
```

**Cookies (HttpOnly, автоматически устанавливаются):**
- `refresh_token` — для обновления сессии
- `token_id` — ID токена для операций
- `fingerprint` — защита от CSRF

### 2. Refresh Token (POST `/api/v1/auth/refresh`)

**Запрос:**
```json
{
  "deviceId": "optional"
}
```

**Ответ:**
```json
{
  "access_token": "eyJhbGc...",
  "expires_in": 900,
  "token_type": "Bearer",
  "user": {
    "id": "uuid",
    "role": "admin"
  }
}
```

**Важно:** Refresh token автоматически берется из cookie, не нужно передавать в теле запроса!

### 3. Logout (POST `/api/v1/auth/logout`)

**Запрос:**
```json
{
  "token": "current_access_token"
}
```

**Ответ:**
```json
{
  "message": "Logged out successfully"
}
```

**Cookies автоматически очищаются!**

## Как это влияет на фронтенд

### 1. **Хранение токенов**

✅ **Access Token:**
- Хранить в `localStorage` или `sessionStorage`
- Отправлять в заголовке `Authorization: Bearer <token>`

❌ **Refresh Token:**
- НЕ хранить в localStorage/sessionStorage
- НЕ доступен для JavaScript
- Автоматически отправляется браузером через cookie

### 2. **Axios/Fetch настройка**

```typescript
const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  withCredentials: true, // ⚠️ ВАЖНО: для отправки cookies
});
```

### 3. **Интерцептор для refresh**

```typescript
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    // Если 401 и еще не пытались обновить
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        // Вызов refresh — cookie отправляется автоматически
        const { data } = await api.post('/auth/refresh', {});
        
        // Сохраняем новый access token
        localStorage.setItem('access_token', data.access_token);
        
        // Повторяем оригинальный запрос
        originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
        return api(originalRequest);
      } catch (refreshError) {
        // Redirect на логин
        window.location.href = '/login';
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);
```

### 4. **Добавление access token к запросам**

```typescript
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

## Преимущества новой архитектуры

### 🔒 Безопасность
- **XSS защита:** Refresh token недоступен для JavaScript
- **CSRF защита:** Fingerprint проверяется на бэкенде
- **Token Rotation:** Refresh token обновляется при каждом использовании

### ⚡ Производительность
- Короткоживущий access token (15 мин) — меньше нагрузка на Redis
- Refresh только при истечении — меньше запросов к Auth сервису

### 🎯 Удобство
- Автоматическая работа с cookies — меньше кода на фронте
- Единая точка обновления токенов — через интерцептор

## Миграция существующего кода

### Было:
```typescript
// ❌ Старый подход
const { data } = await api.post('/auth/login', credentials);
localStorage.setItem('access_token', data.access_token);
localStorage.setItem('refresh_token', data.refresh_token); // Небезопасно!
```

### Стало:
```typescript
// ✅ Новый подход
const { data } = await api.post('/auth/login', credentials);
localStorage.setItem('access_token', data.access_token);
// refresh_token автоматически в cookie — ничего не делаем!
```

### Было:
```typescript
// ❌ Старый refresh
const { data } = await api.post('/auth/refresh', {
  refreshToken: localStorage.getItem('refresh_token')
});
```

### Стало:
```typescript
// ✅ Новый refresh
const { data } = await api.post('/auth/refresh', {});
// cookie отправляется автоматически!
```

## Важные замечания

1. **withCredentials: true** — обязательно для всех запросов к Gateway
2. **CORS настройки** — Gateway должен разрешать credentials
3. **HTTPS обязателен** — для secure cookies в production
4. **SameSite: Strict** — защита от CSRF атак

## Пример полной настройки

См. файл `/frontend/apps/admin-panel/src/lib/api/auth.ts` для полной имплементации.
