-- ============================================
-- Инициализация баз данных для микросервисов
-- ============================================

-- Создание пользователей для каждого сервиса
-- Выполнять под суперпользователем PostgreSQL
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_user') THEN
        CREATE ROLE auth_user LOGIN PASSWORD 'auth_password_secure_2026';
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'users_user') THEN
        CREATE ROLE users_user LOGIN PASSWORD 'users_password_secure_2026';
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'documents_user') THEN
        CREATE ROLE documents_user LOGIN PASSWORD 'documents_password_secure_2026';
    END IF;
END
$$;

-- Создание баз данных для сервисов
SELECT 'CREATE DATABASE auth_db OWNER auth_user'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'auth_db')\gexec

SELECT 'CREATE DATABASE users_db OWNER users_user'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'users_db')\gexec

SELECT 'CREATE DATABASE documents_db OWNER documents_user'
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'documents_db')\gexec

SELECT 'CREATE DATABASE raibecas_chat OWNER raibecas'
    WHERE NOT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'raibecas_chat')\gexec

-- Пользователи уже созданы выше, чтобы сразу назначить их владельцами БД

-- ============================================
-- Выдача прав доступа
-- ============================================

-- Права для auth_user на базу auth_db
ALTER DATABASE auth_db OWNER TO auth_user;
GRANT ALL PRIVILEGES ON DATABASE auth_db TO auth_user;

\connect auth_db

ALTER SCHEMA public OWNER TO auth_user;
GRANT USAGE, CREATE ON SCHEMA public TO auth_user;
GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER
    ON ALL TABLES IN SCHEMA public TO auth_user;
GRANT USAGE, SELECT, UPDATE
    ON ALL SEQUENCES IN SCHEMA public TO auth_user;
ALTER DEFAULT PRIVILEGES FOR ROLE auth_user IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER
    ON TABLES TO auth_user;
ALTER DEFAULT PRIVILEGES FOR ROLE auth_user IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE
    ON SEQUENCES TO auth_user;

-- Права для users_user на базу users_db
\connect users_db

ALTER DATABASE users_db OWNER TO users_user;
GRANT ALL PRIVILEGES ON DATABASE users_db TO users_user;
ALTER SCHEMA public OWNER TO users_user;
GRANT USAGE, CREATE ON SCHEMA public TO users_user;
GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER
    ON ALL TABLES IN SCHEMA public TO users_user;
GRANT USAGE, SELECT, UPDATE
    ON ALL SEQUENCES IN SCHEMA public TO users_user;
ALTER DEFAULT PRIVILEGES FOR ROLE users_user IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER
    ON TABLES TO users_user;
ALTER DEFAULT PRIVILEGES FOR ROLE users_user IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE
    ON SEQUENCES TO users_user;

-- Права для documents_user на базу documents_db
\connect documents_db

ALTER DATABASE documents_db OWNER TO documents_user;
GRANT ALL PRIVILEGES ON DATABASE documents_db TO documents_user;
ALTER SCHEMA public OWNER TO documents_user;
GRANT USAGE, CREATE ON SCHEMA public TO documents_user;
GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER
    ON ALL TABLES IN SCHEMA public TO documents_user;
GRANT USAGE, SELECT, UPDATE
    ON ALL SEQUENCES IN SCHEMA public TO documents_user;
ALTER DEFAULT PRIVILEGES FOR ROLE documents_user IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER
    ON TABLES TO documents_user;
ALTER DEFAULT PRIVILEGES FOR ROLE documents_user IN SCHEMA public
    GRANT USAGE, SELECT, UPDATE
    ON SEQUENCES TO documents_user;

\connect postgres

-- ============================================
-- Информация о созданных ресурсах
-- ============================================

SELECT 'Databases created: auth_db, users_db, documents_db' AS status;
SELECT 'Users created: auth_user, users_user, documents_user' AS status;
SELECT 'All privileges granted successfully' AS status;
