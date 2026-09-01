package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders añade encabezados HTTP de seguridad recomendados
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// No permitir que el navegador intente adivinar el tipo MIME
		c.Header("X-Content-Type-Options", "nosniff")
		// Proteger contra Clickjacking
		c.Header("X-Frame-Options", "DENY")
		// Forzar HTTPS en conexiones futuras (1 año)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Evitar ataques XSS filtrando cargas útiles reflectivas básicas
		c.Header("X-XSS-Protection", "1; mode=block")

		c.Next()
	}
}

// MaxBodySize limita el tamaño del cuerpo de la petición para prevenir DoS
func MaxBodySize(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
