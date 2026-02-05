-- Rollback role enum
ALTER TABLE users
ALTER COLUMN role TYPE VARCHAR(50) USING role::text;

DROP TYPE IF EXISTS role_enum;
