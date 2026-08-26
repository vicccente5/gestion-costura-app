// Package domain — modelos Order y OrderMaterial.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderStatus define los estados válidos de un encargo.
// Usar un tipo string personalizado previene errores de tipeo
// y hace el código más legible que usar constantes string sueltas.
type OrderStatus string

const (
	OrderStatusPendiente   OrderStatus = "pendiente"
	OrderStatusEnProgreso  OrderStatus = "en_progreso"
	OrderStatusCompletado  OrderStatus = "completado"
	OrderStatusEntregado   OrderStatus = "entregado" // Al llegar aquí se genera la Transaction automática
	OrderStatusCancelado   OrderStatus = "cancelado" // Estado terminal — restaura stock si aplica
)

// Order representa un trabajo o encargo de costura.
// Es el módulo central de la aplicación:
//   - Vincula materiales (descuenta inventario)
//   - Calcula rentabilidad (costo materiales + mano de obra vs precio de venta)
//   - Al marcarse como "entregado", genera automáticamente una Transaction de ingreso
//
// REGLA DE ORO: todos los queries de Order deben incluir WHERE user_id = ?
type Order struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Descripcion del trabajo a realizar.
	Descripcion string `gorm:"not null;type:text" json:"descripcion"`

	// Estado con CHECK constraint en la migración SQL.
	// Solo acepta los valores definidos en OrderStatus.
	Estado OrderStatus `gorm:"not null;default:'pendiente'" json:"estado"`

	// Horas trabajadas en el encargo.
	// CHECK (horas >= 0) en la migración.
	Horas float64 `gorm:"not null;default:0" json:"horas"`

	// TarifaHora es el precio por hora de mano de obra en CLP.
	// CHECK (tarifa_hora >= 0) en la migración.
	TarifaHora int64 `gorm:"not null;default:0" json:"tarifa_hora"`

	// PrecioVenta es el precio final cobrado a la clienta en CLP.
	// Si es 0 → el margen de ganancia devuelve null (no se puede calcular).
	// CHECK (precio_venta >= 0) en la migración.
	PrecioVenta int64 `gorm:"not null;default:0" json:"precio_venta"`

	// FechaEntrega es la fecha límite acordada con la clienta (opcional).
	FechaEntrega *time.Time `json:"fecha_entrega,omitempty"`

	// Notas adicionales sobre el encargo.
	Notas *string `gorm:"type:text" json:"notas,omitempty"`

	// ClientID es la FK hacia el cliente del encargo.
	ClientID uuid.UUID `gorm:"type:uuid;not null;index" json:"client_id"`
	Client   Client    `gorm:"foreignKey:ClientID" json:"client,omitempty"`

	// UserID vincula el encargo a la costurera dueña.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`

	// Materials: lista de materiales asignados al encargo.
	Materials []OrderMaterial `gorm:"foreignKey:OrderID" json:"materials,omitempty"`
}

// OrderMaterial es la tabla pivote entre Order y Material.
// Registra qué materiales se usaron en un encargo y en qué cantidad.
//
// DECISIÓN CRÍTICA — CostoUnitarioSnapshot:
// Al asignar un material, se guarda el costo que tenía EN ESE MOMENTO.
// Si el costo del material cambia después (nueva compra → nuevo promedio ponderado),
// la rentabilidad histórica del encargo permanece intacta con el precio original.
// Sin este snapshot, el recálculo de rentabilidad sería incorrecto.
type OrderMaterial struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Cantidad del material usada en el encargo.
	// CHECK (cantidad > 0) en la migración.
	Cantidad float64 `gorm:"not null" json:"cantidad"`

	// CostoUnitarioSnapshot: precio del material al momento de asignarlo.
	// Este valor es INMUTABLE — no cambia aunque el material sea más caro después.
	// Es la base para calcular costo_materiales = suma(cantidad * costo_unitario_snapshot)
	CostoUnitarioSnapshot int64 `gorm:"not null" json:"costo_unitario_snapshot"`

	// OrderID es la FK hacia el encargo.
	OrderID uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	Order   Order     `gorm:"foreignKey:OrderID" json:"-"`

	// MaterialID es la FK hacia el material.
	MaterialID uuid.UUID `gorm:"type:uuid;not null;index" json:"material_id"`
	Material   Material  `gorm:"foreignKey:MaterialID" json:"material,omitempty"`
}
