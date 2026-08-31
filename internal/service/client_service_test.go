// Package service — tests unitarios del servicio de clientes.
package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// Mock del ClientRepository
// ──────────────────────────────────────────────

type MockClientRepository struct {
	mock.Mock
}

func (m *MockClientRepository) Create(ctx context.Context, client *domain.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *MockClientRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Client), args.Error(1)
}

func (m *MockClientRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams) ([]domain.Client, int64, error) {
	args := m.Called(ctx, userID, params)
	return args.Get(0).([]domain.Client), args.Get(1).(int64), args.Error(2)
}

func (m *MockClientRepository) FindByEmail(ctx context.Context, email string, userID uuid.UUID) (*domain.Client, error) {
	args := m.Called(ctx, email, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Client), args.Error(1)
}

func (m *MockClientRepository) Update(ctx context.Context, client *domain.Client) error {
	args := m.Called(ctx, client)
	return args.Error(0)
}

func (m *MockClientRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockClientRepository) HasActiveOrders(ctx context.Context, clientID uuid.UUID) (bool, error) {
	args := m.Called(ctx, clientID)
	return args.Bool(0), args.Error(1)
}

func (m *MockClientRepository) FindOrdersByClientID(ctx context.Context, clientID, userID uuid.UUID) ([]domain.Order, error) {
	args := m.Called(ctx, clientID, userID)
	return args.Get(0).([]domain.Order), args.Error(1)
}

// ──────────────────────────────────────────────
// Tests de Create
// ──────────────────────────────────────────────

func TestClientCreate_Success(t *testing.T) {
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	// Email no existe para este usuario
	mockRepo.On("FindByEmail", ctx, "ana@mail.com", userID).
		Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Client")).
		Return(nil)

	email := "ana@mail.com"
	client, err := svc.Create(ctx, userID, service.ClientInput{
		Nombre: "Ana García",
		Email:  &email,
	})

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "Ana García", client.Nombre)
	assert.Equal(t, userID, client.UserID)
}

func TestClientCreate_EmailSanitizedToLowercase(t *testing.T) {
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	// Se busca con el email ya en minúsculas
	mockRepo.On("FindByEmail", ctx, "ana@mail.com", userID).
		Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Client")).
		Return(nil)

	emailUpper := "ANA@MAIL.COM"
	client, err := svc.Create(ctx, userID, service.ClientInput{
		Nombre: "Ana",
		Email:  &emailUpper,
	})

	assert.NoError(t, err)
	// El email guardado debe estar en minúsculas
	assert.Equal(t, "ana@mail.com", *client.Email)
}

func TestClientCreate_EmailDuplicateForSameUser(t *testing.T) {
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()

	// El email ya existe para este usuario
	existingClient := &domain.Client{ID: uuid.New(), Email: strPtr("ana@mail.com")}
	mockRepo.On("FindByEmail", ctx, "ana@mail.com", userID).
		Return(existingClient, nil)

	email := "ana@mail.com"
	client, err := svc.Create(ctx, userID, service.ClientInput{
		Nombre: "Otra Ana",
		Email:  &email,
	})

	assert.Nil(t, client)
	assert.ErrorIs(t, err, service.ErrClientEmailDuplicate)
	// Create NO debe haberse llamado
	mockRepo.AssertNotCalled(t, "Create")
}

func TestClientCreate_SameEmailDifferentUsers_IsAllowed(t *testing.T) {
	// El mismo email puede existir para DISTINTOS usuarios
	// (dos costureras pueden tener la misma clienta)
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Para user1: email no existe
	mockRepo.On("FindByEmail", ctx, "clienta@mail.com", user1).
		Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Client")).
		Return(nil).Times(2)

	// Para user2: email no existe (distintos espacios de datos)
	mockRepo.On("FindByEmail", ctx, "clienta@mail.com", user2).
		Return(nil, gorm.ErrRecordNotFound)

	email := "clienta@mail.com"
	_, err1 := svc.Create(ctx, user1, service.ClientInput{Nombre: "Cliente A", Email: &email})
	email2 := "clienta@mail.com"
	_, err2 := svc.Create(ctx, user2, service.ClientInput{Nombre: "Cliente A", Email: &email2})

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

// ──────────────────────────────────────────────
// Tests de Delete
// ──────────────────────────────────────────────

func TestClientDelete_BlockedByActiveOrders(t *testing.T) {
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	existingClient := &domain.Client{ID: clientID}
	mockRepo.On("FindByID", ctx, clientID, userID).Return(existingClient, nil)
	// Tiene encargos activos
	mockRepo.On("HasActiveOrders", ctx, clientID).Return(true, nil)

	err := svc.Delete(ctx, clientID, userID)

	assert.ErrorIs(t, err, service.ErrClientHasActiveOrders)
	// Delete no debe haberse llamado
	mockRepo.AssertNotCalled(t, "Delete")
}

func TestClientDelete_SuccessWhenNoActiveOrders(t *testing.T) {
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	existingClient := &domain.Client{ID: clientID}
	mockRepo.On("FindByID", ctx, clientID, userID).Return(existingClient, nil)
	mockRepo.On("HasActiveOrders", ctx, clientID).Return(false, nil)
	mockRepo.On("Delete", ctx, clientID, userID).Return(nil)

	err := svc.Delete(ctx, clientID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestClientDelete_NotFound(t *testing.T) {
	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	mockRepo.On("FindByID", ctx, clientID, userID).Return(nil, gorm.ErrRecordNotFound)

	err := svc.Delete(ctx, clientID, userID)

	assert.ErrorIs(t, err, service.ErrClientNotFound)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func strPtr(s string) *string { return &s }

func TestClient_GetByID_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)

	client := &domain.Client{ID: clientID, UserID: userID, Nombre: "Ana"}
	mockRepo.On("FindByID", ctx, clientID, userID).Return(client, nil)

	result, err := svc.GetByID(ctx, clientID, userID)
	assert.NoError(t, err)
	assert.Equal(t, clientID, result.ID)
}

func TestClient_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)

	mockRepo.On("FindByID", ctx, clientID, userID).Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.GetByID(ctx, clientID, userID)
	assert.ErrorIs(t, err, service.ErrClientNotFound)
}

func TestClient_GetAll_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)

	params := utils.PaginationParams{Page: 1, Limit: 10, Offset: 0}
	clients := []domain.Client{{ID: uuid.New(), UserID: userID, Nombre: "Ana"}}
	mockRepo.On("FindAll", ctx, userID, params).Return(clients, int64(1), nil)

	result, total, err := svc.GetAll(ctx, userID, params)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
}

func TestClient_Update_OK(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)

	existing := &domain.Client{ID: clientID, UserID: userID, Nombre: "Vieja", Email: strPtr("old@mail.com")}
	mockRepo.On("FindByID", ctx, clientID, userID).Return(existing, nil)
	mockRepo.On("FindByEmail", ctx, "nueva@mail.com", userID).Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*domain.Client")).Return(nil)

	result, err := svc.Update(ctx, clientID, userID, service.ClientInput{
		Nombre: "Nueva",
		Email:  strPtr("nueva@mail.com"),
	})

	assert.NoError(t, err)
	assert.Equal(t, "Nueva", result.Nombre)
	assert.NotNil(t, result.Email)
	assert.Equal(t, "nueva@mail.com", *result.Email)
}

func TestClient_Update_EmailMismoUsuario_OK(t *testing.T) {
	// Actualizar con el mismo email del cliente → no debe fallar por duplicado
	ctx := context.Background()
	userID := uuid.New()
	clientID := uuid.New()

	mockRepo := new(MockClientRepository)
	svc := service.NewClientService(mockRepo)

	existing := &domain.Client{ID: clientID, UserID: userID, Nombre: "Ana", Email: strPtr("ana@mail.com")}
	// FindByEmail retorna el mismo cliente (mismo ID) → no es duplicado
	mockRepo.On("FindByID", ctx, clientID, userID).Return(existing, nil)
	mockRepo.On("FindByEmail", ctx, "ana@mail.com", userID).Return(existing, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*domain.Client")).Return(nil)

	result, err := svc.Update(ctx, clientID, userID, service.ClientInput{
		Nombre: "Ana Actualizada",
		Email:  strPtr("ana@mail.com"),
	})

	assert.NoError(t, err)
	assert.Equal(t, "Ana Actualizada", result.Nombre)
}
