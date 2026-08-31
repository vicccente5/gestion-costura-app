// Package handler — HTTP handlers del módulo de transacciones y reportes.
package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/middleware"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// TransactionHandler agrupa los handlers del módulo de flujo de caja.
type TransactionHandler struct {
	txSvc service.TransactionService
}

// NewTransactionHandler crea el handler con inyección del service.
func NewTransactionHandler(txSvc service.TransactionService) *TransactionHandler {
	return &TransactionHandler{txSvc: txSvc}
}

// transactionCreateRequest define los campos para crear una transacción manual.
type transactionCreateRequest struct {
	Tipo        string  `json:"tipo"        validate:"required,oneof=ingreso gasto"`
	Monto       int64   `json:"monto"       validate:"required,min=1"`
	Descripcion string  `json:"descripcion" validate:"required,min=2,max=500"`
	Categoria   *string `json:"categoria"   validate:"omitempty,max=100"`
	Fecha       string  `json:"fecha"       validate:"required"` // ISO 8601: "2024-09-15"
}

// transactionUpdateRequest define los campos editables de una transacción manual.
type transactionUpdateRequest struct {
	Tipo        string  `json:"tipo"        validate:"required,oneof=ingreso gasto"`
	Monto       int64   `json:"monto"       validate:"required,min=1"`
	Descripcion string  `json:"descripcion" validate:"required,min=2,max=500"`
	Categoria   *string `json:"categoria"   validate:"omitempty,max=100"`
	Fecha       string  `json:"fecha"       validate:"required"`
}

// Create godoc
// @Summary      Crear transacción manual
// @Description  Registra un ingreso o gasto manual. No vinculada a encargos o compras de materiales.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        request body transactionCreateRequest true "Datos de la transacción"
// @Security     BearerAuth
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /transactions [post]
func (h *TransactionHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req transactionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos: tipo (ingreso/gasto), monto (>0), descripción y fecha son requeridos")
		return
	}

	fecha, err := time.Parse("2006-01-02", req.Fecha)
	if err != nil {
		utils.BadRequest(c, "Formato de fecha inválido. Usar: YYYY-MM-DD")
		return
	}

	tx, err := h.txSvc.Create(c.Request.Context(), userID, service.TransactionCreateInput{
		Tipo:        domain.TransactionType(req.Tipo),
		Monto:       req.Monto,
		Descripcion: req.Descripcion,
		Categoria:   req.Categoria,
		Fecha:       fecha,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTransactionInvalidTipo):
			utils.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrTransactionMontoZero):
			utils.BadRequest(c, err.Error())
		default:
			utils.InternalError(c, "Error al crear la transacción")
		}
		return
	}

	utils.Created(c, "Transacción registrada exitosamente", tx)
}

// GetAll godoc
// @Summary      Listar transacciones
// @Description  Obtiene la lista de transacciones con soporte para filtrado y paginación.
// @Tags         transactions
// @Produce      json
// @Param        page      query     int     false  "Página"
// @Param        limit     query     int     false  "Límite"
// @Param        tipo      query     string  false  "Filtro por tipo (ingreso, gasto)"
// @Param        source    query     string  false  "Filtro por fuente (manual, order, material)"
// @Param        categoria query     string  false  "Filtro por categoría"
// @Param        desde     query     string  false  "Fecha inicio (YYYY-MM-DD)"
// @Param        hasta     query     string  false  "Fecha fin (YYYY-MM-DD)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /transactions [get]
func (h *TransactionHandler) GetAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := utils.GetPaginationParams(c)

	// Filtros opcionales
	filters := repository.TransactionFilters{
		Tipo:      c.Query("tipo"),
		Source:    c.Query("source"),
		Categoria: c.Query("categoria"),
	}

	if desde := c.Query("desde"); desde != "" {
		if t, err := time.Parse("2006-01-02", desde); err == nil {
			filters.Desde = &t
		}
	}
	if hasta := c.Query("hasta"); hasta != "" {
		if t, err := time.Parse("2006-01-02", hasta); err == nil {
			// El filtro "hasta" debe incluir todo el día
			end := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filters.Hasta = &end
		}
	}

	transactions, total, err := h.txSvc.GetAll(c.Request.Context(), userID, params, filters)
	if err != nil {
		utils.InternalError(c, "Error al obtener las transacciones")
		return
	}

	utils.OK(c, "Transacciones obtenidas", utils.NewPaginatedResponse(transactions, total, params))
}

// GetByID godoc
// @Summary      Obtener transacción por ID
// @Description  Retorna los detalles de una transacción específica.
// @Tags         transactions
// @Produce      json
// @Param        id   path      string  true  "ID de la transacción (UUID)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /transactions/{id} [get]
func (h *TransactionHandler) GetByID(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	tx, err := h.txSvc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrTransactionNotFound) {
			utils.NotFound(c, "Transacción no encontrada")
			return
		}
		utils.InternalError(c, "Error al obtener la transacción")
		return
	}

	utils.OK(c, "Transacción obtenida", tx)
}

// Update godoc
// @Summary      Actualizar transacción manual
// @Description  Actualiza datos de una transacción manual.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        id      path      string                   true  "ID de la transacción (UUID)"
// @Param        request body      transactionUpdateRequest true  "Nuevos datos"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /transactions/{id} [put]
func (h *TransactionHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	var req transactionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "El cuerpo de la petición tiene formato inválido")
		return
	}
	if err := utils.Validate.Struct(req); err != nil {
		utils.BadRequest(c, "Datos de entrada inválidos")
		return
	}

	fecha, err := time.Parse("2006-01-02", req.Fecha)
	if err != nil {
		utils.BadRequest(c, "Formato de fecha inválido. Usar: YYYY-MM-DD")
		return
	}

	tx, err := h.txSvc.Update(c.Request.Context(), id, userID, service.TransactionUpdateInput{
		Tipo:        domain.TransactionType(req.Tipo),
		Monto:       req.Monto,
		Descripcion: req.Descripcion,
		Categoria:   req.Categoria,
		Fecha:       fecha,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTransactionNotFound):
			utils.NotFound(c, "Transacción no encontrada")
		case errors.Is(err, service.ErrTransactionNotEditable):
			utils.Conflict(c, err.Error())
		case errors.Is(err, service.ErrTransactionInvalidTipo),
			errors.Is(err, service.ErrTransactionMontoZero):
			utils.BadRequest(c, err.Error())
		default:
			utils.InternalError(c, "Error al actualizar la transacción")
		}
		return
	}

	utils.OK(c, "Transacción actualizada exitosamente", tx)
}

// Delete godoc
// @Summary      Eliminar transacción manual
// @Description  Elimina una transacción manual (las automáticas no se pueden eliminar por esta vía).
// @Tags         transactions
// @Produce      json
// @Param        id   path      string  true  "ID de la transacción (UUID)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      409  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /transactions/{id} [delete]
func (h *TransactionHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	id, err := parseUUID(c, "id")
	if err != nil {
		return
	}

	if err := h.txSvc.Delete(c.Request.Context(), id, userID); err != nil {
		switch {
		case errors.Is(err, service.ErrTransactionNotFound):
			utils.NotFound(c, "Transacción no encontrada")
		case errors.Is(err, service.ErrTransactionNotEditable):
			utils.Conflict(c, err.Error())
		default:
			utils.InternalError(c, "Error al eliminar la transacción")
		}
		return
	}

	utils.OK(c, "Transacción eliminada exitosamente", nil)
}

// GetBalance godoc
// @Summary      Obtener balance mensual
// @Description  Calcula los ingresos, gastos y ganancia neta para un mes específico.
// @Tags         transactions
// @Produce      json
// @Param        month  query     string  false  "Mes (YYYY-MM). Por defecto, mes actual."
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /transactions/balance [get]
func (h *TransactionHandler) GetBalance(c *gin.Context) {
	userID := middleware.GetUserID(c)

	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01") // mes actual por defecto
	}

	balance, err := h.txSvc.GetMonthlyBalance(c.Request.Context(), userID, month)
	if err != nil {
		if errors.Is(err, service.ErrTransactionInvalidMonth) {
			utils.BadRequest(c, err.Error())
			return
		}
		utils.InternalError(c, "Error al calcular el balance")
		return
	}

	utils.OK(c, "Balance mensual obtenido", balance)
}

// ──────────────────────────────────────────────
// ReportHandler
// ──────────────────────────────────────────────

// ReportHandler agrupa los handlers del módulo de reportes.
type ReportHandler struct {
	reportSvc service.ReportService
	txSvc     service.TransactionService
}

// NewReportHandler crea el handler con inyección de servicios.
func NewReportHandler(reportSvc service.ReportService, txSvc service.TransactionService) *ReportHandler {
	return &ReportHandler{
		reportSvc: reportSvc,
		txSvc:     txSvc,
	}
}

// GetSummary godoc
// @Summary      Resumen general
// @Description  Retorna KPIs principales: encargos activos, ingresos del mes, materiales bajos, etc.
// @Tags         reports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /reports/summary [get]
func (h *ReportHandler) GetSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	summary, err := h.reportSvc.GetSummary(c.Request.Context(), userID)
	if err != nil {
		utils.InternalError(c, "Error al generar el resumen")
		return
	}

	utils.OK(c, "Resumen generado", summary)
}

// GetEarnings godoc
// @Summary      Ingresos anuales
// @Description  Retorna los ingresos desglosados por mes para un año determinado.
// @Tags         reports
// @Produce      json
// @Param        year  query     int  false  "Año (ej: 2024). Por defecto, año actual."
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /reports/earnings [get]
func (h *ReportHandler) GetEarnings(c *gin.Context) {
	userID := middleware.GetUserID(c)

	year := time.Now().Year()
	if y := c.Query("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}

	earnings, err := h.txSvc.GetYearlyEarnings(c.Request.Context(), userID, year)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	utils.OK(c, "Ingresos anuales obtenidos", gin.H{
		"year":   year,
		"months": earnings,
	})
}

// GetTopMaterials godoc
// @Summary      Materiales más usados
// @Description  Retorna los materiales más utilizados en encargos.
// @Tags         reports
// @Produce      json
// @Param        limit  query     int  false  "Límite de resultados (default 10)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /reports/top-materials [get]
func (h *ReportHandler) GetTopMaterials(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	materials, err := h.reportSvc.GetTopMaterials(c.Request.Context(), userID, limit)
	if err != nil {
		utils.InternalError(c, "Error al obtener los materiales más usados")
		return
	}

	utils.OK(c, "Materiales más usados obtenidos", materials)
}

// GetTopOrders godoc
// @Summary      Encargos más rentables
// @Description  Retorna los encargos completados/entregados con mayor rentabilidad.
// @Tags         reports
// @Produce      json
// @Param        limit  query     int  false  "Límite de resultados (default 10)"
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /reports/top-orders [get]
func (h *ReportHandler) GetTopOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	orders, err := h.reportSvc.GetTopOrders(c.Request.Context(), userID, limit)
	if err != nil {
		utils.InternalError(c, "Error al obtener los encargos más rentables")
		return
	}

	utils.OK(c, "Encargos más rentables obtenidos", orders)
}
