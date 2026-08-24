// Package config maneja la carga y validación de variables de entorno.
// Centralizar la configuración aquí evita que las variables de entorno
// estén dispersas por todo el código (patrón de diseño: Config Object).
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config contiene toda la configuración de la aplicación.
// Al centralizar aquí, un error de configuración falla en el arranque
// (fail-fast) y no en medio de una petición de usuario.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
}

// ServerConfig agrupa la configuración del servidor HTTP.
type ServerConfig struct {
	Port    string
	GinMode string
}

// DatabaseConfig agrupa los parámetros de conexión a PostgreSQL.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN construye el Data Source Name para GORM desde los campos individuales.
// Formato: "host=... user=... password=... dbname=... port=... sslmode=..."
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		d.Host, d.User, d.Password, d.Name, d.Port, d.SSLMode,
	)
}

// MigrateURL construye la URL para golang-migrate.
// Formato: "postgres://user:password@host:port/dbname?sslmode=..."
func (d DatabaseConfig) MigrateURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// JWTConfig agrupa los parámetros de autenticación JWT.
type JWTConfig struct {
	Secret              string
	AccessExpiryMinutes int
	RefreshExpiryDays   int
}

// Load carga la configuración desde el archivo .env y variables de entorno del sistema.
// Las variables del sistema tienen precedencia sobre el .env (comportamiento de godotenv).
// Si una variable obligatoria no está definida, retorna un error descriptivo.
func Load() (*Config, error) {
	// Intentar cargar .env (ignora el error si el archivo no existe —
	// en producción con Docker las variables vienen del entorno del contenedor)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     mustGetEnv("DB_USER"),
			Password: mustGetEnv("DB_PASSWORD"),
			Name:     mustGetEnv("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:              mustGetEnv("JWT_SECRET"),
			AccessExpiryMinutes: getEnvInt("JWT_ACCESS_EXPIRY_MINUTES", 15),
			RefreshExpiryDays:   getEnvInt("JWT_REFRESH_EXPIRY_DAYS", 7),
		},
	}

	// Validar que el JWT secret tenga una longitud mínima segura
	if len(cfg.JWT.Secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET debe tener al menos 32 caracteres (tiene %d)", len(cfg.JWT.Secret))
	}

	return cfg, nil
}

// getEnv retorna el valor de la variable de entorno o un valor por defecto.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// mustGetEnv retorna el valor de la variable o falla si no está definida.
// Se usa para variables obligatorias sin valor por defecto seguro.
func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("variable de entorno obligatoria no definida: %s", key))
	}
	return value
}

// getEnvInt retorna el valor entero de una variable de entorno o el valor por defecto.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
