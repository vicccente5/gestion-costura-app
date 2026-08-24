-- Migración 000005 — Crear tabla transactions
-- Depende de: users (000001), orders (000004)

CREATE TABLE transactions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,

    -- Solo "ingreso" o "gasto"
    tipo        VARCHAR(10) NOT NULL
                  CONSTRAINT transactions_tipo_valid CHECK (tipo IN ('ingreso', 'gasto')),

    -- Siempre positivo — el tipo determina si suma o resta del balance
    monto       BIGINT NOT NULL
                  CONSTRAINT transactions_monto_positive CHECK (monto > 0),

    descripcion TEXT NOT NULL
                  CONSTRAINT transactions_desc_not_empty CHECK (LENGTH(TRIM(descripcion)) > 0),

    categoria   VARCHAR(100),

    -- Fecha del movimiento (puede ser distinta de created_at si se registra tarde)
    fecha       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- "manual" = registrado por la costurera
    -- "order"  = generado automáticamente al entregar un encargo
    source      VARCHAR(10) NOT NULL DEFAULT 'manual'
                  CONSTRAINT transactions_source_valid CHECK (source IN ('manual', 'order')),

    -- order_id es NOT NULL cuando source='order', NULL cuando source='manual'
    -- Esta regla de consistencia se valida también en el service de Go
    order_id    UUID REFERENCES orders(id) ON DELETE SET NULL,

    CONSTRAINT transactions_order_consistency
        CHECK (
            (source = 'order' AND order_id IS NOT NULL)
            OR
            (source = 'manual' AND order_id IS NULL)
        ),

    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX transactions_user_id_idx  ON transactions (user_id);
CREATE INDEX transactions_order_id_idx ON transactions (order_id);
CREATE INDEX transactions_fecha_idx    ON transactions (fecha);
CREATE INDEX transactions_source_idx   ON transactions (source);
