// Package middleware — autenticación JWT para rutas protegidas.
// Extrae y valida el Bearer token del header Authorization.
// Inyecta el user_id en el contexto Gin para que los handlers lo usen.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// ContextKeyUserID es la clave del user_id en el contexto Gin.
// Usar una constante tipada evita colisiones con otras claves del contexto.
const ContextKeyUserID = "user_id"

// AuthMiddleware valida el JWT y rechaza la petición si es inválido.
// Las rutas que usen este middleware requieren un Bearer token válido.
func AuthMiddleware(authSvc service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extraer el token del header "Authorization: Bearer <token>"
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Se requiere autenticación")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.Unauthorized(c, "Formato de token inválido. Usar: Bearer <token>")
			c.Abort()
			return
		}

		tokenStr := parts[1]
		if tokenStr == "" {
			utils.Unauthorized(c, "Token vacío")
			c.Abort()
			return
		}

		// Validar firma, algoritmo y vigencia del JWT
		userID, err := authSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			utils.Unauthorized(c, "Token inválido o expirado")
			c.Abort()
			return
		}

		// Inyectar user_id en el contexto para que los handlers lo usen
		// REGLA DE ORO: todos los queries deben filtrar por este user_id
		c.Set(ContextKeyUserID, userID)
		c.Next()
	}
}

// GetUserID extrae el user_id del contexto Gin de forma segura.
// Retorna uuid.Nil si no está presente (no debería ocurrir en rutas protegidas).
func GetUserID(c *gin.Context) uuid.UUID {
	val, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil
	}
	userID, ok := val.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return userID
}
