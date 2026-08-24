// Package utils — respuestas JSON estandarizadas para toda la API.
// Centralizar el formato de respuesta garantiza consistencia en todos los endpoints.
// Formato de éxito:  { "success": true,  "data": {...},        "message": "..." }
// Formato de error:  { "success": false, "error": "descrip.", "code": 400 }
package utils

import "github.com/gin-gonic/gin"

// Response es la estructura base de todas las respuestas de la API.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    int    `json:"code,omitempty"`
}

// OK envía una respuesta exitosa 200 con datos opcionales.
func OK(c *gin.Context, message string, data any) {
	c.JSON(200, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created envía una respuesta 201 para recursos recién creados.
func Created(c *gin.Context, message string, data any) {
	c.JSON(201, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// BadRequest envía un error 400 de validación o petición inválida.
func BadRequest(c *gin.Context, errMsg string) {
	c.JSON(400, Response{
		Success: false,
		Error:   errMsg,
		Code:    400,
	})
}

// Unauthorized envía un error 401 de autenticación fallida.
// IMPORTANTE: El mensaje debe ser siempre genérico en el login
// para no revelar si el email existe o no.
func Unauthorized(c *gin.Context, errMsg string) {
	c.JSON(401, Response{
		Success: false,
		Error:   errMsg,
		Code:    401,
	})
}

// Forbidden envía un error 403 cuando el usuario no tiene permiso.
func Forbidden(c *gin.Context, errMsg string) {
	c.JSON(403, Response{
		Success: false,
		Error:   errMsg,
		Code:    403,
	})
}

// NotFound envía un error 404.
func NotFound(c *gin.Context, errMsg string) {
	c.JSON(404, Response{
		Success: false,
		Error:   errMsg,
		Code:    404,
	})
}

// Conflict envía un error 409 para conflictos de datos (ej: email duplicado).
func Conflict(c *gin.Context, errMsg string) {
	c.JSON(409, Response{
		Success: false,
		Error:   errMsg,
		Code:    409,
	})
}

// TooManyRequests envía un error 429 para rate limiting.
func TooManyRequests(c *gin.Context) {
	c.JSON(429, Response{
		Success: false,
		Error:   "Demasiados intentos. Espera un momento e inténtalo de nuevo.",
		Code:    429,
	})
}

// InternalError envía un error 500. El mensaje interno NO se expone al cliente.
func InternalError(c *gin.Context, errMsg string) {
	c.JSON(500, Response{
		Success: false,
		Error:   errMsg,
		Code:    500,
	})
}
