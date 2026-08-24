// Package domain — modelo Client (cliente del taller de costura).
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Client representa a un cliente de la costurera.
// Está separado del User (costurera) para:
//   - Tener historial de encargos por cliente
//   - Permitir búsqueda y filtrado real
//   - Evitar texto libre en los encargos (que genera inconsistencias)
//
// La unicidad del email es POR usuario, no global:
// dos costureras distintas pueden tener la misma clienta.
// El índice UNIQUE (user_id, email) está en la migración SQL.
type Client struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Nombre es obligatorio — validado con go-playground/validator en el handler.
	Nombre string `gorm:"not null;size:150" json:"nombre"`

	// Telefono es opcional — nil si no se proporciona.
	Telefono *string `gorm:"size:20" json:"telefono,omitempty"`

	// Email es opcional — nil si no se proporciona.
	// Si se proporciona, se convierte a minúsculas antes de guardar.
	// El índice UNIQUE (user_id, email) previene duplicados para el mismo usuario.
	Email *string `gorm:"size:255" json:"email,omitempty"`

	// UserID es la FK que vincula el cliente a su costurera dueña.
	// TODOS los queries deben incluir WHERE user_id = ? para aislar datos.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`

	// Orders: relación uno-a-muchos para obtener el historial de encargos.
	Orders []Order `gorm:"foreignKey:ClientID" json:"orders,omitempty"`
}
