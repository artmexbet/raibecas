DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'role_enum') THEN
            CREATE TYPE role_enum AS ENUM ('User', 'Admin', 'SuperAdmin');

            UPDATE users
            SET role = CASE role
                           WHEN 'user' THEN 'User'
                           WHEN 'admin' THEN 'Admin'
                           WHEN 'superadmin' THEN 'SuperAdmin'
                           ELSE role
                END;
        END IF;
    END
$$;

ALTER TABLE IF EXISTS users
    ALTER COLUMN role DROP DEFAULT;

ALTER TABLE IF EXISTS users
    ALTER COLUMN role TYPE role_enum USING role::role_enum;

ALTER TABLE IF EXISTS users
    ALTER COLUMN role SET DEFAULT 'User'::role_enum;


