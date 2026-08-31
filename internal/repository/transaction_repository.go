// Package repository — interfaz e implementación del repositorio de transacciones.
//
// Decisión de diseño crítica — campos source y tipo:
//   - source="order"  → generadas automáticamente al entregar encargo. NUNCA se crea/edita/elimina manualmente.
//   - source="manual" → ingresadas por la costurera (gastos fijos, otros ingresos).
//   - tipo="ingreso"  → suma al balance.
//   - tipo="gasto"    → resta al balance.
//
// El balance mensual es: SUM(monto WHERE tipo=ingreso) - SUM(monto WHERE tipo=gasto)
// Sin el campo source, un encargo entregado + registro manual duplicaría el ingreso.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// TransactionFilters agrupa los filtros opcionales del listado de transacciones.
type TransactionFilters struct {
	Tipo      string // "ingreso" | "gasto" | "" (todos)
	Source    string // "manual" | "order" | "" (todos)
	Categoria string // categoría exacta | "" (todas)
	Desde     *time.Time
	Hasta     *time.Time
}

// MonthlyBalance contiene el balance calculado de un mes.
type MonthlyBalance struct {
	Mes           string  `json:"mes"`            // formato "YYYY-MM"
	TotalIngresos int64   `json:"total_ingresos"` // suma de todos los ingresos
	TotalGastos   int64   `json:"total_gastos"`   // suma de todos los gastos
	Balance       int64   `json:"balance"`        // ingresos - gastos
	NumTransacciones int  `json:"num_transacciones"`
}

// TransactionRepository define el contrato de acceso a datos de transacciones.
type TransactionRepository interface {
	// Create crea una nueva transacción. Solo debe llamarse con source="manual" desde la API.
	// El servicio de encargos usa directamente la DB en transacción atómica para source="order".
	Create(ctx context.Context, tx *domain.Transaction) error

	// FindByID busca una transacción por ID y user_id.
	FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error)

	// FindAll retorna transacciones paginadas con filtros.
	FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, filters TransactionFilters) ([]domain.Transaction, int64, error)

	// Update edita una transacción. El servicio verifica que sea source="manual" antes de llamar aquí.
	Update(ctx context.Context, tx *domain.Transaction) error

	// Delete hace soft delete. El servicio verifica que sea source="manual".
	Delete(ctx context.Context, id, userID uuid.UUID) error

	// GetMonthlyBalance calcula el balance (ingresos - gastos) de un mes específico.
	// El mes viene en formato "YYYY-MM" (ej: "2024-09").
	GetMonthlyBalance(ctx context.Context, userID uuid.UUID, month string) (*MonthlyBalance, error)

	// GetYearlyEarnings retorna el resumen de ingresos/gastos por mes para un año.
	GetYearlyEarnings(ctx context.Context, userID uuid.UUID, year int) ([]MonthlyBalance, error)
}

// transactionRepository es la implementación GORM.
type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository crea una nueva instancia.
func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *transactionRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	var tx domain.Transaction
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		First(&tx)
	if result.Error != nil {
		return nil, result.Error
	}
	return &tx, nil
}

// FindAll aplica filtros opcionales y pagina resultados.
// Los filtros se acumulan — se aplican todos los que no estén vacíos.
func (r *transactionRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, filters TransactionFilters) ([]domain.Transaction, int64, error) {
	var transactions []domain.Transaction
	var total int64

	query := r.db.WithContext(ctx).
		Model(&domain.Transaction{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	if filters.Tipo != "" {
		query = query.Where("tipo = ?", filters.Tipo)
	}
	if filters.Source != "" {
		query = query.Where("source = ?", filters.Source)
	}
	if filters.Categoria != "" {
		query = query.Where("categoria = ?", filters.Categoria)
	}
	if filters.Desde != nil {
		query = query.Where("fecha >= ?", filters.Desde)
	}
	if filters.Hasta != nil {
		query = query.Where("fecha <= ?", filters.Hasta)
	}
	if params.Search != "" {
		query = query.Where("descripcion ILIKE ?", "%"+params.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("fecha DESC, created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

func (r *transactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	return r.db.WithContext(ctx).Save(tx).Error
}

func (r *transactionRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&domain.Transaction{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetMonthlyBalance calcula el balance de un mes específico (YYYY-MM).
// Usa EXTRACT de PostgreSQL para filtrar por año/mes sin preocuparse por timezone.
// La consulta agrega en una sola pasada en la DB para evitar N+1.
func (r *transactionRepository) GetMonthlyBalance(ctx context.Context, userID uuid.UUID, month string) (*MonthlyBalance, error) {
	// Parsear "YYYY-MM" en un time.Time del primer día del mes
	monthStart, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, err
	}
	monthEnd := monthStart.AddDate(0, 1, 0) // primer día del mes siguiente

	type result struct {
		Tipo          string `gorm:"column:tipo"`
		Total         int64  `gorm:"column:total"`
		NumTrans      int    `gorm:"column:num_trans"`
	}

	var rows []result
	if err := r.db.WithContext(ctx).
		Model(&domain.Transaction{}).
		Select("tipo, SUM(monto) as total, COUNT(*) as num_trans").
		Where("user_id = ? AND deleted_at IS NULL AND fecha >= ? AND fecha < ?", userID, monthStart, monthEnd).
		Group("tipo").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	balance := &MonthlyBalance{Mes: month}
	for _, row := range rows {
		if row.Tipo == string(domain.TransactionTypeIngreso) {
			balance.TotalIngresos = row.Total
			balance.NumTransacciones += row.NumTrans
		} else {
			balance.TotalGastos = row.Total
			balance.NumTransacciones += row.NumTrans
		}
	}
	balance.Balance = balance.TotalIngresos - balance.TotalGastos

	return balance, nil
}

// GetYearlyEarnings retorna el resumen mensual de un año completo.
// Si un mes no tiene transacciones, no aparece en el resultado (el cliente
// debe rellenar los meses vacíos con balance=0).
func (r *transactionRepository) GetYearlyEarnings(ctx context.Context, userID uuid.UUID, year int) ([]MonthlyBalance, error) {
	yearStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := yearStart.AddDate(1, 0, 0)

	type rawRow struct {
		Year  int    `gorm:"column:yr"`
		Month int    `gorm:"column:mo"`
		Tipo  string `gorm:"column:tipo"`
		Total int64  `gorm:"column:total"`
		Num   int    `gorm:"column:num"`
	}

	var rows []rawRow
	if err := r.db.WithContext(ctx).
		Model(&domain.Transaction{}).
		Select("EXTRACT(YEAR FROM fecha)::int as yr, EXTRACT(MONTH FROM fecha)::int as mo, tipo, SUM(monto) as total, COUNT(*) as num").
		Where("user_id = ? AND deleted_at IS NULL AND fecha >= ? AND fecha < ?", userID, yearStart, yearEnd).
		Group("yr, mo, tipo").
		Order("yr, mo").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	// Agrupar por mes
	monthMap := make(map[string]*MonthlyBalance)
	for _, row := range rows {
		key := time.Date(row.Year, time.Month(row.Month), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		if _, ok := monthMap[key]; !ok {
			monthMap[key] = &MonthlyBalance{Mes: key}
		}
		if row.Tipo == string(domain.TransactionTypeIngreso) {
			monthMap[key].TotalIngresos = row.Total
		} else {
			monthMap[key].TotalGastos = row.Total
		}
		monthMap[key].NumTransacciones += row.Num
		monthMap[key].Balance = monthMap[key].TotalIngresos - monthMap[key].TotalGastos
	}

	// Convertir a slice ordenado
	result := make([]MonthlyBalance, 0, len(monthMap))
	for _, v := range monthMap {
		result = append(result, *v)
	}

	return result, nil
}
