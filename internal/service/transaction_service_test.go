// Package service — tests unitarios del servicio de transacciones.
// Verifica las reglas de negocio críticas:
//  1. source="order" no se puede editar ni eliminar desde la API
//  2. Monto siempre > 0
//  3. Tipo solo puede ser "ingreso" o "gasto"
//  4. Formato de mes YYYY-MM es validado antes de consultar la DB
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// Mock del TransactionRepository
// ──────────────────────────────────────────────

type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, filters repository.TransactionFilters) ([]domain.Transaction, int64, error) {
	args := m.Called(ctx, userID, params, filters)
	return args.Get(0).([]domain.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTransactionRepository) GetMonthlyBalance(ctx context.Context, userID uuid.UUID, month string) (*repository.MonthlyBalance, error) {
	args := m.Called(ctx, userID, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.MonthlyBalance), args.Error(1)
}

func (m *MockTransactionRepository) GetYearlyEarnings(ctx context.Context, userID uuid.UUID, year int) ([]repository.MonthlyBalance, error) {
	args := m.Called(ctx, userID, year)
	return args.Get(0).([]repository.MonthlyBalance), args.Error(1)
}

// ──────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────

func TestTransaction_Create_Valido(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Transaction")).Return(nil)

	tx, err := svc.Create(ctx, userID, service.TransactionCreateInput{
		Tipo:        domain.TransactionTypeGasto,
		Monto:       15000,
		Descripcion: "Compra de hilo",
		Fecha:       time.Now(),
	})

	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, domain.TransactionSourceManual, tx.Source) // siempre manual desde API
	assert.Equal(t, int64(15000), tx.Monto)
}

func TestTransaction_Create_MontoZero_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	_, err := svc.Create(ctx, userID, service.TransactionCreateInput{
		Tipo:        domain.TransactionTypeIngreso,
		Monto:       0, // monto inválido
		Descripcion: "Test",
		Fecha:       time.Now(),
	})

	assert.ErrorIs(t, err, service.ErrTransactionMontoZero)
}

func TestTransaction_Create_TipoInvalido_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	_, err := svc.Create(ctx, userID, service.TransactionCreateInput{
		Tipo:        domain.TransactionType("venta"), // tipo inválido
		Monto:       5000,
		Descripcion: "Test",
		Fecha:       time.Now(),
	})

	assert.ErrorIs(t, err, service.ErrTransactionInvalidTipo)
}

func TestTransaction_Update_SourceOrder_Error(t *testing.T) {
	// REGLA CRÍTICA: las transacciones source="order" no pueden editarse manualmente.
	// Si se pudieran editar, el balance mensual sería incorrecto (el ingreso
	// automático de un encargo podría cambiar su monto sin reflejar el precio real).
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	// Simular una transacción source="order" (generada automáticamente)
	orderTx := &domain.Transaction{
		ID:     txID,
		UserID: userID,
		Source: domain.TransactionSourceOrder, // ← inmutable
		Monto:  20000,
	}

	mockRepo.On("FindByID", ctx, txID, userID).Return(orderTx, nil)

	_, err := svc.Update(ctx, txID, userID, service.TransactionUpdateInput{
		Tipo:        domain.TransactionTypeIngreso,
		Monto:       25000, // intento de cambiar el monto
		Descripcion: "Modificación",
		Fecha:       time.Now(),
	})

	assert.ErrorIs(t, err, service.ErrTransactionNotEditable)
}

func TestTransaction_Delete_SourceOrder_Error(t *testing.T) {
	// Igual que el test anterior pero para DELETE.
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	orderTx := &domain.Transaction{
		ID:     txID,
		UserID: userID,
		Source: domain.TransactionSourceOrder,
	}

	mockRepo.On("FindByID", ctx, txID, userID).Return(orderTx, nil)

	err := svc.Delete(ctx, txID, userID)
	assert.ErrorIs(t, err, service.ErrTransactionNotEditable)
}

func TestTransaction_Delete_SourceManual_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	manualTx := &domain.Transaction{
		ID:     txID,
		UserID: userID,
		Source: domain.TransactionSourceManual, // sí se puede eliminar
	}

	mockRepo.On("FindByID", ctx, txID, userID).Return(manualTx, nil)
	mockRepo.On("Delete", ctx, txID, userID).Return(nil)

	err := svc.Delete(ctx, txID, userID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransaction_GetMonthlyBalance_FormatoInvalido(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	// Formato incorrecto — debería retornar error antes de tocar la DB
	_, err := svc.GetMonthlyBalance(ctx, userID, "09-2024") // formato incorrecto
	assert.ErrorIs(t, err, service.ErrTransactionInvalidMonth)

	// El mock nunca debería ser llamado
	mockRepo.AssertNotCalled(t, "GetMonthlyBalance")
}

func TestTransaction_GetMonthlyBalance_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	expectedBalance := &repository.MonthlyBalance{
		Mes:           "2024-09",
		TotalIngresos: 150000,
		TotalGastos:   30000,
		Balance:       120000,
	}

	mockRepo.On("GetMonthlyBalance", ctx, userID, "2024-09").Return(expectedBalance, nil)

	balance, err := svc.GetMonthlyBalance(ctx, userID, "2024-09")
	assert.NoError(t, err)
	assert.Equal(t, int64(120000), balance.Balance)
	assert.Equal(t, int64(150000), balance.TotalIngresos)
	assert.Equal(t, int64(30000), balance.TotalGastos)
}

func TestTransaction_Categoria_Sanitizada(t *testing.T) {
	// La categoría debe guardarse en minúsculas y sin espacios.
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	categoria := "  Arriendo  " // con espacios y mayúscula
	mockRepo.On("Create", ctx, mock.MatchedBy(func(tx *domain.Transaction) bool {
		return tx.Categoria != nil && *tx.Categoria == "arriendo"
	})).Return(nil)

	_, err := svc.Create(ctx, userID, service.TransactionCreateInput{
		Tipo:        domain.TransactionTypeGasto,
		Monto:       80000,
		Descripcion: "Arriendo del local",
		Categoria:   &categoria,
		Fecha:       time.Now(),
	})

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTransaction_Update_SourceManual_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	manualTx := &domain.Transaction{
		ID:          txID,
		UserID:      userID,
		Source:      domain.TransactionSourceManual,
		Tipo:        domain.TransactionTypeGasto,
		Monto:       10000,
		Descripcion: "Hilo original",
		Fecha:       time.Now(),
	}

	mockRepo.On("FindByID", ctx, txID, userID).Return(manualTx, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*domain.Transaction")).Return(nil)

	updated, err := svc.Update(ctx, txID, userID, service.TransactionUpdateInput{
		Tipo:        domain.TransactionTypeGasto,
		Monto:       12000,
		Descripcion: "Hilo actualizado",
		Fecha:       time.Now(),
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(12000), updated.Monto)
	assert.Equal(t, "Hilo actualizado", updated.Descripcion)
}

func TestTransaction_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	txID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	mockRepo.On("FindByID", ctx, txID, userID).Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.GetByID(ctx, txID, userID)
	assert.ErrorIs(t, err, service.ErrTransactionNotFound)
}

func TestTransaction_GetAll_ConFiltros(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	params := utils.PaginationParams{Page: 1, Limit: 20, Offset: 0}
	filters := repository.TransactionFilters{Tipo: "gasto"}
	txs := []domain.Transaction{{ID: uuid.New(), UserID: userID, Tipo: domain.TransactionTypeGasto}}

	mockRepo.On("FindAll", ctx, userID, params, filters).Return(txs, int64(1), nil)

	result, total, err := svc.GetAll(ctx, userID, params, filters)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
}

func TestTransaction_GetYearlyEarnings_AnoInvalido(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	_, err := svc.GetYearlyEarnings(ctx, userID, 1800) // año inválido
	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "GetYearlyEarnings")
}

func TestTransaction_GetYearlyEarnings_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockTransactionRepository)
	svc := service.NewTransactionService(mockRepo)

	balances := []repository.MonthlyBalance{
		{Mes: "2024-01", TotalIngresos: 50000, TotalGastos: 10000, Balance: 40000},
		{Mes: "2024-02", TotalIngresos: 80000, TotalGastos: 20000, Balance: 60000},
	}

	mockRepo.On("GetYearlyEarnings", ctx, userID, 2024).Return(balances, nil)

	result, err := svc.GetYearlyEarnings(ctx, userID, 2024)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
}
