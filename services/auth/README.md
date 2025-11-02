# Auth Service

Authentication microservice for the Raibecas platform. Handles user registration, login, logout, and token management using JWT, PostgreSQL, and Redis.

## Features

- **Registration with Moderation**: Users submit registration requests that require admin approval
- **JWT Authentication**: Secure token-based authentication with access and refresh tokens
- **Session Management**: Redis-backed session storage for refresh tokens
- **Event-Driven Architecture**: NATS Pub/Sub for event communication with other services
- **Password Security**: bcrypt password hashing with configurable cost
- **Modern Go Patterns**: Clean architecture with dependency injection

## Architecture

The service follows clean architecture principles:

```
auth/
├── cmd/
│   └── auth/           # Application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── domain/         # Domain models and errors
│   ├── repository/     # Data access layer (PostgreSQL)
│   ├── storeredis/     # Redis token storage
│   ├── service/        # Business logic
│   ├── handler/        # HTTP handlers
│   ├── middleware/     # HTTP middleware
│   ├── nats/          # NATS event pub/sub
│   └── server/        # Server setup
├── pkg/
│   └── jwt/           # JWT token management
└── migrations/        # Database migrations
```

## API Endpoints

### Public Endpoints

#### POST /api/v1/register
Create a registration request (pending admin approval).

**Request:**
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePassword123",
  "metadata": {
    "reason": "Research purposes"
  }
}
```

**Response:**
```json
{
  "request_id": "uuid",
  "status": "pending",
  "message": "Registration request submitted successfully. Waiting for admin approval."
}
```

#### POST /api/v1/login
Authenticate user and get tokens.

**Request:**
```json
{
  "email": "john@example.com",
  "password": "SecurePassword123"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "uuid",
  "expires_in": 900
}
```

#### POST /api/v1/validate
Validate an access token.

**Request:**
```json
{
  "token": "eyJhbGc..."
}
```

**Response:**
```json
{
  "valid": true,
  "user_id": "uuid",
  "role": "user"
}
```

### Protected Endpoints (Require Authorization: Bearer <token>)

#### POST /api/v1/logout
Logout from current device.

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

#### POST /api/v1/logout-all
Logout from all devices.

**Response:**
```json
{
  "message": "Logged out from all devices successfully"
}
```

#### POST /api/v1/change-password
Change user password.

**Request:**
```json
{
  "old_password": "OldPassword123",
  "new_password": "NewPassword456"
}
```

**Response:**
```json
{
  "message": "Password changed successfully"
}
```

## NATS Events

### Published Events

- `auth.user.registered` - When a new user is created (after approval)
- `auth.user.login` - When a user logs in
- `auth.user.logout` - When a user logs out
- `auth.password.reset` - When a password is changed
- `auth.registration.requested` - When a registration request is created

### Subscribed Events

- `admin.registration.approved` - Admin approves a registration request
- `admin.registration.rejected` - Admin rejects a registration request

## Configuration

Configuration is loaded from environment variables:

### Server Configuration
- `SERVER_PORT` - Server port (default: 8081)
- `SERVER_READ_TIMEOUT` - Read timeout (default: 10s)
- `SERVER_WRITE_TIMEOUT` - Write timeout (default: 10s)
- `SERVER_SHUTDOWN_TIMEOUT` - Shutdown timeout (default: 5s)

### Database Configuration
- `DB_HOST` - PostgreSQL host (default: localhost)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - PostgreSQL user (default: raibecas)
- `DB_PASSWORD` - PostgreSQL password (required)
- `DB_NAME` - Database name (default: raibecas)
- `DB_SSL_MODE` - SSL mode (default: disable)
- `DB_MAX_CONNS` - Max connections (default: 25)
- `DB_MIN_CONNS` - Min connections (default: 5)

### Redis Configuration
- `REDIS_HOST` - Redis host (default: localhost)
- `REDIS_PORT` - Redis port (default: 6379)
- `REDIS_PASSWORD` - Redis password (optional)
- `REDIS_DB` - Redis database number (default: 0)

### NATS Configuration
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)
- `NATS_MAX_RECONNECTS` - Max reconnection attempts (default: 10)
- `NATS_RECONNECT_WAIT` - Reconnection wait time (default: 2s)

### JWT Configuration
- `JWT_SECRET` - JWT signing secret (required)
- `JWT_ACCESS_TTL` - Access token TTL (default: 15m)
- `JWT_REFRESH_TTL` - Refresh token TTL (default: 168h / 7 days)
- `JWT_ISSUER` - Token issuer (default: raibecas-auth)

## Development

### Prerequisites
- Go 1.25.1 or later
- Docker and Docker Compose
- PostgreSQL 16 with pgvector extension
- Redis 7
- NATS Server

### Setup

1. Start dependencies:
```bash
docker-compose -f docker-compose.dev.yml up -d postgres redis nats
```

2. Run migrations:
```bash
# Connect to PostgreSQL and run migration files in migrations/ directory
psql -h localhost -U raibecas -d raibecas -f migrations/001_create_users_table.sql
psql -h localhost -U raibecas -d raibecas -f migrations/002_create_registration_requests_table.sql
```

3. Set environment variables:
```bash
export DB_PASSWORD=raibecas_dev
export JWT_SECRET=dev_secret_change_in_production
```

4. Run the service:
```bash
go run cmd/auth/main.go
```

### Running with Docker Compose

```bash
docker-compose -f docker-compose.dev.yml up --build
```

## Testing

Run unit tests:
```bash
go test ./... -v
```

Run integration tests (requires testcontainers):
```bash
go test ./... -v -tags=integration
```

## Security

- Passwords are hashed using bcrypt with cost 12
- JWT tokens are signed using HS256
- Access tokens expire after 15 minutes
- Refresh tokens expire after 7 days
- All sensitive endpoints require authentication
- Password change automatically logs out all devices

## Database Schema

### users table
- `id` - UUID primary key
- `username` - Unique username
- `email` - Unique email
- `password_hash` - bcrypt hashed password
- `role` - User role (user/admin)
- `is_active` - Account active status
- `created_at` - Creation timestamp
- `updated_at` - Update timestamp

### registration_requests table
- `id` - UUID primary key
- `username` - Requested username
- `email` - Requested email
- `password` - Hashed password
- `status` - Request status (pending/approved/rejected)
- `metadata` - Additional JSON metadata
- `created_at` - Creation timestamp
- `updated_at` - Update timestamp
- `approved_by` - UUID of approver (foreign key to users)
- `approved_at` - Approval timestamp

## License

MIT
