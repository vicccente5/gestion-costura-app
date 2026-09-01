// Package config — conexión a la base de datos y ejecución de migraciones.
package config

import (
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // driver postgres para migrate
	_ "github.com/golang-migrate/migrate/v4/source/file"       // leer migraciones desde archivos
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabase crea y configura la conexión GORM a PostgreSQL.
// Retorna la instancia *gorm.DB lista para usar en los repositories.
//
// IMPORTANTE: No se usa AutoMigrate aquí — las migraciones se ejecutan
// con golang-migrate (función RunMigrations) que usa SQL versionado.
func NewDatabase(cfg DatabaseConfig) (*gorm.DB, error) {
	// Configurar el logger de GORM según el modo de la aplicación
	gormLogger := logger.Default.LogMode(logger.Info)

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormLogger,
		// PrepareStmt cachea las consultas preparadas → mejor rendimiento
		PrepareStmt: true,
		// TranslateError convierte errores de PG a errores GORM tipados
		// (ej: ErrRecordNotFound, ErrDuplicatedKey)
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("error conectando a PostgreSQL: %w", err)
	}

	// Obtener la conexión SQL subyacente para configurar el pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo sql.DB: %w", err)
	}

	// Configurar el pool de conexiones para evitar DoS por conexiones agotadas
	sqlDB.SetMaxOpenConns(25)  // Máximo de conexiones simultáneas abiertas
	sqlDB.SetMaxIdleConns(10)  // Conexiones mantenidas en espera (reutilizables)

	// Verificar que la conexión está activa
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("error haciendo ping a PostgreSQL: %w", err)
	}

	// Configurar Connection Pool (Fase 10 - Seguridad y Estabilidad)
	sqlDB, dbErr := db.DB()
	if dbErr == nil {
		sqlDB.SetMaxOpenConns(25) // Evita agotar conexiones en Postgres bajo ataques DoS
		sqlDB.SetMaxIdleConns(5)  // Mantiene 5 conexiones vivas para respuestas rápidas
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
	}

	return db, nil
}

// RunMigrations ejecuta las migraciones pendientes usando golang-migrate.
// Lee los archivos SQL de la carpeta migrations/ en orden numérico.
//
// Comportamiento:
//   - Si no hay migraciones pendientes → no hace nada (idempotente)
//   - Si una migración falla → retorna error y la migración queda en estado "dirty"
//   - Para limpiar un estado dirty: migrate -database $URL force <versión>
func RunMigrations(migrateURL string) error {
	m, err := migrate.New(
		"file://migrations", // Ruta relativa a los archivos SQL
		migrateURL,
	)
	if err != nil {
		return fmt.Errorf("error inicializando migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error ejecutando migraciones: %w", err)
	}

	return nil
}
