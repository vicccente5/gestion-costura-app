// Package middleware — rate limiting para proteger endpoints sensibles.
// Implementa un token bucket por IP usando golang.org/x/time/rate.
// Configurado específicamente para el endpoint de login: máx 5 intentos/minuto.
package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"golang.org/x/time/rate"
)

// ipLimiter almacena el limiter y el último tiempo de acceso para limpieza.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter gestiona los limiters por IP con limpieza automática de entradas antiguas.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit // tokens por segundo
	b        int        // capacidad máxima del bucket
}

// NewRateLimiter crea un nuevo gestor de rate limiting.
//
// Parámetros para login (5 intentos por minuto):
//   - r = 5.0/60.0  → 1 token cada 12 segundos (regeneración continua)
//   - b = 5          → burst máximo de 5 intentos
//
// Con estos valores: si alguien hace 5 intentos seguidos, debe esperar 60 segundos
// antes de poder intentar de nuevo.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		b:        b,
	}
	// Limpiar entradas antiguas cada 5 minutos para evitar fugas de memoria
	go rl.cleanupOldEntries()
	return rl
}

// getLimiter obtiene o crea el limiter para una IP específica.
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &ipLimiter{
			limiter: rate.NewLimiter(rl.r, rl.b),
		}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanupOldEntries elimina IPs que no han hecho peticiones en los últimos 10 minutos.
// Esto evita que la memoria crezca indefinidamente con IPs únicas.
func (rl *RateLimiter) cleanupOldEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware retorna el gin.HandlerFunc que aplica el rate limiting.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			utils.TooManyRequests(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// LoginRateLimiter crea un limiter preconfigurado para el endpoint de login.
// Configuración: máx 5 intentos por minuto por IP.
func LoginRateLimiter() *RateLimiter {
	// rate.Limit(5.0/60.0) = 1 token cada 12 segundos
	// burst = 5 → permite hasta 5 intentos rápidos antes de throttlear
	return NewRateLimiter(rate.Limit(5.0/60.0), 5)
}
