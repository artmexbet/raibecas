CREATE TYPE role_enum AS ENUM ('User', 'Admin', 'SuperAdmin');

UPDATE users
SET role = CASE role
    WHEN 'user' THEN 'User'
    WHEN 'admin' THEN 'Admin'
    WHEN 'superadmin' THEN 'SuperAdmin'
    ELSE role
END;

ALTER TABLE users
    ALTER COLUMN role DROP DEFAULT;

ALTER TABLE users
    ALTER COLUMN role TYPE role_enum USING role::role_enum;

ALTER TABLE users
    ALTER COLUMN role SET DEFAULT 'User'::role_enum;


