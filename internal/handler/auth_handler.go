// Package handler — HTTP handler de autenticación.
// Recibe peticiones HTTP, valida los inputs con go-playground/validator,
// llama al service y serializa la respuesta JSON.
// Los handlers son delgados: no contienen lógica de negocio, solo traducen
// HTTP ↔ dominio.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// AuthHandler agrupa los handlers de autenticación.
type AuthHandler struct {
	authSvc service.AuthService
}

// NewAuthHandler crea el handler con inyección del service.
func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// registerRequest define los campos esperados en el body del registro.
// Los tags de `validate` son procesados por go-playground/validator.
type registerRequest struct {
	Nombre   string `json:"nombre"    validate:"required,min=2,max=100"`
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8"`
}

// Register godoc
// @Summary      Registrar usuaria
// @Description  Crea una nueva cuenta de costurera.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Datos de registro"
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}

	// Validar campos con go-playground/validator
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, formatValidationError(err))
		return
	}

	user, err := h.authSvc.Register(c.Request.Context(), req.Nombre, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al crear la cuenta")
		return
	}

	// Respuesta: solo los datos públicos del usuario, nunca el password_hash
	utils.Created(c, "Cuenta creada exitosamente", gin.H{
		"id":         user.ID,
		"nombre":     user.Nombre,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

// loginRequest define los campos del login.
type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

// Login godoc
// @Summary      Iniciar sesión
// @Description  Autentica a una costurera y retorna access + refresh tokens.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Credenciales"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		// ⚠️ REGLA DE SEGURIDAD: en login NO detallar qué campo falló
		// para no dar pistas sobre si el email existe.
		// Sin embargo, la validación de formato antes de llegar al service es correcta.
		utils.BadRequest(c, "Datos de acceso inválidos")
		return
	}

	tokens, err := h.authSvc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			// Siempre 401 con el mismo mensaje, sin revelar cuál campo falló
			utils.Unauthorized(c, "Credenciales inválidas")
			return
		}
		utils.InternalError(c, "Error al iniciar sesión")
		return
	}

	utils.OK(c, "Sesión iniciada", tokens)
}

// refreshRequest contiene el refresh token para renovar el access token.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Refresh godoc
// @Summary      Renovar access token
// @Description  Genera un nuevo access token usando el refresh token (Token Rotation).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body refreshRequest true "Refresh Token"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}

	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "refresh_token es requerido")
		return
	}

	tokens, err := h.authSvc.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefreshToken) || errors.Is(err, service.ErrTokenReuseDetected) {
			utils.Unauthorized(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al renovar la sesión")
		return
	}

	utils.OK(c, "Tokens renovados", tokens)
}

// logoutRequest contiene el refresh token a invalidar.
type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Logout godoc
// @Summary      Cerrar sesión
// @Description  Invalida el refresh token en la DB.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body logoutRequest true "Refresh Token"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}

	// Logout es idempotente: si el token ya no existe, igual retornamos éxito
	_ = h.authSvc.Logout(c.Request.Context(), req.RefreshToken)

	utils.OK(c, "Sesión cerrada exitosamente", nil)
}

// formatValidationError convierte los errores de go-playground/validator
// en un mensaje legible para el usuario (en español).
func formatValidationError(err error) string {
	// En una implementación completa se iteroría sobre err.(validator.ValidationErrors)
	// para generar mensajes por campo. Por ahora un mensaje genérico es suficiente.
	return "Datos de entrada inválidos. Verifica los campos requeridos."
}
