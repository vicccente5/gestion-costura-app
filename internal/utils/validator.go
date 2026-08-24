// Package utils — instancia global del validador go-playground/validator.
// Centralizar el validador evita crear una instancia nueva en cada request
// (costoso en rendimiento). Con una instancia global también se pueden
// registrar validadores y traductores personalizados en un solo lugar.
package utils

import "github.com/go-playground/validator/v10"

// Validate es la instancia global del validador.
// Se inicializa una sola vez en el arranque de la aplicación.
var Validate = validator.New()
