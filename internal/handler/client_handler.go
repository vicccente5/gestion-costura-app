// Package handler — HTTP handlers del módulo de clientes.
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/middleware"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// ClientHandler agrupa los handlers del módulo de clientes.
type ClientHandler struct {
	clientSvc service.ClientService
}

// NewClientHandler crea el handler con inyección del service.
func NewClientHandler(clientSvc service.ClientService) *ClientHandler {
	return &ClientHandler{clientSvc: clientSvc}
}

// clientRequest define los campos de entrada para crear/editar un cliente.
type clientRequest struct {
	Nombre   string  `json:"nombre"    validate:"required,min=2,max=150"`
	Telefono *string `json:"telefono"  validate:"omitempty,min=7,max=20"`
	Email    *string `json:"email"     validate:"omitempty,email"`
}

// Create godoc
// POST /api/v1/clients
func (h *ClientHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req clientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos. Verifica nombre (obligatorio), email y teléfono.")
		return
	}

	input := service.ClientInput{
		Nombre:   req.Nombre,
		Telefono: req.Telefono,
		Email:    req.Email,
	}

	client, err := h.clientSvc.Create(c.Request.Context(), userID, input)
	if err != nil {
		if errors.Is(err, service.ErrClientEmailDuplicate) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al crear el cliente")
		return
	}

	utils.Created(c, "Cliente creado exitosamente", client)
}

// GetAll godoc
// GET /api/v1/clients?page=1&limit=20&search=ana
func (h *ClientHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := utils.GetPaginationParams(c)

	clients, total, err := h.clientSvc.GetAll(c.Request.Context(), userID, params)
	if err != nil {
		utils.InternalError(c, "Error al obtener los clientes")
		return
	}

	utils.OK(c, "Clientes obtenidos", utils.NewPaginatedResponse(clients, total, params))
}

// GetByID godoc
// GET /api/v1/clients/:id
func (h *ClientHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	client, err := h.clientSvc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrClientNotFound) {
			utils.NotFound(c, "Cliente no encontrado")
			return
		}
		utils.InternalError(c, "Error al obtener el cliente")
		return
	}

	utils.OK(c, "Cliente obtenido", client)
}

// Update godoc
// PUT /api/v1/clients/:id
func (h *ClientHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	var req clientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos")
		return
	}

	input := service.ClientInput{
		Nombre:   req.Nombre,
		Telefono: req.Telefono,
		Email:    req.Email,
	}

	client, err := h.clientSvc.Update(c.Request.Context(), id, userID, input)
	if err != nil {
		if errors.Is(err, service.ErrClientNotFound) {
			utils.NotFound(c, "Cliente no encontrado")
			return
		}
		if errors.Is(err, service.ErrClientEmailDuplicate) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al actualizar el cliente")
		return
	}

	utils.OK(c, "Cliente actualizado exitosamente", client)
}

// Delete godoc
// DELETE /api/v1/clients/:id
func (h *ClientHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	if err := h.clientSvc.Delete(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, service.ErrClientNotFound) {
			utils.NotFound(c, "Cliente no encontrado")
			return
		}
		if errors.Is(err, service.ErrClientHasActiveOrders) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al eliminar el cliente")
		return
	}

	utils.OK(c, "Cliente eliminado exitosamente", nil)
}

// GetOrders godoc
// GET /api/v1/clients/:id/orders
func (h *ClientHandler) GetOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	orders, err := h.clientSvc.GetOrders(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrClientNotFound) {
			utils.NotFound(c, "Cliente no encontrado")
			return
		}
		utils.InternalError(c, "Error al obtener los encargos")
		return
	}

	utils.OK(c, "Encargos del cliente", orders)
}

// parseUUID extrae y valida un parámetro UUID del path.
// Si falla, envía la respuesta de error y retorna un error para que el handler retorne.
func parseUUID(c *gin.Context, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		utils.BadRequest(c, "ID inválido — debe ser un UUID")
		return uuid.Nil, err
	}
	return id, nil
}
