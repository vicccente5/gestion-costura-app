// Package router — configuración de rutas y separación de rutas públicas vs protegidas.
// Toda la configuración de Gin está centralizada aquí para facilitar el mantenimiento.
// Regla: NUNCA registrar rutas directamente en main.go.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/vicccente5/gestion-costura-app/internal/handler"
	"github.com/vicccente5/gestion-costura-app/internal/middleware"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// Setup configura el router Gin con todos los middlewares globales y rutas.
// Recibe los servicios por inyección de dependencias.
func Setup(authSvc service.AuthService, clientSvc service.ClientService, materialSvc service.MaterialService) *gin.Engine {
	// Crear el engine de Gin (ya sin el logger de Gin — usamos zerolog en main)
	r := gin.New()
	r.Use(gin.Recovery()) // Recupera panics y retorna 500 en lugar de crashear

	// Inicializar handlers
	authHandler := handler.NewAuthHandler(authSvc)
	clientHandler := handler.NewClientHandler(clientSvc)
	materialHandler := handler.NewMaterialHandler(materialSvc)

	// Rate limiter para login — instancia única compartida entre todos los workers de Gin
	loginLimiter := middleware.LoginRateLimiter()

	// ──────────────────────────────────────────────────────────
	// RUTAS PÚBLICAS — Sin autenticación requerida
	// ──────────────────────────────────────────────────────────
	public := r.Group("/api/v1")
	{
		// Información de la versión — útil para el cliente Flutter saber con qué versión habla
		public.GET("/version", func(c *gin.Context) {
			utils.OK(c, "API funcionando", gin.H{
				"version": "1.0.0",
				"name":    "Gestión Costura API",
			})
		})

		// Health check — para monitoreo del servidor y docker healthcheck
		public.GET("/health", func(c *gin.Context) {
			utils.OK(c, "OK", nil)
		})

		// Autenticación — rutas públicas
		auth := public.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", loginLimiter.Middleware(), authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}
	}

	// ──────────────────────────────────────────────────────────
	// RUTAS PROTEGIDAS — Requieren JWT válido
	// Se añaden más rutas en fases 3-8
	// ──────────────────────────────────────────────────────────
	protected := r.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(authSvc))
	{
		// Endpoint de prueba para verificar que el middleware funciona
		protected.GET("/me", func(c *gin.Context) {
			userID := middleware.GetUserID(c)
			utils.OK(c, "Token válido", gin.H{"user_id": userID})
		})

		// Módulo de Clientes (Fase 3)
		clients := protected.Group("/clients")
		{
			clients.GET("", clientHandler.GetAll)
			clients.POST("", clientHandler.Create)
			clients.GET("/:id", clientHandler.GetByID)
			clients.PUT("/:id", clientHandler.Update)
			clients.DELETE("/:id", clientHandler.Delete)
			clients.GET("/:id/orders", clientHandler.GetOrders)
		}

		// Módulo de Materiales (Fase 4)
		// ⚠️ low-stock ANTES de /:id para que Gin no lo interprete como un UUID
		materials := protected.Group("/materials")
		{
			materials.GET("", materialHandler.GetAll)
			materials.POST("", materialHandler.Create)
			materials.GET("/alerts/low-stock", materialHandler.GetLowStock)
			materials.GET("/:id", materialHandler.GetByID)
			materials.PUT("/:id", materialHandler.Update)
			materials.DELETE("/:id", materialHandler.Delete)
			materials.POST("/:id/purchases", materialHandler.RegisterPurchase)
			materials.GET("/:id/purchases", materialHandler.GetPurchases)
		}

		// Módulos de Fases 5-8 se agregan aquí progresivamente:
		// orders := protected.Group("/orders")
		// transactions := protected.Group("/transactions")
	}

	return r
}
