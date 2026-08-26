// Package handler — HTTP handlers del módulo de encargos.
package handler

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/middleware"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// OrderHandler agrupa los handlers del módulo de encargos.
type OrderHandler struct {
	orderSvc service.OrderService
}

// NewOrderHandler crea el handler con inyección del service.
func NewOrderHandler(orderSvc service.OrderService) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
}

// orderMaterialRequest define un material en el body de creación.
type orderMaterialRequest struct {
	MaterialID string  `json:"material_id" validate:"required,uuid"`
	Cantidad   float64 `json:"cantidad"    validate:"required,gt=0"`
}

// orderCreateRequest define los campos para crear un encargo.
type orderCreateRequest struct {
	Descripcion  string                 `json:"descripcion"   validate:"required,min=3,max=500"`
	PrecioVenta  int64                  `json:"precio_venta"  validate:"min=0"`
	Horas        float64                `json:"horas"         validate:"min=0"`
	TarifaHora   int64                  `json:"tarifa_hora"   validate:"min=0"`
	FechaEntrega *string                `json:"fecha_entrega"` // ISO 8601 opcional: "2024-09-30"
	Notas        *string                `json:"notas"         validate:"omitempty,max=1000"`
	ClientID     string                 `json:"client_id"    validate:"required,uuid"`
	Materials    []orderMaterialRequest `json:"materials"    validate:"omitempty,dive"`
}

// orderUpdateRequest permite editar metadata de un encargo pendiente.
type orderUpdateRequest struct {
	Descripcion  string   `json:"descripcion"  validate:"required,min=3,max=500"`
	PrecioVenta  int64    `json:"precio_venta" validate:"min=0"`
	Horas        float64  `json:"horas"        validate:"min=0"`
	TarifaHora   int64    `json:"tarifa_hora"  validate:"min=0"`
	FechaEntrega *string  `json:"fecha_entrega"`
	Notas        *string  `json:"notas"        validate:"omitempty,max=1000"`
}

// statusChangeRequest define el nuevo estado a aplicar.
type statusChangeRequest struct {
	Estado string `json:"estado" validate:"required"`
}

// Create godoc
// POST /api/v1/orders
func (h *OrderHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req orderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos. Verifica descripción, client_id y materiales.")
		return
	}

	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		utils.BadRequest(c, "client_id inválido")
		return
	}

	// Parsear fecha de entrega si se proporcionó
	var fechaEntrega *time.Time
	if req.FechaEntrega != nil && *req.FechaEntrega != "" {
		parsed, err := time.Parse("2006-01-02", *req.FechaEntrega)
		if err != nil {
			utils.BadRequest(c, "Formato de fecha_entrega inválido. Usar: YYYY-MM-DD")
			return
		}
		fechaEntrega = &parsed
	}

	// Convertir materiales del request
	var materials []service.OrderMaterialInput
	for _, m := range req.Materials {
		matID, err := uuid.Parse(m.MaterialID)
		if err != nil {
			utils.BadRequest(c, "material_id inválido: "+m.MaterialID)
			return
		}
		materials = append(materials, service.OrderMaterialInput{
			MaterialID: matID,
			Cantidad:   m.Cantidad,
		})
	}

	order, err := h.orderSvc.Create(c.Request.Context(), userID, service.OrderCreateInput{
		Descripcion:  req.Descripcion,
		PrecioVenta:  req.PrecioVenta,
		Horas:        req.Horas,
		TarifaHora:   req.TarifaHora,
		FechaEntrega: fechaEntrega,
		Notas:        req.Notas,
		ClientID:     clientID,
		Materials:    materials,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderClientNotFound):
			utils.NotFound(c, err.Error())
		case errors.Is(err, service.ErrOrderMaterialNotFound):
			utils.NotFound(c, err.Error())
		case errors.Is(err, service.ErrInsufficientStock):
			utils.Conflict(c, err.Error())
		default:
			utils.InternalError(c, "Error al crear el encargo")
		}
		return
	}

	utils.Created(c, "Encargo creado exitosamente", order)
}

// GetAll godoc
// GET /api/v1/orders?page=1&limit=20&estado=pendiente&search=vestido
func (h *OrderHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := utils.GetPaginationParams(c)
	estado := c.Query("estado")

	orders, total, err := h.orderSvc.GetAll(c.Request.Context(), userID, params, estado)
	if err != nil {
		utils.InternalError(c, "Error al obtener los encargos")
		return
	}

	utils.OK(c, "Encargos obtenidos", utils.NewPaginatedResponse(orders, total, params))
}

// GetByID godoc
// GET /api/v1/orders/:id
func (h *OrderHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	order, err := h.orderSvc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			utils.NotFound(c, "Encargo no encontrado")
			return
		}
		utils.InternalError(c, "Error al obtener el encargo")
		return
	}

	utils.OK(c, "Encargo obtenido", order)
}

// UpdateMetadata godoc
// PUT /api/v1/orders/:id
func (h *OrderHandler) UpdateMetadata(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	var req orderUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos")
		return
	}

	var fechaEntrega *time.Time
	if req.FechaEntrega != nil && *req.FechaEntrega != "" {
		parsed, err := time.Parse("2006-01-02", *req.FechaEntrega)
		if err != nil {
			utils.BadRequest(c, "Formato de fecha_entrega inválido. Usar: YYYY-MM-DD")
			return
		}
		fechaEntrega = &parsed
	}

	order, err := h.orderSvc.UpdateMetadata(c.Request.Context(), id, userID, service.OrderUpdateInput{
		Descripcion:  req.Descripcion,
		PrecioVenta:  req.PrecioVenta,
		Horas:        req.Horas,
		TarifaHora:   req.TarifaHora,
		FechaEntrega: fechaEntrega,
		Notas:        req.Notas,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			utils.NotFound(c, "Encargo no encontrado")
		case errors.Is(err, service.ErrOrderNotEditable):
			utils.Conflict(c, err.Error())
		default:
			utils.InternalError(c, "Error al actualizar el encargo")
		}
		return
	}

	utils.OK(c, "Encargo actualizado exitosamente", order)
}

// ChangeStatus godoc
// PATCH /api/v1/orders/:id/status
func (h *OrderHandler) ChangeStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	var req statusChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}

	// Validar que el estado sea uno de los valores permitidos
	newStatus := domain.OrderStatus(req.Estado)
	validStatuses := map[domain.OrderStatus]bool{
		domain.OrderStatusEnProgreso: true,
		domain.OrderStatusCompletado: true,
		domain.OrderStatusEntregado:  true,
		domain.OrderStatusCancelado:  true,
	}
	if !validStatuses[newStatus] {
		utils.BadRequest(c, "Estado inválido. Valores permitidos: en_progreso, completado, entregado, cancelado")
		return
	}

	order, err := h.orderSvc.ChangeStatus(c.Request.Context(), id, userID, newStatus)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			utils.NotFound(c, "Encargo no encontrado")
		case errors.Is(err, service.ErrOrderInvalidStatusChange),
			errors.Is(err, service.ErrOrderAlreadyDelivered):
			utils.Conflict(c, err.Error())
		case errors.Is(err, service.ErrOrderNoPrice):
			utils.BadRequest(c, err.Error())
		default:
			utils.InternalError(c, "Error al cambiar el estado")
		}
		return
	}

	utils.OK(c, "Estado actualizado exitosamente", order)
}

// Delete godoc
// DELETE /api/v1/orders/:id
func (h *OrderHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	if err := h.orderSvc.Delete(c.Request.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			utils.NotFound(c, "Encargo no encontrado")
		case errors.Is(err, service.ErrOrderNotDeletable):
			utils.Conflict(c, err.Error())
		default:
			utils.InternalError(c, "Error al eliminar el encargo")
		}
		return
	}

	utils.OK(c, "Encargo eliminado exitosamente", nil)
}
