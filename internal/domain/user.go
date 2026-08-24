// Package domain contiene los modelos de datos (entidades) de la aplicación.
// Estos structs representan las tablas de la base de datos usando GORM.
// IMPORTANTE: Las migraciones se manejan con golang-migrate (archivos SQL en /migrations/),
// NO con AutoMigrate de GORM, para tener control total y reversibilidad.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User representa a una costurera dueña de un taller.
// Es la entidad raíz: todos los demás datos (clientes, materiales,
// encargos, transacciones) pertenecen a un User específico.
// Esto garantiza el aislamiento total de datos entre usuarios.
type User struct {
	// ID usa UUID en lugar de entero secuencial para evitar que
	// un usuario pueda predecir o enumerar IDs de otros usuarios.
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// DeletedAt habilita el soft delete de GORM:
	// en lugar de borrar el registro, marca la fecha de eliminación.
	// Los queries normales excluyen registros con DeletedAt != NULL.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Nombre visible de la costurera en la aplicación.
	Nombre string `gorm:"not null;size:100" json:"nombre"`

	// Email único (a nivel global) — se convierte a minúsculas antes de guardar.
	// El índice UNIQUE está definido en la migración SQL.
	Email string `gorm:"not null;uniqueIndex;size:255" json:"email"`

	// PasswordHash almacena el hash bcrypt. NUNCA se almacena la contraseña en texto plano.
	// El campo json:"-" evita que se serialice accidentalmente en respuestas JSON.
	PasswordHash string `gorm:"not null" json:"-"`

	// RefreshTokens: relación uno-a-muchos con los tokens de refresco activos.
	// Un usuario puede tener múltiples sesiones (dispositivos) simultáneas.
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"-"`
}

// RefreshToken almacena los tokens de refresco válidos en la DB.
// Esto permite invalidar sesiones específicas en el logout (logout seguro).
// Sin esta tabla, sería imposible revocar tokens antes de su expiración.
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	// Token es el JWT de refresco en texto plano (ya está firmado, no necesita hash).
	Token string `gorm:"not null;uniqueIndex;type:text" json:"-"`

	// ExpiresAt permite limpiar tokens expirados de la DB con un cron job.
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`

	// RevokedAt es NULL si el token está activo. Se llena al hacer logout.
	// Esto implementa la revocación sin borrar el registro (para auditoría).
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	// UserID es la FK hacia el usuario dueño del token.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"-"`
}
