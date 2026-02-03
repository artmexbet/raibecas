-- Add role enum type to auth database
CREATE TYPE role_enum AS ENUM ('User', 'Admin', 'SuperAdmin');

-- Update users table role column to use enum
ALTER TABLE users
    ALTER COLUMN role DROP DEFAULT;
ALTER TABLE users
ALTER COLUMN role TYPE role_enum USING role::role_enum;

ALTER TABLE users
ALTER COLUMN role SET DEFAULT 'User'::role_enum;
