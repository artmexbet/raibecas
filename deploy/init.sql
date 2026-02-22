-- ============================================
-- Инициализация баз данных для микросервисов
-- ============================================

-- Создание базы данных для сервиса auth
CREATE DATABASE IF NOT EXISTS auth_db;

-- Создание базы данных для сервиса users
CREATE DATABASE IF NOT EXISTS users_db;

-- Создание базы данных для сервиса documents
CREATE DATABASE IF NOT EXISTS documents_db;

-- ============================================
-- Создание пользователей для каждого сервиса
-- ============================================

-- Пользователь для auth сервиса
CREATE USER IF NOT EXISTS 'auth_user'@'%' IDENTIFIED BY 'auth_password_secure_2026';

-- Пользователь для users сервиса
CREATE USER IF NOT EXISTS 'users_user'@'%' IDENTIFIED BY 'users_password_secure_2026';

-- Пользователь для documents сервиса
CREATE USER IF NOT EXISTS 'documents_user'@'%' IDENTIFIED BY 'documents_password_secure_2026';

-- ============================================
-- Выдача прав доступа
-- ============================================

-- Права для auth_user на базу auth_db
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, DROP, INDEX, REFERENCES
ON auth_db.* TO 'auth_user'@'%';

-- Права для users_user на базу users_db
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, DROP, INDEX, REFERENCES
ON users_db.* TO 'users_user'@'%';

-- Права для documents_user на базу documents_db
GRANT ALL PRIVILEGES
    ON DATABASE corpus_db TO "corpus_service";

-- ============================================
-- Информация о созданных ресурсах
-- ============================================

SELECT 'Databases created: auth_db, users_db, documents_db' AS status;
SELECT 'Users created: auth_user, users_user, documents_user' AS status;
SELECT 'All privileges granted successfully' AS status;
