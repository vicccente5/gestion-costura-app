// Package service — lógica de negocio de reportes.
//
// Los reportes son consultas de solo lectura sobre datos ya existentes.
// No crean ni modifican ningún dato.
// Para evitar queries N+1, todas las consultas se hacen directamente en SQL
// con agrupaciones en la DB, no en Go.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrderSummary contiene las métricas generales del taller.
type OrderSummary struct {
	TotalEncargos    int64   `json:"total_encargos"`
	PendientesHoy    int64   `json:"pendientes_hoy"`
	EnProgreso       int64   `json:"en_progreso"`
	EntregadosMes    int64   `json:"entregados_mes"`     // encargos entregados en el mes actual
	IngresosMes      int64   `json:"ingresos_mes"`        // ingresos del mes actual (de transactions)
	GastosMes        int64   `json:"gastos_mes"`          // gastos del mes actual
	BalanceMes       int64   `json:"balance_mes"`         // ingresos - gastos del mes actual
}

// TopMaterial representa un material con cuánto se usó.
type TopMaterial struct {
	MaterialID   string  `json:"material_id"`
	Nombre       string  `json:"nombre"`
	Categoria    string  `json:"categoria"`
	Unidad       string  `json:"unidad"`
	TotalUsado   float64 `json:"total_usado"`    // suma de cantidades en order_materials
	VecesUsado   int64   `json:"veces_usado"`    // en cuántos encargos aparece
}

// TopOrder representa un encargo rentable o costoso.
type TopOrder struct {
	OrderID      string  `json:"order_id"`
	Descripcion  string  `json:"descripcion"`
	ClientNombre string  `json:"client_nombre"`
	PrecioVenta  int64   `json:"precio_venta"`
	CostoTotal   int64   `json:"costo_total"`    // costo materiales + mano de obra
	GananciaNeta int64   `json:"ganancia_neta"`
	Estado       string  `json:"estado"`
	FechaCreated string  `json:"fecha_created"`
}

// ReportService define el contrato del servicio de reportes.
type ReportService interface {
	GetSummary(ctx context.Context, userID uuid.UUID) (*OrderSummary, error)
	GetTopMaterials(ctx context.Context, userID uuid.UUID, limit int) ([]TopMaterial, error)
	GetTopOrders(ctx context.Context, userID uuid.UUID, limit int) ([]TopOrder, error)
}

// reportService implementa el servicio de reportes.
type reportService struct {
	db *gorm.DB
}

// NewReportService crea el servicio con acceso directo a la DB.
// Los reportes necesitan consultas complejas que no encajan bien en un repositorio simple.
func NewReportService(db *gorm.DB) ReportService {
	return &reportService{db: db}
}

// GetSummary retorna el resumen general del taller para el dashboard.
func (s *reportService) GetSummary(ctx context.Context, userID uuid.UUID) (*OrderSummary, error) {
	summary := &OrderSummary{}

	// 1. Total de encargos del usuario
	if err := s.db.WithContext(ctx).
		Model(&struct{ ID string }{}).
		Table("orders").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&summary.TotalEncargos).Error; err != nil {
		return nil, err
	}

	// 2. Encargos pendientes
	if err := s.db.WithContext(ctx).
		Table("orders").
		Where("user_id = ? AND estado = ? AND deleted_at IS NULL", userID, "pendiente").
		Count(&summary.PendientesHoy).Error; err != nil {
		return nil, err
	}

	// 3. Encargos en progreso
	if err := s.db.WithContext(ctx).
		Table("orders").
		Where("user_id = ? AND estado = ? AND deleted_at IS NULL", userID, "en_progreso").
		Count(&summary.EnProgreso).Error; err != nil {
		return nil, err
	}

	// 4. Encargos entregados este mes
	monthStart := time.Now().UTC().Truncate(24 * time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1)
	if err := s.db.WithContext(ctx).
		Table("orders").
		Where("user_id = ? AND estado = ? AND deleted_at IS NULL AND updated_at >= ?", userID, "entregado", monthStart).
		Count(&summary.EntregadosMes).Error; err != nil {
		return nil, err
	}

	// 5. Ingresos y gastos del mes actual (desde transactions)
	monthEnd := monthStart.AddDate(0, 1, 0)
	type balanceRow struct {
		Tipo  string `gorm:"column:tipo"`
		Total int64  `gorm:"column:total"`
	}
	var rows []balanceRow
	if err := s.db.WithContext(ctx).
		Table("transactions").
		Select("tipo, SUM(monto) as total").
		Where("user_id = ? AND deleted_at IS NULL AND fecha >= ? AND fecha < ?", userID, monthStart, monthEnd).
		Group("tipo").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row.Tipo == "ingreso" {
			summary.IngresosMes = row.Total
		} else {
			summary.GastosMes = row.Total
		}
	}
	summary.BalanceMes = summary.IngresosMes - summary.GastosMes

	return summary, nil
}

// GetTopMaterials retorna los materiales más utilizados en encargos.
func (s *reportService) GetTopMaterials(ctx context.Context, userID uuid.UUID, limit int) ([]TopMaterial, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var results []TopMaterial
	if err := s.db.WithContext(ctx).
		Table("order_materials om").
		Select(`
			m.id::text as material_id,
			m.nombre,
			m.categoria,
			m.unidad,
			SUM(om.cantidad) as total_usado,
			COUNT(DISTINCT om.order_id) as veces_usado
		`).
		Joins("JOIN materials m ON m.id = om.material_id").
		Joins("JOIN orders o ON o.id = om.order_id").
		Where("o.user_id = ? AND o.deleted_at IS NULL AND m.deleted_at IS NULL", userID).
		Group("m.id, m.nombre, m.categoria, m.unidad").
		Order("total_usado DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// GetTopOrders retorna los encargos con mayor ganancia neta.
// costo_total = SUM(cantidad * costo_unitario_snapshot) + horas * tarifa_hora
// ganancia_neta = precio_venta - costo_total
func (s *reportService) GetTopOrders(ctx context.Context, userID uuid.UUID, limit int) ([]TopOrder, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var results []TopOrder
	if err := s.db.WithContext(ctx).
		Table("orders o").
		Select(`
			o.id::text as order_id,
			o.descripcion,
			c.nombre as client_nombre,
			o.precio_venta,
			(COALESCE(mat_cost.costo_materiales, 0) + (o.horas * o.tarifa_hora)::bigint) as costo_total,
			(o.precio_venta - COALESCE(mat_cost.costo_materiales, 0) - (o.horas * o.tarifa_hora)::bigint) as ganancia_neta,
			o.estado,
			o.created_at::text as fecha_created
		`).
		Joins("JOIN clients c ON c.id = o.client_id").
		Joins(`LEFT JOIN (
			SELECT order_id, SUM(cantidad * costo_unitario_snapshot)::bigint as costo_materiales
			FROM order_materials
			GROUP BY order_id
		) mat_cost ON mat_cost.order_id = o.id`).
		Where("o.user_id = ? AND o.deleted_at IS NULL AND o.precio_venta > 0", userID).
		Order("ganancia_neta DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
