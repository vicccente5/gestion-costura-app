// Package main — punto de entrada de la aplicación Gestión Costura App.
// Responsabilidades de main:
//   1. Cargar configuración
//   2. Ejecutar migraciones
//   3. Conectar a la DB
//   4. Construir el árbol de dependencias (repository → service → handler → router)
//   5. Levantar el servidor HTTP
//
// main.go es el único lugar donde se ensamblan las dependencias.
// Esto implementa el patrón "Composition Root" — toda la inyección de dependencias
// ocurre en un solo punto, manteniendo el resto del código desacoplado.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vicccente5/gestion-costura-app/config"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/router"
	"github.com/vicccente5/gestion-costura-app/internal/service"
)

func main() {
	// Configurar zerolog — salida legible en desarrollo, JSON en producción
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().Msg("🧵 Iniciando Gestión Costura App...")

	// 1. Cargar configuración desde .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Error cargando configuración")
	}
	log.Info().
		Str("port", cfg.Server.Port).
		Str("mode", cfg.Server.GinMode).
		Str("db", cfg.Database.Name).
		Msg("✅ Configuración cargada")

	// 2. Ejecutar migraciones — ANTES de conectar GORM
	log.Info().Msg("Ejecutando migraciones...")
	if err := config.RunMigrations(cfg.Database.MigrateURL()); err != nil {
		log.Fatal().Err(err).Msg("Error ejecutando migraciones")
	}
	log.Info().Msg("✅ Migraciones aplicadas")

	// 3. Conectar a PostgreSQL
	db, err := config.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Error conectando a PostgreSQL")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	log.Info().Msg("✅ Conexión a PostgreSQL establecida")

	// 4. Construir árbol de dependencias (Composition Root)
	//    repository ← recibe *gorm.DB
	//    service    ← recibe repository + config
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, cfg)

	clientRepo := repository.NewClientRepository(db)
	clientSvc := service.NewClientService(clientRepo)

	materialRepo := repository.NewMaterialRepository(db)
	materialSvc := service.NewMaterialService(materialRepo)

	orderRepo := repository.NewOrderRepository(db)
	orderSvc := service.NewOrderService(orderRepo, clientRepo, materialRepo)

	txRepo := repository.NewTransactionRepository(db)
	txSvc := service.NewTransactionService(txRepo)

	reportSvc := service.NewReportService(db)

	// 5. Configurar modo de Gin (debug en dev, release en prod)
	gin.SetMode(cfg.Server.GinMode)

	// 6. Configurar y arrancar el router
	r := router.Setup(authSvc, clientSvc, materialSvc, orderSvc, txSvc, reportSvc)

	// 7. Servidor HTTP con Graceful Shutdown
	//    Graceful Shutdown: al recibir SIGTERM/SIGINT, espera hasta 10 segundos
	//    para que las peticiones activas terminen antes de apagar el servidor.
	//    Sin esto, un `docker stop` mataría peticiones a la mitad.
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Levantar el servidor en una goroutine separada
	go func() {
		log.Info().
			Str("addr", "http://localhost:"+cfg.Server.Port).
			Msg("🚀 Servidor HTTP iniciado")

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Error en el servidor HTTP")
		}
	}()

	// Esperar señal de apagado (Ctrl+C o docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Apagando el servidor...")

	// Dar 10 segundos para que terminen las peticiones activas
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error durante el graceful shutdown")
	}

	log.Info().Msg("✅ Servidor detenido correctamente")
}
