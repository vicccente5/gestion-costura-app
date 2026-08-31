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
// @Summary      Crear cliente
// @Description  Crea un nuevo cliente para la costurera autenticada.
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        request body clientRequest true "Datos del cliente"
// @Security     BearerAuth
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /clients [post]
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
// @Summary      Listar clientes
// @Description  Obtiene la lista de clientes paginada y permite búsqueda por nombre.
// @Tags         clients
// @Produce      json
// @Param        page    query     int     false  "Página"
// @Param        limit   query     int     false  "Límite"
// @Param        search  query     string  false  "Búsqueda por nombre"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /clients [get]
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
// @Summary      Obtener cliente por ID
// @Description  Retorna los detalles de un cliente específico.
// @Tags         clients
// @Produce      json
// @Param        id   path      string  true  "ID del cliente (UUID)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /clients/{id} [get]
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
// @Summary      Actualizar cliente
// @Description  Actualiza los datos de un cliente existente.
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        id      path      string         true  "ID del cliente (UUID)"
// @Param        request body      clientRequest  true  "Nuevos datos del cliente"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /clients/{id} [put]
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
// @Summary      Eliminar cliente
// @Description  Elimina un cliente. Retorna error si el cliente tiene encargos activos.
// @Tags         clients
// @Produce      json
// @Param        id   path      string  true  "ID del cliente (UUID)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /clients/{id} [delete]
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
// @Summary      Historial de encargos de cliente
// @Description  Retorna la lista de encargos asociados a este cliente.
// @Tags         clients
// @Produce      json
// @Param        id   path      string  true  "ID del cliente (UUID)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /clients/{id}/orders [get]
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
