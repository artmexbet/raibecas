-- Add role enum type and update users table
CREATE TYPE role_enum AS ENUM ('User', 'Admin', 'SuperAdmin');

-- Recreate users table with enum type for role
ALTER TABLE users
    ALTER COLUMN role TYPE role_enum USING role::role_enum;

-- Update constraint default
ALTER TABLE users
    ALTER COLUMN role SET DEFAULT 'user'::role_enum;
