package service_test

import (
	"context"
	"testing"

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
// Mocks
// ──────────────────────────────────────────────

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) CreateWithMaterials(ctx context.Context, order *domain.Order, items []repository.OrderItem) error {
	args := m.Called(ctx, order, items)
	return args.Error(0)
}

func (m *MockOrderRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Order, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockOrderRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, estado string) ([]domain.Order, int64, error) {
	args := m.Called(ctx, userID, params, estado)
	return args.Get(0).([]domain.Order), args.Get(1).(int64), args.Error(2)
}

func (m *MockOrderRepository) UpdateMetadata(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) UpdateStatus(ctx context.Context, order *domain.Order, newStatus domain.OrderStatus, transaction *domain.Transaction) error {
	args := m.Called(ctx, order, newStatus, transaction)
	return args.Error(0)
}

func (m *MockOrderRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockOrderRepository) RestoreStock(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) AddMaterial(ctx context.Context, orderID, userID uuid.UUID, item repository.OrderItem) (*domain.OrderMaterial, error) {
	args := m.Called(ctx, orderID, userID, item)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OrderMaterial), args.Error(1)
}

func (m *MockOrderRepository) EditMaterialQuantity(ctx context.Context, orderMaterialID, userID uuid.UUID, nuevaCantidad float64) error {
	args := m.Called(ctx, orderMaterialID, userID, nuevaCantidad)
	return args.Error(0)
}

func (m *MockOrderRepository) RemoveMaterial(ctx context.Context, orderMaterialID, userID uuid.UUID) error {
	args := m.Called(ctx, orderMaterialID, userID)
	return args.Error(0)
}

type MockClientRepositoryForOrder struct {
	mock.Mock
}

// Implementación mínima para satisfacer interface de pruebas o si lo pasamos directo,
// aquí implementamos solo lo que OrderService usa: FindByID.
func (m *MockClientRepositoryForOrder) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Client), args.Error(1)
}

// Como las interfaces en Go se implementan implícitamente, agregaremos las demás para compilar
func (m *MockClientRepositoryForOrder) Create(ctx context.Context, client *domain.Client) error { return nil }
func (m *MockClientRepositoryForOrder) FindByEmail(ctx context.Context, email string, userID uuid.UUID) (*domain.Client, error) { return nil, nil }
func (m *MockClientRepositoryForOrder) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams) ([]domain.Client, int64, error) { return nil, 0, nil }
func (m *MockClientRepositoryForOrder) Update(ctx context.Context, client *domain.Client) error { return nil }
func (m *MockClientRepositoryForOrder) Delete(ctx context.Context, id, userID uuid.UUID) error { return nil }
func (m *MockClientRepositoryForOrder) HasActiveOrders(ctx context.Context, id uuid.UUID) (bool, error) { return false, nil }
func (m *MockClientRepositoryForOrder) GetOrders(ctx context.Context, clientID, userID uuid.UUID, params utils.PaginationParams) ([]domain.Order, int64, error) { return nil, 0, nil }



// ──────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────

func TestChangeStatus_ValidTransitions(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusPendiente,
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)
	// Validar que actualiza el estado y transaction es nulo al pasar a en progreso
	mockOrderRepo.On("UpdateStatus", ctx, order, domain.OrderStatusEnProgreso, (*domain.Transaction)(nil)).Return(nil)

	updatedOrder, err := svc.ChangeStatus(ctx, orderID, userID, domain.OrderStatusEnProgreso)
	assert.NoError(t, err)
	assert.Equal(t, domain.OrderStatusEnProgreso, updatedOrder.Estado)
}

func TestChangeStatus_Entregado_GeneratesTransaction(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:          orderID,
		UserID:      userID,
		Estado:      domain.OrderStatusCompletado, // listo para entregar
		PrecioVenta: 15000,
		Horas:       2.5,
		TarifaHora:  2000,
		Descripcion: "Basta de pantalón",
		Materials: []domain.OrderMaterial{
			{
				Cantidad:              1.5,
				CostoUnitarioSnapshot: 1000,
			},
		},
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	// Validar que se pasa una Transaction con source="order" y monto = precio_venta
	mockOrderRepo.On("UpdateStatus", ctx, order, domain.OrderStatusEntregado, mock.MatchedBy(func(tx *domain.Transaction) bool {
		return tx.Monto == 15000 &&
			tx.Source == domain.TransactionSourceOrder &&
			tx.Tipo == domain.TransactionTypeIngreso &&
			tx.OrderID != nil && *tx.OrderID == orderID
	})).Return(nil)

	updatedOrder, err := svc.ChangeStatus(ctx, orderID, userID, domain.OrderStatusEntregado)
	assert.NoError(t, err)
	assert.Equal(t, domain.OrderStatusEntregado, updatedOrder.Estado)
}

func TestChangeStatus_Entregado_NoPrice_ReturnsError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:          orderID,
		UserID:      userID,
		Estado:      domain.OrderStatusCompletado,
		PrecioVenta: 0, // 0 = no se puede entregar
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	_, err := svc.ChangeStatus(ctx, orderID, userID, domain.OrderStatusEntregado)
	assert.ErrorIs(t, err, service.ErrOrderNoPrice)
}

func TestChangeStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusPendiente, // no puede saltar a entregado directo
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	_, err := svc.ChangeStatus(ctx, orderID, userID, domain.OrderStatusEntregado)
	assert.ErrorIs(t, err, service.ErrOrderInvalidStatusChange)
}

func TestChangeStatus_Cancelado_RestoresStock(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusEnProgreso,
		Materials: []domain.OrderMaterial{
			{Cantidad: 2}, // hay materiales que restaurar
		},
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)
	mockOrderRepo.On("RestoreStock", ctx, order).Return(nil)
	mockOrderRepo.On("UpdateStatus", ctx, order, domain.OrderStatusCancelado, (*domain.Transaction)(nil)).Return(nil)

	updatedOrder, err := svc.ChangeStatus(ctx, orderID, userID, domain.OrderStatusCancelado)
	assert.NoError(t, err)
	assert.Equal(t, domain.OrderStatusCancelado, updatedOrder.Estado)
	mockOrderRepo.AssertExpectations(t)
}

func TestAddMaterial_InsufficientStock(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()
	materialID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	mockMatRepo := new(MockMaterialRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, mockMatRepo)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusPendiente,
	}

	material := &domain.Material{
		ID:          materialID,
		StockActual: 2.0, // Solo quedan 2
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)
	mockMatRepo.On("FindByID", ctx, materialID, userID).Return(material, nil)

	_, err := svc.AddMaterial(ctx, orderID, userID, service.OrderMaterialInput{
		MaterialID: materialID,
		Cantidad:   5.0, // pide 5
	})

	assert.ErrorIs(t, err, service.ErrInsufficientStock)
}

func TestAddMaterial_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()
	materialID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	mockMatRepo := new(MockMaterialRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, mockMatRepo)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusPendiente,
	}

	material := &domain.Material{
		ID:           materialID,
		Nombre:       "Tela algodón",
		StockActual:  10.0,
		CostoUnitario: 1500,
	}

	om := &domain.OrderMaterial{ID: uuid.New(), OrderID: orderID, MaterialID: materialID, Cantidad: 3}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)
	mockMatRepo.On("FindByID", ctx, materialID, userID).Return(material, nil)
	mockOrderRepo.On("AddMaterial", ctx, orderID, userID, mock.AnythingOfType("repository.OrderItem")).Return(om, nil)

	result, err := svc.AddMaterial(ctx, orderID, userID, service.OrderMaterialInput{
		MaterialID: materialID,
		Cantidad:   3.0,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockOrderRepo.AssertExpectations(t)
}

func TestAddMaterial_OrderNotPendiente_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	// Encargo ya en progreso — no se pueden agregar materiales
	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusEnProgreso, // no es pendiente
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	_, err := svc.AddMaterial(ctx, orderID, userID, service.OrderMaterialInput{
		MaterialID: uuid.New(),
		Cantidad:   1.0,
	})

	assert.ErrorIs(t, err, service.ErrOrderNotEditable)
}

func TestEditMaterialQuantity_OrderNotPendiente_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusCompletado, // no editable
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	err := svc.EditMaterialQuantity(ctx, orderID, uuid.New(), userID, 5.0)
	assert.ErrorIs(t, err, service.ErrOrderNotEditable)
}

func TestRemoveMaterial_OrderNotPendiente_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusEntregado, // terminal, no editable
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	err := svc.RemoveMaterial(ctx, orderID, uuid.New(), userID)
	assert.ErrorIs(t, err, service.ErrOrderNotEditable)
}

func TestDelete_OrderEntregado_Error(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusEntregado, // no se puede borrar
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	err := svc.Delete(ctx, orderID, userID)
	assert.ErrorIs(t, err, service.ErrOrderNotDeletable)
}

func TestGetAll_ReturnsOrders(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	params := utils.PaginationParams{Page: 1, Limit: 10, Offset: 0}
	orders := []domain.Order{{ID: uuid.New(), UserID: userID}}

	mockOrderRepo.On("FindAll", ctx, userID, params, "").Return(orders, int64(1), nil)

	result, total, err := svc.GetAll(ctx, userID, params, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
}

func TestOrder_GetByID_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{ID: orderID, UserID: userID}
	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)

	result, err := svc.GetByID(ctx, orderID, userID)
	assert.NoError(t, err)
	assert.Equal(t, orderID, result.ID)
}

func TestOrder_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.GetByID(ctx, orderID, userID)
	assert.ErrorIs(t, err, service.ErrOrderNotFound)
}

func TestOrder_UpdateMetadata_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	orderID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	svc := service.NewOrderService(mockOrderRepo, nil, nil)

	order := &domain.Order{
		ID:     orderID,
		UserID: userID,
		Estado: domain.OrderStatusPendiente, // Solo pendiente es editable
	}

	mockOrderRepo.On("FindByID", ctx, orderID, userID).Return(order, nil)
	mockOrderRepo.On("UpdateMetadata", ctx, mock.AnythingOfType("*domain.Order")).Return(nil)

	result, err := svc.UpdateMetadata(ctx, orderID, userID, service.OrderUpdateInput{
		Descripcion: "Nueva descripcion",
		PrecioVenta: 20000,
	})

	assert.NoError(t, err)
	assert.Equal(t, "Nueva descripcion", result.Descripcion)
	assert.Equal(t, int64(20000), result.PrecioVenta)
}

func TestOrder_Create_SinMateriales_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	mockOrderRepo := new(MockOrderRepository)
	mockClientRepo := new(MockClientRepository)
	svc := service.NewOrderService(mockOrderRepo, mockClientRepo, nil)

	client := &domain.Client{ID: clientID, UserID: userID}
	mockClientRepo.On("FindByID", ctx, clientID, userID).Return(client, nil)

	mockOrderRepo.On("CreateWithMaterials", ctx, mock.AnythingOfType("*domain.Order"), mock.Anything).Return(nil)

	input := service.OrderCreateInput{
		Descripcion: "Vestido",
		PrecioVenta: 50000,
		ClientID:    clientID,
	}

	result, err := svc.Create(ctx, userID, input)
	assert.NoError(t, err)
	assert.Equal(t, "Vestido", result.Descripcion)
	assert.Equal(t, domain.OrderStatusPendiente, result.Estado)
}
