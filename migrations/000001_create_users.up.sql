-- Migración 000001 — Crear tabla users y refresh_tokens
-- Habilitar la extensión para gen_random_uuid() y para el CHECK de email con regex
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    nombre        VARCHAR(100) NOT NULL
                    CONSTRAINT users_nombre_not_empty CHECK (LENGTH(TRIM(nombre)) > 0),

    -- Email único globalmente y siempre en minúsculas (sanitizado en el service antes de guardar)
    email         VARCHAR(255) NOT NULL UNIQUE
                    CONSTRAINT users_email_format CHECK (email ~* '^[^@\s]+@[^@\s]+\.[^@\s]+$'),

    password_hash TEXT NOT NULL
);

-- Índice parcial: excluye usuarios con soft delete del índice de email único
-- Sin esto, restaurar un usuario eliminado podría fallar si el email ya fue reutilizado
CREATE UNIQUE INDEX users_email_active_idx ON users (email) WHERE deleted_at IS NULL;

-- Tabla de refresh tokens para logout seguro y revocación de sesiones
CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    token      TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,

    -- NULL = token activo. Fecha = token revocado (logout o Token Rotation)
    revoked_at TIMESTAMPTZ,

    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_token_idx   ON refresh_tokens (token);
