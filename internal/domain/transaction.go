// Package domain — modelo Transaction (flujo de caja).
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransactionType define si es un ingreso o un gasto.
type TransactionType string

const (
	TransactionTypeIngreso TransactionType = "ingreso"
	TransactionTypeGasto   TransactionType = "gasto"
)

// TransactionSource identifica el origen de la transacción.
// DECISIÓN CRÍTICA — El campo Source previene duplicados en el balance mensual:
//
//   Sin Source: si la costurera cobra un encargo de $10.000 y también lo registra
//   manualmente como ingreso, el balance contaría $20.000 (duplicado).
//
//   Con Source:
//   - "order"  → generado automáticamente al marcar el encargo como "entregado"
//   - "manual" → ingresado directamente por la costurera (gastos, otros ingresos)
//
//   La UI puede mostrar de dónde vino cada transacción y el balance es confiable.
type TransactionSource string

const (
	TransactionSourceManual TransactionSource = "manual"
	TransactionSourceOrder  TransactionSource = "order"
)

// Transaction registra un movimiento financiero del taller.
// Puede ser:
//   - Un ingreso manual (ej: venta de telas sobrantes)
//   - Un gasto manual (ej: compra de máquina de coser)
//   - Un ingreso automático al entregar un encargo (source="order")
//
// Las transacciones source="order" NO pueden crearse, editarse ni eliminarse
// manualmente desde la API — solo el sistema las genera.
type Transaction struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Tipo con CHECK constraint en migración: solo "ingreso" o "gasto".
	Tipo TransactionType `gorm:"not null" json:"tipo"`

	// Monto en CLP (entero). CHECK (monto > 0) en migración — siempre positivo.
	// El tipo (ingreso/gasto) determina si suma o resta del balance.
	Monto int64 `gorm:"not null" json:"monto"`

	// Descripcion explica el concepto de la transacción.
	Descripcion string `gorm:"not null;type:text" json:"descripcion"`

	// Categoria agrupa transacciones (ej: "arriendo", "insumos", "servicios").
	Categoria *string `gorm:"size:100" json:"categoria,omitempty"`

	// Fecha de la transacción — puede diferir de created_at si se registra tarde.
	Fecha time.Time `gorm:"not null" json:"fecha"`

	// Source indica el origen: "manual" o "order".
	// CHECK (source IN ('manual', 'order')) en migración.
	Source TransactionSource `gorm:"not null;default:'manual'" json:"source"`

	// OrderID es nullable — solo tiene valor cuando source="order".
	// Vincula la transacción al encargo que la generó para trazabilidad.
	// Regla: si source='order' entonces order_id NO puede ser NULL.
	OrderID *uuid.UUID `gorm:"type:uuid;index" json:"order_id,omitempty"`
	Order   *Order     `gorm:"foreignKey:OrderID" json:"order,omitempty"`

	// UserID vincula la transacción a la costurera dueña.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`
}
