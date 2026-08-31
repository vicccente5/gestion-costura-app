// Package middleware — request logger estructurado con zerolog.
//
// Por qué logging estructurado:
//   - Cada request genera un log en formato JSON (en producción) o legible (en dev).
//   - Los campos son consistentes: método, ruta, status, latencia, user_id.
//   - Facilita búsquedas en herramientas como Grafana Loki, Datadog, etc.
//
// Campos registrados por request:
//   - method:    GET, POST, etc.
//   - path:      /api/v1/clients (sin query params para no loguear datos sensibles)
//   - status:    200, 404, 500...
//   - latency:   duración en milisegundos
//   - user_id:   inyectado si la ruta está autenticada (puede estar vacío en rutas públicas)
//   - client_ip: IP del cliente (útil para detectar abusos)
//
// Datos NUNCA logueados (seguridad):
//   - Passwords
//   - JWT tokens
//   - Query params (podrían contener datos sensibles)
//   - Body de la petición
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestLogger registra cada request HTTP con zerolog.
// Debe registrarse ANTES del AuthMiddleware para capturar el user_id correctamente,
// pero usando c.Next() — zerolog loguea DESPUÉS de que el handler termine para
// tener el status code real.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Procesar la petición primero
		c.Next()

		// Calcular latencia DESPUÉS de que el handler terminó
		latency := time.Since(start)
		status := c.Writer.Status()

		// Extraer user_id si está presente en el contexto (rutas autenticadas)
		userIDStr := ""
		if id := GetUserID(c); id.String() != "00000000-0000-0000-0000-000000000000" {
			userIDStr = id.String()
		}

		// Nivel del log según el status code:
		// 5xx → Error (problema del servidor)
		// 4xx → Warn  (problema del cliente, útil para detectar ataques)
		// resto → Info
		event := log.Info()
		if status >= 500 {
			event = log.Error()
		} else if status >= 400 {
			event = log.Warn()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", c.FullPath()).   // ruta con parámetros ej: /api/v1/clients/:id
			Int("status", status).
			Dur("latency", latency).
			Str("user_id", userIDStr).
			Str("client_ip", c.ClientIP()).
			Msg("request")
	}
}
