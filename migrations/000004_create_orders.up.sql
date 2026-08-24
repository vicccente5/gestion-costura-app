-- Migración 000004 — Crear tablas orders y order_materials
-- Depende de: users (000001), clients (000002), materials (000003)

CREATE TABLE orders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,

    descripcion   TEXT NOT NULL
                    CONSTRAINT orders_descripcion_not_empty CHECK (LENGTH(TRIM(descripcion)) > 0),

    -- Estado con valores válidos explícitos
    estado        VARCHAR(20) NOT NULL DEFAULT 'pendiente'
                    CONSTRAINT orders_estado_valid
                    CHECK (estado IN ('pendiente', 'en_progreso', 'completado', 'entregado')),

    horas         NUMERIC(8, 2) NOT NULL DEFAULT 0
                    CONSTRAINT orders_horas_positive CHECK (horas >= 0),

    tarifa_hora   BIGINT NOT NULL DEFAULT 0
                    CONSTRAINT orders_tarifa_positive CHECK (tarifa_hora >= 0),

    -- 0 = precio no asignado → margen_porcentaje devuelve null (no error 500)
    precio_venta  BIGINT NOT NULL DEFAULT 0
                    CONSTRAINT orders_precio_positive CHECK (precio_venta >= 0),

    fecha_entrega TIMESTAMPTZ,
    notas         TEXT,

    client_id     UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    user_id       UUID NOT NULL REFERENCES users(id)   ON DELETE CASCADE
);

CREATE INDEX orders_user_id_idx    ON orders (user_id);
CREATE INDEX orders_client_id_idx  ON orders (client_id);
CREATE INDEX orders_estado_idx     ON orders (estado);

-- Tabla pivote: materiales asignados a cada encargo
CREATE TABLE order_materials (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    cantidad                 NUMERIC(12, 3) NOT NULL
                               CONSTRAINT order_materials_cantidad_positive CHECK (cantidad > 0),

    -- SNAPSHOT del costo unitario al momento de asignar el material.
    -- INMUTABLE: no cambia aunque el material sea más caro después.
    -- Garantiza que la rentabilidad histórica del encargo sea siempre correcta.
    costo_unitario_snapshot  BIGINT NOT NULL
                               CONSTRAINT order_materials_snapshot_positive CHECK (costo_unitario_snapshot >= 0),

    order_id    UUID NOT NULL REFERENCES orders(id)    ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE RESTRICT
);

CREATE INDEX order_materials_order_id_idx    ON order_materials (order_id);
CREATE INDEX order_materials_material_id_idx ON order_materials (material_id);
