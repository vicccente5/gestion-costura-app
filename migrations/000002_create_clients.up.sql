-- Migración 000002 — Crear tabla clients
-- Depende de: users (000001)

CREATE TABLE clients (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    nombre     VARCHAR(150) NOT NULL
                 CONSTRAINT clients_nombre_not_empty CHECK (LENGTH(TRIM(nombre)) > 0),

    -- Teléfono opcional
    telefono   VARCHAR(20),

    -- Email opcional — si se proporciona, debe tener formato válido y estar en minúsculas
    email      VARCHAR(255)
                 CONSTRAINT clients_email_format
                 CHECK (email IS NULL OR email ~* '^[^@\s]+@[^@\s]+\.[^@\s]+$'),

    -- FK hacia la costurera dueña del cliente
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- Índice UNIQUE compuesto (user_id, email):
-- El mismo email puede existir para distintos usuarios (dos costureras pueden tener
-- la misma clienta), pero NO para el mismo usuario.
-- Es parcial: excluye registros con soft delete y emails NULL.
CREATE UNIQUE INDEX clients_user_email_unique_idx
    ON clients (user_id, email)
    WHERE deleted_at IS NULL AND email IS NOT NULL;

CREATE INDEX clients_user_id_idx ON clients (user_id);
