// Package handler — HTTP handlers del módulo de inventario de materiales.
package handler

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicccente5/gestion-costura-app/internal/middleware"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// MaterialHandler agrupa los handlers del módulo de inventario.
type MaterialHandler struct {
	materialSvc service.MaterialService
}

// NewMaterialHandler crea el handler con inyección del service.
func NewMaterialHandler(materialSvc service.MaterialService) *MaterialHandler {
	return &MaterialHandler{materialSvc: materialSvc}
}

// materialCreateRequest define los campos para crear un material.
type materialCreateRequest struct {
	Nombre        string  `json:"nombre"         validate:"required,min=2,max=150"`
	Categoria     string  `json:"categoria"      validate:"required,min=2,max=100"`
	Unidad        string  `json:"unidad"         validate:"required,min=1,max=50"` // ej: metros, gramos, unidades
	StockMinimo   float64 `json:"stock_minimo"   validate:"min=0"`
	CostoUnitario int64   `json:"costo_unitario" validate:"min=0"` // estimación inicial en CLP
}

// materialUpdateRequest permite modificar metadatos (NO stock ni costo directo).
type materialUpdateRequest struct {
	Nombre      string  `json:"nombre"       validate:"required,min=2,max=150"`
	Categoria   string  `json:"categoria"    validate:"required,min=2,max=100"`
	Unidad      string  `json:"unidad"       validate:"required,min=1,max=50"`
	StockMinimo float64 `json:"stock_minimo" validate:"min=0"`
}

// purchaseRequest define los campos para registrar una compra.
type purchaseRequest struct {
	Cantidad       float64  `json:"cantidad"        validate:"required,gt=0"`
	PrecioUnitario int64    `json:"precio_unitario" validate:"required,gt=0"` // CLP
	Fecha          string   `json:"fecha"           validate:"required"`        // ISO 8601: "2024-08-24"
	Notas          *string  `json:"notas"           validate:"omitempty,max=500"`
}

// Create godoc
// POST /api/v1/materials
func (h *MaterialHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req materialCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos. Verifica nombre, categoría, unidad y stock mínimo.")
		return
	}

	material, err := h.materialSvc.Create(c.Request.Context(), userID, service.MaterialInput{
		Nombre:        req.Nombre,
		Categoria:     req.Categoria,
		Unidad:        req.Unidad,
		StockMinimo:   req.StockMinimo,
		CostoUnitario: req.CostoUnitario,
	})
	if err != nil {
		if errors.Is(err, service.ErrMaterialNameDuplicate) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al crear el material")
		return
	}

	utils.Created(c, "Material creado exitosamente", material)
}

// GetAll godoc
// GET /api/v1/materials?page=1&limit=20&search=tela&categoria=tela
func (h *MaterialHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := utils.GetPaginationParams(c)
	categoria := c.Query("categoria")

	materials, total, err := h.materialSvc.GetAll(c.Request.Context(), userID, params, categoria)
	if err != nil {
		utils.InternalError(c, "Error al obtener los materiales")
		return
	}

	utils.OK(c, "Materiales obtenidos", utils.NewPaginatedResponse(materials, total, params))
}

// GetLowStock godoc
// GET /api/v1/materials/alerts/low-stock
// ⚠️ IMPORTANTE: esta ruta debe registrarse ANTES de /:id en Gin
// para evitar que Gin la interprete como un ID con valor "alerts".
func (h *MaterialHandler) GetLowStock(c *gin.Context) {
	userID := middleware.GetUserID(c)

	materials, err := h.materialSvc.GetLowStock(c.Request.Context(), userID)
	if err != nil {
		utils.InternalError(c, "Error al obtener alertas de stock")
		return
	}

	utils.OK(c, "Materiales con stock bajo", gin.H{
		"count":     len(materials),
		"materials": materials,
	})
}

// GetByID godoc
// GET /api/v1/materials/:id
func (h *MaterialHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	material, err := h.materialSvc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrMaterialNotFound) {
			utils.NotFound(c, "Material no encontrado")
			return
		}
		utils.InternalError(c, "Error al obtener el material")
		return
	}

	utils.OK(c, "Material obtenido", material)
}

// Update godoc
// PUT /api/v1/materials/:id
func (h *MaterialHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	var req materialUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos")
		return
	}

	material, err := h.materialSvc.Update(c.Request.Context(), id, userID, service.MaterialInput{
		Nombre:      req.Nombre,
		Categoria:   req.Categoria,
		Unidad:      req.Unidad,
		StockMinimo: req.StockMinimo,
	})
	if err != nil {
		if errors.Is(err, service.ErrMaterialNotFound) {
			utils.NotFound(c, "Material no encontrado")
			return
		}
		if errors.Is(err, service.ErrMaterialNameDuplicate) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al actualizar el material")
		return
	}

	utils.OK(c, "Material actualizado exitosamente", material)
}

// Delete godoc
// DELETE /api/v1/materials/:id
func (h *MaterialHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	if err := h.materialSvc.Delete(c.Request.Context(), id, userID); err != nil {
		if errors.Is(err, service.ErrMaterialNotFound) {
			utils.NotFound(c, "Material no encontrado")
			return
		}
		if errors.Is(err, service.ErrMaterialUsedInOrders) {
			utils.Conflict(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al eliminar el material")
		return
	}

	utils.OK(c, "Material eliminado exitosamente", nil)
}

// RegisterPurchase godoc
// POST /api/v1/materials/:id/purchases
func (h *MaterialHandler) RegisterPurchase(c *gin.Context) {
	userID := middleware.GetUserID(c)

	materialID, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	var req purchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos inválidos. Cantidad y precio deben ser mayores a 0. Fecha en formato YYYY-MM-DD.")
		return
	}

	// Parsear la fecha del string ISO 8601
	fecha, err := time.Parse("2006-01-02", req.Fecha)
	if err != nil {
		utils.BadRequest(c, "Formato de fecha inválido. Usar: YYYY-MM-DD")
		return
	}

	purchase, updatedMaterial, err := h.materialSvc.RegisterPurchase(
		c.Request.Context(), materialID, userID,
		service.PurchaseInput{
			Cantidad:       req.Cantidad,
			PrecioUnitario: req.PrecioUnitario,
			Fecha:          fecha,
			Notas:          req.Notas,
		},
	)
	if err != nil {
		if errors.Is(err, service.ErrMaterialNotFound) {
			utils.NotFound(c, "Material no encontrado")
			return
		}
		if errors.Is(err, service.ErrPurchaseQuantityInvalid) || errors.Is(err, service.ErrPurchasePriceInvalid) {
			utils.BadRequest(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al registrar la compra")
		return
	}

	utils.Created(c, "Compra registrada exitosamente", gin.H{
		"compra":   purchase,
		"material": updatedMaterial, // incluir material actualizado para que el cliente refresque el stock
	})
}

// GetPurchases godoc
// GET /api/v1/materials/:id/purchases
func (h *MaterialHandler) GetPurchases(c *gin.Context) {
	userID := middleware.GetUserID(c)

	materialID, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	purchases, err := h.materialSvc.GetPurchases(c.Request.Context(), materialID, userID)
	if err != nil {
		if errors.Is(err, service.ErrMaterialNotFound) {
			utils.NotFound(c, "Material no encontrado")
			return
		}
		utils.InternalError(c, "Error al obtener las compras")
		return
	}

	utils.OK(c, "Historial de compras", purchases)
}
