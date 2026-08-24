// Package main — punto de entrada de la aplicación Gestión Costura App.
// En la Fase 1 solo levanta la conexión a la DB y ejecuta las migraciones.
// El servidor HTTP se levanta en la Fase 2 (junto con auth y los primeros endpoints).
package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vicccente5/gestion-costura-app/config"
)

func main() {
	// Configurar zerolog — salida legible en desarrollo, JSON en producción
	// En Fase 7 se añade el middleware de logging de requests HTTP
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("🧵 Iniciando Gestión Costura App...")

	// 1. Cargar configuración desde .env / variables de entorno
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Error cargando configuración")
	}
	log.Info().
		Str("port", cfg.Server.Port).
		Str("gin_mode", cfg.Server.GinMode).
		Str("db_host", cfg.Database.Host).
		Str("db_name", cfg.Database.Name).
		Msg("✅ Configuración cargada")

	// 2. Ejecutar migraciones ANTES de abrir conexión GORM
	// Esto garantiza que las tablas existen cuando los repositories las necesiten
	log.Info().Msg("Ejecutando migraciones de base de datos...")
	if err := config.RunMigrations(cfg.Database.MigrateURL()); err != nil {
		log.Fatal().Err(err).Msg("Error ejecutando migraciones")
	}
	log.Info().Msg("✅ Migraciones aplicadas")

	// 3. Conectar a PostgreSQL con GORM
	db, err := config.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Error conectando a la base de datos")
	}

	// Obtener sql.DB subyacente para cerrar la conexión al salir
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Error obteniendo sql.DB")
	}
	defer sqlDB.Close()

	log.Info().Msg("✅ Conexión a PostgreSQL establecida")

	// ============================================================
	// Fase 1 completa — el servidor HTTP se levanta en Fase 2
	// ============================================================
	fmt.Printf("\n🎉 Base de datos lista. Tablas creadas con golang-migrate.\n")
	fmt.Printf("   Conectar DBeaver a: %s:%s/%s\n",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)
	fmt.Printf("   El servidor HTTP se implementará en la Fase 2.\n\n")

	// Mantener la aplicación viva para verificar la conexión en Fase 1
	// En Fase 2 esto se reemplaza por el servidor Gin
	log.Info().Msg("Presiona Ctrl+C para salir")
	select {} // Bloquear indefinidamente
}
