-- CreateUser users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- CreateUser index on email for faster lookups
CREATE INDEX idx_users_email ON users(email);

-- CreateUser index on username for faster lookups
CREATE INDEX idx_users_username ON users(username);

-- CreateUser index on role for filtering
CREATE INDEX idx_users_role ON users(role);
