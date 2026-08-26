// Package service — tests unitarios del servicio de materiales.
// El foco principal es verificar el cálculo del promedio ponderado móvil,
// que es la lógica de negocio más crítica de este módulo.
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// Mock del MaterialRepository
// ──────────────────────────────────────────────

type MockMaterialRepository struct {
	mock.Mock
}

func (m *MockMaterialRepository) Create(ctx context.Context, material *domain.Material) error {
	args := m.Called(ctx, material)
	return args.Error(0)
}

func (m *MockMaterialRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Material, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Material), args.Error(1)
}

func (m *MockMaterialRepository) FindByName(ctx context.Context, nombre string, userID uuid.UUID) (*domain.Material, error) {
	args := m.Called(ctx, nombre, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Material), args.Error(1)
}

func (m *MockMaterialRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, categoria string) ([]domain.Material, int64, error) {
	args := m.Called(ctx, userID, params, categoria)
	return args.Get(0).([]domain.Material), args.Get(1).(int64), args.Error(2)
}

func (m *MockMaterialRepository) FindLowStock(ctx context.Context, userID uuid.UUID) ([]domain.Material, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.Material), args.Error(1)
}

func (m *MockMaterialRepository) Update(ctx context.Context, material *domain.Material) error {
	args := m.Called(ctx, material)
	return args.Error(0)
}

func (m *MockMaterialRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMaterialRepository) IsUsedInOrders(ctx context.Context, materialID uuid.UUID) (bool, error) {
	args := m.Called(ctx, materialID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMaterialRepository) CreatePurchase(ctx context.Context, purchase *domain.MaterialPurchase, material *domain.Material) error {
	args := m.Called(ctx, purchase, material)
	return args.Error(0)
}

func (m *MockMaterialRepository) FindPurchasesByMaterialID(ctx context.Context, materialID, userID uuid.UUID) ([]domain.MaterialPurchase, error) {
	args := m.Called(ctx, materialID, userID)
	return args.Get(0).([]domain.MaterialPurchase), args.Error(1)
}

// ──────────────────────────────────────────────
// Tests del promedio ponderado — el corazón de la Fase 4
// ──────────────────────────────────────────────

func TestRegisterPurchase_WeightedAverage_FirstPurchase(t *testing.T) {
	// Primera compra: stock es 0, el nuevo costo = precio de la compra
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	materialID := uuid.New()

	material := &domain.Material{
		ID:            materialID,
		Nombre:        "Tela algodón",
		StockActual:   0,
		CostoUnitario: 0,
		UserID:        userID,
	}

	mockRepo.On("FindByID", ctx, materialID, userID).Return(material, nil)
	mockRepo.On("CreatePurchase", ctx,
		mock.AnythingOfType("*domain.MaterialPurchase"),
		mock.AnythingOfType("*domain.Material")).Return(nil)

	_, updatedMaterial, err := svc.RegisterPurchase(ctx, materialID, userID, service.PurchaseInput{
		Cantidad:       10,     // 10 metros
		PrecioUnitario: 500,    // $500/metro
		Fecha:          time.Now(),
	})

	assert.NoError(t, err)
	assert.Equal(t, 10.0, updatedMaterial.StockActual)
	// Primera compra: costo = precio de compra
	assert.Equal(t, int64(500), updatedMaterial.CostoUnitario)
}

func TestRegisterPurchase_WeightedAverage_SecondPurchase(t *testing.T) {
	// Cálculo del promedio ponderado:
	// Stock existente: 10m a $500/m = $5000
	// Nueva compra: 5m a $600/m = $3000
	// Total: 15m a $533/m promedio (redondeado de 533.33...)
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	materialID := uuid.New()

	material := &domain.Material{
		ID:            materialID,
		StockActual:   10.0,  // ya tiene 10m
		CostoUnitario: 500,   // a $500/m
		UserID:        userID,
	}

	mockRepo.On("FindByID", ctx, materialID, userID).Return(material, nil)
	mockRepo.On("CreatePurchase", ctx,
		mock.AnythingOfType("*domain.MaterialPurchase"),
		mock.AnythingOfType("*domain.Material")).Return(nil)

	_, updatedMaterial, err := svc.RegisterPurchase(ctx, materialID, userID, service.PurchaseInput{
		Cantidad:       5,
		PrecioUnitario: 600,
		Fecha:          time.Now(),
	})

	assert.NoError(t, err)
	assert.Equal(t, 15.0, updatedMaterial.StockActual)
	// (10*500 + 5*600) / 15 = (5000+3000)/15 = 8000/15 = 533.33 → 533
	assert.Equal(t, int64(533), updatedMaterial.CostoUnitario)
}

func TestRegisterPurchase_WeightedAverage_ThirdPurchase(t *testing.T) {
	// Tercer escenario: verifica que el promedio ponderado acumula correctamente
	// Stock: 15m a $533/m
	// Nueva compra: 10m a $400/m
	// (15*533 + 10*400) / 25 = (7995+4000)/25 = 11995/25 = 479.8 → 480
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	materialID := uuid.New()

	material := &domain.Material{
		ID:            materialID,
		StockActual:   15.0,
		CostoUnitario: 533,
		UserID:        userID,
	}

	mockRepo.On("FindByID", ctx, materialID, userID).Return(material, nil)
	mockRepo.On("CreatePurchase", ctx,
		mock.AnythingOfType("*domain.MaterialPurchase"),
		mock.AnythingOfType("*domain.Material")).Return(nil)

	_, updatedMaterial, err := svc.RegisterPurchase(ctx, materialID, userID, service.PurchaseInput{
		Cantidad:       10,
		PrecioUnitario: 400,
		Fecha:          time.Now(),
	})

	assert.NoError(t, err)
	assert.Equal(t, 25.0, updatedMaterial.StockActual)
	assert.Equal(t, int64(480), updatedMaterial.CostoUnitario)
}

func TestRegisterPurchase_PrecioTotal_Calculado(t *testing.T) {
	// Verificar que el precio_total se calcula correctamente
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	materialID := uuid.New()

	material := &domain.Material{ID: materialID, StockActual: 0, CostoUnitario: 0, UserID: userID}
	mockRepo.On("FindByID", ctx, materialID, userID).Return(material, nil)

	var capturedPurchase *domain.MaterialPurchase
	mockRepo.On("CreatePurchase", ctx,
		mock.MatchedBy(func(p *domain.MaterialPurchase) bool {
			capturedPurchase = p
			return true
		}),
		mock.MatchedBy(func(m *domain.Material) bool { return true })).Return(nil)

	svc.RegisterPurchase(ctx, materialID, userID, service.PurchaseInput{
		Cantidad:       7,
		PrecioUnitario: 300,
		Fecha:          time.Now(),
	})

	// precio_total = 7 * 300 = 2100
	assert.Equal(t, int64(2100), capturedPurchase.PrecioTotal)
}

func TestRegisterPurchase_InvalidQuantity(t *testing.T) {
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()

	_, _, err := svc.RegisterPurchase(ctx, uuid.New(), uuid.New(), service.PurchaseInput{
		Cantidad:       0, // inválido
		PrecioUnitario: 500,
		Fecha:          time.Now(),
	})

	assert.ErrorIs(t, err, service.ErrPurchaseQuantityInvalid)
	mockRepo.AssertNotCalled(t, "FindByID")
}

func TestRegisterPurchase_InvalidPrice(t *testing.T) {
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()

	_, _, err := svc.RegisterPurchase(ctx, uuid.New(), uuid.New(), service.PurchaseInput{
		Cantidad:       5,
		PrecioUnitario: -100, // inválido
		Fecha:          time.Now(),
	})

	assert.ErrorIs(t, err, service.ErrPurchasePriceInvalid)
}

// ──────────────────────────────────────────────
// Tests de Create
// ──────────────────────────────────────────────

func TestMaterialCreate_NameDuplicate(t *testing.T) {
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	existingMaterial := &domain.Material{ID: uuid.New(), Nombre: "Tela algodón"}
	mockRepo.On("FindByName", ctx, "Tela algodón", userID).Return(existingMaterial, nil)

	_, err := svc.Create(ctx, userID, service.MaterialInput{
		Nombre:    "Tela algodón",
		Categoria: "Telas",
		Unidad:    "metros",
	})

	assert.ErrorIs(t, err, service.ErrMaterialNameDuplicate)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestMaterialCreate_StockInitialIsZero(t *testing.T) {
	// El stock inicial siempre debe ser 0 — se alimenta con compras
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("FindByName", ctx, "Hilo negro", userID).Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Material")).Return(nil)

	material, err := svc.Create(ctx, userID, service.MaterialInput{
		Nombre:    "Hilo negro",
		Categoria: "Hilos",
		Unidad:    "rollos",
	})

	assert.NoError(t, err)
	assert.Equal(t, 0.0, material.StockActual)
}

// ──────────────────────────────────────────────
// Tests de Delete
// ──────────────────────────────────────────────

func TestMaterialDelete_BlockedWhenUsedInOrders(t *testing.T) {
	mockRepo := new(MockMaterialRepository)
	svc := service.NewMaterialService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	materialID := uuid.New()

	mockRepo.On("FindByID", ctx, materialID, userID).
		Return(&domain.Material{ID: materialID}, nil)
	mockRepo.On("IsUsedInOrders", ctx, materialID).Return(true, nil)

	err := svc.Delete(ctx, materialID, userID)

	assert.ErrorIs(t, err, service.ErrMaterialUsedInOrders)
	mockRepo.AssertNotCalled(t, "Delete")
}
