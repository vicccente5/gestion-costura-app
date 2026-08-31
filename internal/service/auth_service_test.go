// Package service — tests unitarios del servicio de autenticación.
// Usa mocks de la interfaz UserRepository para no depender de la DB real.
// Esto permite correr los tests rápido y en cualquier entorno (CI/CD, local).
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vicccente5/gestion-costura-app/config"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// Mock del UserRepository
// ──────────────────────────────────────────────

// MockUserRepository implementa la interfaz UserRepository con testify/mock.
// Permite definir el comportamiento esperado en cada test de forma declarativa.
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserRepository) FindRefreshToken(ctx context.Context, tokenStr string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, tokenStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *MockUserRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockUserRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// ──────────────────────────────────────────────
// Helpers para los tests
// ──────────────────────────────────────────────

// newTestConfig crea una configuración de prueba con valores fijos.
func newTestConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:              "test-secret-key-of-at-least-32-characters-long",
			AccessExpiryMinutes: 15,
			RefreshExpiryDays:   7,
		},
	}
}

// newTestService crea el servicio con el mock inyectado.
func newTestService(mockRepo *MockUserRepository) service.AuthService {
	return service.NewAuthService(mockRepo, newTestConfig())
}

// ──────────────────────────────────────────────
// Tests de Register
// ──────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// El email no existe aún
	mockRepo.On("FindByEmail", ctx, "ana@mail.com").
		Return(nil, gorm.ErrRecordNotFound)

	// La creación es exitosa
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).
		Return(nil)

	user, err := svc.Register(ctx, "Ana García", "Ana@Mail.com", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	// El email debe quedar en minúsculas (sanitizado)
	assert.Equal(t, "ana@mail.com", user.Email)
	// El nombre debe ser el proporcionado
	assert.Equal(t, "Ana García", user.Nombre)
	// PasswordHash nunca debe ser la contraseña original
	assert.NotEqual(t, "password123", user.PasswordHash)
	assert.NotEmpty(t, user.PasswordHash)

	mockRepo.AssertExpectations(t)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// El email ya existe en la DB
	existingUser := &domain.User{Email: "ana@mail.com"}
	mockRepo.On("FindByEmail", ctx, "ana@mail.com").
		Return(existingUser, nil) // nil error = usuario encontrado

	user, err := svc.Register(ctx, "Ana García", "ana@mail.com", "password123")

	assert.Nil(t, user)
	assert.ErrorIs(t, err, service.ErrEmailAlreadyExists)

	// Create NO debe haber sido llamado
	mockRepo.AssertNotCalled(t, "Create")
	mockRepo.AssertExpectations(t)
}

func TestRegister_EmailSanitizedToLowercase(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// Busca con el email en minúsculas, aunque el input venga con mayúsculas
	mockRepo.On("FindByEmail", ctx, "ana@mail.com").
		Return(nil, gorm.ErrRecordNotFound)
	mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.User")).
		Return(nil)

	// Input con mayúsculas
	user, err := svc.Register(ctx, "Ana", "ANA@MAIL.COM", "password123")

	assert.NoError(t, err)
	// El email guardado debe ser en minúsculas
	assert.Equal(t, "ana@mail.com", user.Email)
}

// ──────────────────────────────────────────────
// Tests de Login
// ──────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// Crear un hash real de bcrypt para el test
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	existingUser := &domain.User{
		ID:           uuid.New(),
		Email:        "ana@mail.com",
		PasswordHash: string(hash),
	}

	mockRepo.On("FindByEmail", ctx, "ana@mail.com").Return(existingUser, nil)
	mockRepo.On("SaveRefreshToken", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)

	tokens, err := svc.Login(ctx, "ana@mail.com", "password123")

	assert.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.True(t, tokens.ExpiresAt.After(time.Now()))

	mockRepo.AssertExpectations(t)
}

func TestLogin_WrongPassword_ReturnsGenericError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), 12)
	existingUser := &domain.User{
		ID:           uuid.New(),
		Email:        "ana@mail.com",
		PasswordHash: string(hash),
	}

	mockRepo.On("FindByEmail", ctx, "ana@mail.com").Return(existingUser, nil)

	tokens, err := svc.Login(ctx, "ana@mail.com", "wrong-password")

	assert.Nil(t, tokens)
	// REGLA CRÍTICA: el error debe ser el genérico, no "contraseña incorrecta"
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestLogin_EmailNotFound_ReturnsGenericError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// Email no existe
	mockRepo.On("FindByEmail", ctx, "noexiste@mail.com").
		Return(nil, gorm.ErrRecordNotFound)

	tokens, err := svc.Login(ctx, "noexiste@mail.com", "cualquier-password")

	assert.Nil(t, tokens)
	// REGLA CRÍTICA: mismo error que contraseña incorrecta — no revelar si el email existe
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestLogin_EmailNotFound_SameErrorAsWrongPassword(t *testing.T) {
	// Este test verifica que los dos casos de error de login son indistinguibles
	// para el cliente — ambos deben retornar ErrInvalidCredentials
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// Caso 1: email no existe
	mockRepo.On("FindByEmail", ctx, "noexiste@mail.com").
		Return(nil, gorm.ErrRecordNotFound)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), 12)
	existingUser := &domain.User{ID: uuid.New(), Email: "existe@mail.com", PasswordHash: string(hash)}
	mockRepo.On("FindByEmail", ctx, "existe@mail.com").Return(existingUser, nil)

	_, err1 := svc.Login(ctx, "noexiste@mail.com", "password")
	_, err2 := svc.Login(ctx, "existe@mail.com", "wrong-password")

	// Ambos errores deben ser idénticos
	assert.Equal(t, err1, err2)
	assert.Equal(t, errors.Is(err1, service.ErrInvalidCredentials), true)
	assert.Equal(t, errors.Is(err2, service.ErrInvalidCredentials), true)
}

// ──────────────────────────────────────────────
// Tests de ValidateAccessToken
// ──────────────────────────────────────────────

func TestValidateAccessToken_ValidToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)
	ctx := context.Background()

	// Obtener un token real haciendo login
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	userID := uuid.New()
	existingUser := &domain.User{ID: userID, Email: "ana@mail.com", PasswordHash: string(hash)}
	mockRepo.On("FindByEmail", ctx, "ana@mail.com").Return(existingUser, nil)
	mockRepo.On("SaveRefreshToken", ctx, mock.AnythingOfType("*domain.RefreshToken")).Return(nil)

	tokens, _ := svc.Login(ctx, "ana@mail.com", "password123")

	// Validar el token generado
	extractedID, err := svc.ValidateAccessToken(tokens.AccessToken)

	assert.NoError(t, err)
	assert.Equal(t, userID, extractedID)
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)

	_, err := svc.ValidateAccessToken("esto.no.es.un.jwt.valido")

	assert.Error(t, err)
}

func TestValidateAccessToken_TamperedToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)

	// Token con firma modificada
	tampered := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyJ9.INVALID_SIGNATURE"
	_, err := svc.ValidateAccessToken(tampered)

	assert.Error(t, err)
}

// ──────────────────────────────────────────────
// Tests de RefreshTokens
// ──────────────────────────────────────────────

func TestRefreshTokens_TokenInvalido_Error(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)

	mockRepo.On("FindRefreshToken", mock.Anything, "token-invalido").
		Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.RefreshTokens(context.Background(), "token-invalido")
	assert.ErrorIs(t, err, service.ErrInvalidRefreshToken)
}

func TestRefreshTokens_TokenRevocado_Error(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)

	revokedToken := &domain.RefreshToken{
		ID:        uuid.New(),
		Token:     "token-revocado",
		UserID:    uuid.New(),
		RevokedAt: func() *time.Time { t := time.Now(); return &t }(), // ya fue revocado
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	mockRepo.On("FindRefreshToken", mock.Anything, "token-revocado").Return(revokedToken, nil)
	// Token Rotation: al detectar reuseado, revoca todos
	mockRepo.On("RevokeAllUserRefreshTokens", mock.Anything, revokedToken.UserID).Return(nil)

	_, err := svc.RefreshTokens(context.Background(), "token-revocado")
	assert.ErrorIs(t, err, service.ErrTokenReuseDetected)
}

// ──────────────────────────────────────────────
// Tests de Logout
// ──────────────────────────────────────────────

func TestLogout_TokenValido_OK(t *testing.T) {
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)

	tokenID := uuid.New()
	rt := &domain.RefreshToken{
		ID:        tokenID,
		Token:     "mi-refresh-token",
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	mockRepo.On("FindRefreshToken", mock.Anything, "mi-refresh-token").Return(rt, nil)
	mockRepo.On("RevokeRefreshToken", mock.Anything, tokenID).Return(nil)

	err := svc.Logout(context.Background(), "mi-refresh-token")
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestLogout_TokenNoExiste_Idempotente(t *testing.T) {
	// Logout es idempotente: si el token no existe, no retorna error al cliente
	mockRepo := new(MockUserRepository)
	svc := newTestService(mockRepo)

	mockRepo.On("FindRefreshToken", mock.Anything, "token-inexistente").
		Return(nil, gorm.ErrRecordNotFound)

	err := svc.Logout(context.Background(), "token-inexistente")
	// El handler captura el error y retorna éxito igual — el service sí retorna error,
	// pero el handler lo ignora. Aquí verificamos el comportamiento del service.
	assert.NoError(t, err) // el service NO retorna error, consideramos exitoso
}
