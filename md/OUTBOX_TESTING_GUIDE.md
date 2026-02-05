# Quick Start: Testing Outbox Pattern

## Prerequisites

1. **PostgreSQL** running with two databases:
   - `users_db` - for users service
   - `auth_db` - for auth service

2. **NATS** server running on default port (4222)

3. **Redis** running (for auth service tokens)

## Step 1: Apply Migrations

### Users Service
```powershell
# Set environment variable with database URL
$env:DATABASE_URL="postgresql://user:password@localhost:5432/users_db?sslmode=disable"

# Apply migrations
cd C:\Users\artem\GolandProjects\raibecas\services\users
migrate -path migrations -database $env:DATABASE_URL up
```

### Auth Service
```powershell
# Set environment variable
$env:DATABASE_URL="postgresql://user:password@localhost:5432/auth_db?sslmode=disable"

# Apply migrations
cd C:\Users\artem\GolandProjects\raibecas\services\auth
migrate -path migrations -database $env:DATABASE_URL up
```

## Step 2: Start Services

### Terminal 1: Users Service
```powershell
cd C:\Users\artem\GolandProjects\raibecas\services\users
.\users.exe
```

Expected output:
```
INFO starting outbox processor poll_interval=5s batch_size=10
INFO subscribed to user registration events
INFO starting server address=:8082
```

### Terminal 2: Auth Service
```powershell
cd C:\Users\artem\GolandProjects\raibecas\services\auth
.\auth.exe
```

Expected output:
```
INFO subscribed to user registration events
INFO Auth service is ready and listening on NATS topics
```

### Terminal 3: Gateway
```powershell
cd C:\Users\artem\GolandProjects\raibecas\services\gateway
.\gateway.exe
```

## Step 3: Test Registration Flow

### 1. Create Registration Request
```powershell
curl -X POST http://localhost:8080/api/v1/registration-requests `
  -H "Content-Type: application/json" `
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

Expected response:
```json
{
  "id": "uuid",
  "username": "testuser",
  "email": "test@example.com",
  "status": "pending"
}
```

### 2. Login as Admin (to approve request)
```powershell
$response = curl -X POST http://localhost:8080/api/v1/auth/login `
  -H "Content-Type: application/json" `
  -d '{
    "email": "admin@example.com",
    "password": "admin"
  }' | ConvertFrom-Json

$token = $response.access_token
```

### 3. Get Pending Requests
```powershell
curl -X GET "http://localhost:8080/api/v1/registration-requests?status=pending" `
  -H "Authorization: Bearer $token"
```

### 4. Approve Registration Request
```powershell
$requestId = "uuid-from-step-1"

curl -X POST "http://localhost:8080/api/v1/registration-requests/$requestId/approve" `
  -H "Authorization: Bearer $token" `
  -H "Content-Type: application/json"
```

Expected logs in **Users Service**:
```
INFO processing outbox events count=1
INFO event processed successfully event_id=... event_type=user.registered subject=users.user.registered
```

Expected logs in **Auth Service**:
```
INFO received user registered event user_id=... email=test@example.com username=testuser
INFO user created successfully user_id=... email=test@example.com username=testuser
```

### 5. Test Login with New User
```powershell
curl -X POST http://localhost:8080/api/v1/auth/login `
  -H "Content-Type: application/json" `
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

Expected response:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "..."
}
```

## Step 4: Verify Outbox Processing

### Check Outbox Table in Users DB
```sql
SELECT id, aggregate_id, event_type, processed_at, retry_count 
FROM outbox 
ORDER BY created_at DESC 
LIMIT 10;
```

Expected result:
- `processed_at` should have timestamp
- `retry_count` should be 0

### Check Users in Both DBs

**Users DB:**
```sql
SELECT id, username, email, role FROM users WHERE email = 'test@example.com';
```

**Auth DB:**
```sql
SELECT id, username, email, role FROM users WHERE email = 'test@example.com';
```

**Both should return the same user with the same UUID!**

## Step 5: Test Idempotency

### Simulate Duplicate Processing

1. Stop auth service
2. Create and approve another registration request
3. Manually mark outbox event as unprocessed:
```sql
-- In users DB
UPDATE outbox 
SET processed_at = NULL 
WHERE event_type = 'user.registered' 
AND aggregate_id = (SELECT id FROM users WHERE email = 'test2@example.com');
```
4. Start auth service
5. Check auth logs - should see "user already exists, skipping"
6. Verify no duplicate users in auth DB

## Troubleshooting

### Issue: Outbox events not processing
**Check:**
- Users service logs for "starting outbox processor"
- NATS connection status
- Database connectivity

### Issue: Auth service not receiving events
**Check:**
- NATS subject: should be `users.user.registered`
- Auth logs for subscription confirmation
- NATS monitoring: `nats sub "users.user.registered"`

### Issue: Duplicate users in auth DB
**Check:**
- Consumer idempotency logic
- Database constraints (email/username UNIQUE)
- Outbox `retry_count` - should not exceed 5

## Monitoring

### Check Outbox Status
```sql
-- Unprocessed events
SELECT COUNT(*) FROM outbox WHERE processed_at IS NULL;

-- Failed events (multiple retries)
SELECT id, event_type, retry_count, last_error 
FROM outbox 
WHERE retry_count > 0 
ORDER BY created_at DESC;

-- Average processing time
SELECT 
    event_type,
    AVG(EXTRACT(EPOCH FROM (processed_at - created_at))) as avg_seconds
FROM outbox 
WHERE processed_at IS NOT NULL
GROUP BY event_type;
```

## Success Criteria

✅ Registration request created successfully  
✅ User created in users DB  
✅ Outbox event created in transaction  
✅ Event processed within 5 seconds  
✅ User created in auth DB with same UUID  
✅ Login works with new credentials  
✅ No duplicate users on retry  
✅ All outbox events marked as processed  

## Next Steps

After successful testing, consider:
- Adding Prometheus metrics
- Implementing cleanup job for old outbox events
- Adding exponential backoff for retries
- Setting up alerting for failed events
