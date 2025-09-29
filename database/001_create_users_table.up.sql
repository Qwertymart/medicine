-- Включаем расширение для UUID
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Создаем таблицу пользователей
CREATE TABLE users (
                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                       count_pictures INTEGER NOT NULL DEFAULT 0,
                       name VARCHAR(100) NOT NULL,
                       last_name VARCHAR(100) NOT NULL,
                       email VARCHAR(255) UNIQUE NOT NULL,
                       password_hash VARCHAR(255) NOT NULL
);

-- Создаем индексы
CREATE INDEX idx_users_email ON users(email);

