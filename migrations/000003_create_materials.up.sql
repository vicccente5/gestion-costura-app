-- Migración 000003 — Crear tablas materials y material_purchases
-- Depende de: users (000001)

CREATE TABLE materials (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,

    nombre         VARCHAR(150) NOT NULL
                     CONSTRAINT materials_nombre_not_empty CHECK (LENGTH(TRIM(nombre)) > 0),

    categoria      VARCHAR(100),

    -- Unidad de medida: "metros", "unidades", "rollos", etc.
    unidad         VARCHAR(50) NOT NULL
                     CONSTRAINT materials_unidad_not_empty CHECK (LENGTH(TRIM(unidad)) > 0),

    -- Stock nunca puede ser negativo
    stock_actual   NUMERIC(12, 3) NOT NULL DEFAULT 0
                     CONSTRAINT materials_stock_actual_positive CHECK (stock_actual >= 0),

    stock_minimo   NUMERIC(12, 3) NOT NULL DEFAULT 0
                     CONSTRAINT materials_stock_minimo_positive CHECK (stock_minimo >= 0),

    -- Costo en CLP (entero) — calculado con promedio ponderado móvil
    costo_unitario BIGINT NOT NULL DEFAULT 0
                     CONSTRAINT materials_costo_positive CHECK (costo_unitario >= 0),

    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- Índice UNIQUE (user_id, nombre): no puede haber dos materiales con el mismo nombre
-- para la misma costurera. Parcial para manejar soft delete correctamente.
CREATE UNIQUE INDEX materials_user_nombre_unique_idx
    ON materials (user_id, nombre)
    WHERE deleted_at IS NULL;

CREATE INDEX materials_user_id_idx ON materials (user_id);

-- Historial de compras de materiales
CREATE TABLE material_purchases (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    cantidad        NUMERIC(12, 3) NOT NULL
                      CONSTRAINT purchases_cantidad_positive CHECK (cantidad > 0),

    -- Precio pagado por unidad en esta compra (puede diferir del promedio)
    precio_unitario BIGINT NOT NULL
                      CONSTRAINT purchases_precio_unitario_positive CHECK (precio_unitario > 0),

    -- Total = cantidad * precio_unitario (calculado y almacenado para consultas)
    precio_total    BIGINT NOT NULL
                      CONSTRAINT purchases_precio_total_positive CHECK (precio_total > 0),

    fecha           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notas           TEXT,

    material_id     UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE
);

CREATE INDEX material_purchases_material_id_idx ON material_purchases (material_id);
