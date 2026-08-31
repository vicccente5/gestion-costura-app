// Package middleware — recovery middleware que captura panics y retorna JSON estandarizado.
//
// Por qué reemplazar gin.Recovery():
//   - gin.Recovery() retorna texto plano o HTML, no JSON → inconsistente con nuestra API.
//   - gin.Recovery() puede imprimir el stack trace en la respuesta HTTP → expone internals.
//   - Nuestro recovery loguea el stack con zerolog (solo en logs del servidor)
//     y retorna un JSON genérico 500 al cliente.
//
// IMPORTANTE: El stack trace se loguea solo en el servidor (zerolog), NUNCA en la respuesta HTTP.
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// RecoveryWithLogger captura cualquier panic durante el procesamiento de un request,
// lo registra con zerolog (incluyendo el stack trace para debugging),
// y retorna una respuesta 500 estandarizada al cliente.
func RecoveryWithLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Loguear el panic con stack trace completo en el servidor
				// El stack trace NUNCA llega al cliente — solo al log interno
				log.Error().
					Interface("panic", r).
					Str("stack", string(debug.Stack())).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Msg("panic recuperado — request abortado con 500")

				// Respuesta genérica al cliente: sin detalles internos
				c.AbortWithStatusJSON(http.StatusInternalServerError, utils.Response{
					Success: false,
					Error:   "Error interno del servidor. Por favor intenta nuevamente.",
					Code:    http.StatusInternalServerError,
				})
			}
		}()

		c.Next()
	}
}
