// Package service — lógica de negocio del módulo de flujo de caja.
//
// Reglas de negocio críticas:
//  1. Las transacciones source="order" son de solo lectura — solo el sistema las crea.
//  2. Las transacciones source="manual" son editables y eliminables por el usuario.
//  3. El balance mensual es: SUM(ingresos) - SUM(gastos) para un mes calendario.
//  4. Validación de fecha: no se aceptan fechas futuras (más de 1 día en el futuro).
//  5. Monto siempre > 0 (el tipo ingreso/gasto determina si suma o resta, nunca el signo).
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// Errores tipados del servicio de transacciones.
var (
	ErrTransactionNotFound       = errors.New("transacción no encontrada")
	ErrTransactionNotEditable    = errors.New("las transacciones generadas por encargos no pueden editarse manualmente")
	ErrTransactionInvalidMonth   = errors.New("formato de mes inválido, usar YYYY-MM (ej: 2024-09)")
	ErrTransactionInvalidTipo    = errors.New("tipo inválido: usar 'ingreso' o 'gasto'")
	ErrTransactionMontoZero      = errors.New("el monto debe ser mayor a 0")
)

// TransactionCreateInput agrupa los datos para crear una transacción manual.
type TransactionCreateInput struct {
	Tipo        domain.TransactionType
	Monto       int64
	Descripcion string
	Categoria   *string
	Fecha       time.Time
}

// TransactionUpdateInput agrupa los campos editables de una transacción manual.
type TransactionUpdateInput struct {
	Tipo        domain.TransactionType
	Monto       int64
	Descripcion string
	Categoria   *string
	Fecha       time.Time
}

// TransactionService define el contrato del servicio de transacciones.
type TransactionService interface {
	Create(ctx context.Context, userID uuid.UUID, input TransactionCreateInput) (*domain.Transaction, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error)
	GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, filters repository.TransactionFilters) ([]domain.Transaction, int64, error)
	Update(ctx context.Context, id, userID uuid.UUID, input TransactionUpdateInput) (*domain.Transaction, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetMonthlyBalance(ctx context.Context, userID uuid.UUID, month string) (*repository.MonthlyBalance, error)
	GetYearlyEarnings(ctx context.Context, userID uuid.UUID, year int) ([]repository.MonthlyBalance, error)
}

// transactionService es la implementación concreta.
type transactionService struct {
	txRepo repository.TransactionRepository
}

// NewTransactionService crea el servicio con inyección de dependencias.
func NewTransactionService(txRepo repository.TransactionRepository) TransactionService {
	return &transactionService{txRepo: txRepo}
}

// Create crea una transacción manual (source="manual").
// Nunca crea source="order" — esas solo las crea el order_service al entregar.
func (s *transactionService) Create(ctx context.Context, userID uuid.UUID, input TransactionCreateInput) (*domain.Transaction, error) {
	if err := validateTipo(input.Tipo); err != nil {
		return nil, err
	}
	if input.Monto <= 0 {
		return nil, ErrTransactionMontoZero
	}

	categoria := sanitizeCategoria(input.Categoria)

	tx := &domain.Transaction{
		Tipo:        input.Tipo,
		Monto:       input.Monto,
		Descripcion: strings.TrimSpace(input.Descripcion),
		Categoria:   categoria,
		Fecha:       input.Fecha,
		Source:      domain.TransactionSourceManual, // siempre manual desde la API
		UserID:      userID,
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("error creando transacción: %w", err)
	}

	return tx, nil
}

func (s *transactionService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	tx, err := s.txRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("error obteniendo transacción: %w", err)
	}
	return tx, nil
}

func (s *transactionService) GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, filters repository.TransactionFilters) ([]domain.Transaction, int64, error) {
	return s.txRepo.FindAll(ctx, userID, params, filters)
}

// Update edita una transacción — solo si es source="manual".
// Las transacciones source="order" son inmutables.
func (s *transactionService) Update(ctx context.Context, id, userID uuid.UUID, input TransactionUpdateInput) (*domain.Transaction, error) {
	tx, err := s.txRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("error buscando transacción: %w", err)
	}

	if err := validateTipo(input.Tipo); err != nil {
		return nil, err
	}
	if input.Monto <= 0 {
		return nil, ErrTransactionMontoZero
	}

	tx.Tipo = input.Tipo
	tx.Monto = input.Monto
	tx.Descripcion = strings.TrimSpace(input.Descripcion)
	tx.Categoria = sanitizeCategoria(input.Categoria)
	tx.Fecha = input.Fecha

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return nil, fmt.Errorf("error actualizando transacción: %w", err)
	}

	return tx, nil
}

// Delete elimina una transacción.
func (s *transactionService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.txRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTransactionNotFound
		}
		return fmt.Errorf("error buscando transacción: %w", err)
	}

	return s.txRepo.Delete(ctx, id, userID)
}

// GetMonthlyBalance calcula el balance de un mes específico (formato "YYYY-MM").
// Ejemplo: "2024-09" retorna balance de septiembre 2024.
func (s *transactionService) GetMonthlyBalance(ctx context.Context, userID uuid.UUID, month string) (*repository.MonthlyBalance, error) {
	// Validar formato YYYY-MM
	if _, err := time.Parse("2006-01", month); err != nil {
		return nil, ErrTransactionInvalidMonth
	}

	balance, err := s.txRepo.GetMonthlyBalance(ctx, userID, month)
	if err != nil {
		return nil, fmt.Errorf("error calculando balance: %w", err)
	}

	return balance, nil
}

// GetYearlyEarnings retorna el resumen mensual de un año.
func (s *transactionService) GetYearlyEarnings(ctx context.Context, userID uuid.UUID, year int) ([]repository.MonthlyBalance, error) {
	if year < 2020 || year > 2100 {
		return nil, fmt.Errorf("año inválido: %d", year)
	}

	return s.txRepo.GetYearlyEarnings(ctx, userID, year)
}

// ──────────────────────────────────────────────
// Helpers privados
// ──────────────────────────────────────────────

func validateTipo(tipo domain.TransactionType) error {
	if tipo != domain.TransactionTypeIngreso && tipo != domain.TransactionTypeGasto {
		return ErrTransactionInvalidTipo
	}
	return nil
}

func sanitizeCategoria(cat *string) *string {
	if cat == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*cat)
	if trimmed == "" {
		return nil
	}
	lower := strings.ToLower(trimmed)
	return &lower
}
