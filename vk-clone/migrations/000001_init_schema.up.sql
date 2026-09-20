-- Включение расширения для работы с UUID (если потребуется) или генерации хешей
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. ПОЛЬЗОВАТЕЛИ
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR(32) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. ТРЕДЫ (ФОРУМ)
CREATE TABLE threads (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. ОТВЕТЫ В ТРЕДАХ (ВЕТКИ ОБСУЖДЕНИЙ)
CREATE TABLE thread_replies (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    thread_id BIGINT NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    parent_reply_id BIGINT REFERENCES thread_replies(id) ON DELETE CASCADE, -- NULL если ответ в корень треда
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. ПОСТЫ (ЛЕНТА В СТИЛЕ ВК)
CREATE TABLE posts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    media_urls TEXT[] DEFAULT '{}', -- Массив ссылок на прикрепленные медиа
    views_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. ЛАЙКИ К ПОСТАМ
CREATE TABLE post_likes (
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

-- 6. КОММЕНТАРИИ К ПОСТАМ
CREATE TABLE post_comments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES post_comments(id) ON DELETE CASCADE, -- Для древовидных комментариев
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 7. ЧАТ-КОМНАТЫ
CREATE TABLE chat_rooms (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100), -- NULL для личных диалогов (DM)
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Участники чатов (многие-ко-многим)
CREATE TABLE chat_room_members (
    room_id BIGINT NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);

-- 8. СООБЩЕНИЯ ЧАТА
CREATE TABLE chat_messages (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    room_id BIGINT NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    media_url TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ИНДЕКСЫ ДЛЯ ОПТИМИЗАЦИИ ВЫБОРКИ
CREATE INDEX idx_threads_created_at ON threads(created_at DESC);
CREATE INDEX idx_thread_replies_thread_id ON thread_replies(thread_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_post_comments_post_id ON post_comments(post_id);
CREATE INDEX idx_chat_messages_room_id_created_at ON chat_messages(room_id, created_at DESC);
CREATE INDEX idx_chat_room_members_user_id ON chat_room_members(user_id);