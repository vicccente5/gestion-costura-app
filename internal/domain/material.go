// Package domain — modelos Material y MaterialPurchase.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Material representa un insumo del taller (tela, hilo, botones, etc.).
// El costo unitario se calcula con promedio ponderado móvil en cada compra
// para reflejar con precisión el costo real del stock disponible.
//
// DECISIÓN DE DISEÑO — Precios en enteros (CLP):
// Los precios se almacenan en pesos chilenos como enteros (int64).
// Ejemplo: $1.500 CLP se almacena como 1500.
// Esto evita los problemas de precisión de los números flotantes
// (0.1 + 0.2 = 0.30000000000000004 en float64).
type Material struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Nombre único por usuario — índice UNIQUE (user_id, nombre) en migración SQL.
	Nombre string `gorm:"not null;size:150" json:"nombre"`

	// Categoria agrupa los materiales (ej: "telas", "hilos", "botones").
	Categoria string `gorm:"size:100" json:"categoria,omitempty"`

	// Unidad de medida: "metros", "unidades", "rollos", "kilos", etc.
	Unidad string `gorm:"not null;size:50" json:"unidad"`

	// StockActual es la cantidad disponible en el taller.
	// CHECK (stock_actual >= 0) en la migración — nunca negativo.
	StockActual float64 `gorm:"not null;default:0" json:"stock_actual"`

	// StockMinimo es el umbral de alerta de bajo stock.
	// Si stock_actual <= stock_minimo → aparece en /alerts/low-stock.
	StockMinimo float64 `gorm:"not null;default:0" json:"stock_minimo"`

	// CostoUnitario es el costo por unidad en CLP (entero).
	// Se recalcula con promedio ponderado móvil en cada compra.
	// CHECK (costo_unitario >= 0) en la migración.
	CostoUnitario int64 `gorm:"not null;default:0" json:"costo_unitario"`

	// UserID vincula el material a la costurera dueña.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`

	// Purchases: historial de compras de este material.
	Purchases []MaterialPurchase `gorm:"foreignKey:MaterialID" json:"purchases,omitempty"`
}

// MaterialPurchase registra cada compra de un material.
// Es el historial que permite:
//   - Auditar de dónde vino el stock
//   - Recalcular el costo unitario por promedio ponderado móvil
//
// Fórmula del promedio ponderado móvil:
// nuevo_costo = (stock_actual * costo_anterior + cantidad * precio_unitario) / (stock_actual + cantidad)
// Más preciso que dividir el total porque considera el stock que ya existía.
type MaterialPurchase struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	// Cantidad comprada (en la misma unidad que el material).
	// CHECK (cantidad > 0) en la migración.
	Cantidad float64 `gorm:"not null" json:"cantidad"`

	// PrecioUnitario es el precio pagado por unidad en esta compra (CLP entero).
	// Puede diferir del CostoUnitario actual del material (que es el promedio).
	PrecioUnitario int64 `gorm:"not null" json:"precio_unitario"`

	// PrecioTotal = cantidad * precio_unitario (calculado al crear, no almacenado como float).
	PrecioTotal int64 `gorm:"not null" json:"precio_total"`

	// Fecha de la compra — por defecto es la fecha de creación del registro.
	Fecha time.Time `gorm:"not null" json:"fecha"`

	// Notas opcionales (proveedor, número de factura, etc.).
	Notas *string `gorm:"type:text" json:"notas,omitempty"`

	// MaterialID es la FK hacia el material comprado.
	MaterialID uuid.UUID `gorm:"type:uuid;not null;index" json:"material_id"`
	Material   Material  `gorm:"foreignKey:MaterialID" json:"-"`
}
