// Package handler_test — tests de integración para TransactionHandler.
// Verifica status codes, validaciones de input y la regla crítica:
// source="order" no puede editarse/eliminarse desde la API.
package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/handler"
	"github.com/vicccente5/gestion-costura-app/internal/middleware"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
)

// ──────────────────────────────────────────────
// Mock TransactionService
// ──────────────────────────────────────────────

type MockTransactionSvc struct {
	mock.Mock
}

func (m *MockTransactionSvc) Create(ctx context.Context, userID uuid.UUID, input service.TransactionCreateInput) (*domain.Transaction, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionSvc) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Transaction, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionSvc) GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, filters repository.TransactionFilters) ([]domain.Transaction, int64, error) {
	args := m.Called(ctx, userID, params, filters)
	return args.Get(0).([]domain.Transaction), args.Get(1).(int64), args.Error(2)
}

func (m *MockTransactionSvc) Update(ctx context.Context, id, userID uuid.UUID, input service.TransactionUpdateInput) (*domain.Transaction, error) {
	args := m.Called(ctx, id, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Transaction), args.Error(1)
}

func (m *MockTransactionSvc) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockTransactionSvc) GetMonthlyBalance(ctx context.Context, userID uuid.UUID, month string) (*repository.MonthlyBalance, error) {
	args := m.Called(ctx, userID, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.MonthlyBalance), args.Error(1)
}

func (m *MockTransactionSvc) GetYearlyEarnings(ctx context.Context, userID uuid.UUID, year int) ([]repository.MonthlyBalance, error) {
	args := m.Called(ctx, userID, year)
	return args.Get(0).([]repository.MonthlyBalance), args.Error(1)
}

// ──────────────────────────────────────────────
// Helper: router autenticado con user_id inyectado
// ──────────────────────────────────────────────

// newAuthenticatedRouter crea un router con el user_id ya inyectado en el contexto.
// Simula que el AuthMiddleware ya validó el token — así podemos testear handlers protegidos
// sin necesidad de generar un JWT real en cada test.
func newAuthenticatedRouter(method, path string, userID uuid.UUID, h gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Handle(method, path, func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID) // inyectar user_id como si AuthMiddleware hubiera validado
		h(c)
	})
	return r
}

// ──────────────────────────────────────────────
// Tests: TransactionHandler.Create
// ──────────────────────────────────────────────

func TestTransactionHandler_Create_Exito(t *testing.T) {
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	router := newAuthenticatedRouter(http.MethodPost, "/transactions", userID, h.Create)

	txID := uuid.New()
	mockSvc.On("Create", mock.Anything, userID, mock.AnythingOfType("service.TransactionCreateInput")).
		Return(&domain.Transaction{
			ID:          txID,
			Tipo:        domain.TransactionTypeGasto,
			Monto:       15000,
			Descripcion: "Compra de hilo",
			Source:      domain.TransactionSourceManual,
			UserID:      userID,
			Fecha:       time.Now(),
		}, nil)

	w := postJSON(router, "/transactions", map[string]any{
		"tipo":        "gasto",
		"monto":       15000,
		"descripcion": "Compra de hilo",
		"fecha":       "2024-09-15",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestTransactionHandler_Create_MontoNegativo_BadRequest(t *testing.T) {
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	router := newAuthenticatedRouter(http.MethodPost, "/transactions", userID, h.Create)

	w := postJSON(router, "/transactions", map[string]any{
		"tipo":        "gasto",
		"monto":       -500, // inválido
		"descripcion": "Test",
		"fecha":       "2024-09-15",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

func TestTransactionHandler_Create_FechaInvalida_BadRequest(t *testing.T) {
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	router := newAuthenticatedRouter(http.MethodPost, "/transactions", userID, h.Create)

	w := postJSON(router, "/transactions", map[string]any{
		"tipo":        "ingreso",
		"monto":       5000,
		"descripcion": "Test",
		"fecha":       "15/09/2024", // formato incorrecto, debe ser YYYY-MM-DD
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "Create")
}

// ──────────────────────────────────────────────
// Tests: TransactionHandler.Delete (source="order" protegida)
// ──────────────────────────────────────────────

func TestTransactionHandler_Delete_SourceOrder_Conflict(t *testing.T) {
	// REGLA CRÍTICA: DELETE de una transacción source="order" debe retornar 409
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	txID := uuid.New()
	router := newAuthenticatedRouter(http.MethodDelete, "/transactions/:id", userID, h.Delete)

	mockSvc.On("Delete", mock.Anything, txID, userID).
		Return(service.ErrTransactionNotEditable)

	req := httptest.NewRequest(http.MethodDelete, "/transactions/"+txID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ──────────────────────────────────────────────
// Tests: TransactionHandler.GetBalance
// ──────────────────────────────────────────────

func TestTransactionHandler_GetBalance_MesActual(t *testing.T) {
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	router := newAuthenticatedRouter(http.MethodGet, "/transactions/balance", userID, h.GetBalance)

	currentMonth := time.Now().Format("2006-01")
	mockSvc.On("GetMonthlyBalance", mock.Anything, userID, currentMonth).
		Return(&repository.MonthlyBalance{
			Mes:           currentMonth,
			TotalIngresos: 100000,
			TotalGastos:   30000,
			Balance:       70000,
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/transactions/balance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTransactionHandler_GetBalance_MesEspecifico(t *testing.T) {
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	router := newAuthenticatedRouter(http.MethodGet, "/transactions/balance", userID, h.GetBalance)

	mockSvc.On("GetMonthlyBalance", mock.Anything, userID, "2024-09").
		Return(&repository.MonthlyBalance{
			Mes:     "2024-09",
			Balance: 50000,
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/transactions/balance?month=2024-09", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTransactionHandler_GetBalance_FormatoMesInvalido(t *testing.T) {
	mockSvc := new(MockTransactionSvc)
	h := handler.NewTransactionHandler(mockSvc)
	userID := uuid.New()
	router := newAuthenticatedRouter(http.MethodGet, "/transactions/balance", userID, h.GetBalance)

	mockSvc.On("GetMonthlyBalance", mock.Anything, userID, "09-2024").
		Return(nil, service.ErrTransactionInvalidMonth)

	req := httptest.NewRequest(http.MethodGet, "/transactions/balance?month=09-2024", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
