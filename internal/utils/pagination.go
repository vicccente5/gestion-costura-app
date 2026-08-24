// Package utils — paginación reutilizable para todos los endpoints de listado.
// Centralizar la paginación evita duplicar la lógica en cada repository/handler.
// Todos los endpoints de listado (clients, materials, orders, transactions)
// usan este mismo struct para garantizar un comportamiento consistente.
package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams contiene los parámetros de paginación de una petición.
type PaginationParams struct {
	Page    int    // Página actual (1-indexed)
	Limit   int    // Resultados por página
	Search  string // Término de búsqueda opcional (por nombre)
	Offset  int    // Calculado: (page-1) * limit
}

// PaginatedResponse es la estructura de respuesta para listados paginados.
type PaginatedResponse struct {
	Data       any   `json:"data"`
	Total      int64 `json:"total"`       // Total de registros sin paginar
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// GetPaginationParams extrae y valida los parámetros de paginación del query string.
// Valores por defecto: page=1, limit=20.
// Límites: máx. 100 resultados por página para proteger el servidor.
func GetPaginationParams(c *gin.Context) PaginationParams {
	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	search := c.Query("search") // Búsqueda por nombre (opcional)

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Search: search,
		Offset: (page - 1) * limit,
	}
}

// NewPaginatedResponse construye la respuesta paginada con total_pages calculado.
func NewPaginatedResponse(data any, total int64, params PaginationParams) PaginatedResponse {
	totalPages := int(total) / params.Limit
	if int(total)%params.Limit != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	return PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}
}
