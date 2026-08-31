// Package handler_test — tests de integración para la capa HTTP.
//
// Estrategia: mockeamos los services usando testify/mock, igual que en service_test.go.
// Esto nos permite probar la capa HTTP de forma aislada y rápida (sin DB real):
//   - Serialización/deserialización JSON correcta
//   - Validación de campos requeridos
//   - Status codes HTTP correctos según la respuesta del service
//   - Que el JWT se valide correctamente en rutas protegidas
//
// Cada test usa httptest.NewRecorder() para capturar la respuesta HTTP.
// Los mocks retornan valores predefinidos para simular casos de éxito y error.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/vicccente5/gestion-costura-app/internal/service"
)

func init() {
	// Usar modo test para que Gin no imprima la tabla de rutas ni los logs de debug
	gin.SetMode(gin.TestMode)
}

// ──────────────────────────────────────────────
// Mock AuthService
// ──────────────────────────────────────────────

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, nombre, email, password string) (*domain.User, error) {
	args := m.Called(ctx, nombre, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*service.AuthTokens, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthTokens), args.Error(1)
}

func (m *MockAuthService) ValidateAccessToken(token string) (uuid.UUID, error) {
	args := m.Called(token)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*service.AuthTokens, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AuthTokens), args.Error(1)
}

func (m *MockAuthService) Logout(ctx context.Context, refreshToken string) error {
	args := m.Called(ctx, refreshToken)
	return args.Error(0)
}

// ──────────────────────────────────────────────
// Helpers de test
// ──────────────────────────────────────────────

// newTestRouter crea un router Gin mínimo con un handler registrado.
// No incluye middlewares de autenticación — los tests de handlers públicos
// no los necesitan. Para rutas protegidas se inyecta el user_id manualmente.
func newTestRouter(method, path string, h gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Handle(method, path, h)
	return r
}

// postJSON envía una petición POST con un body JSON.
func postJSON(router *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// getRequest envía una petición GET.
func getRequest(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ──────────────────────────────────────────────
// Tests: AuthHandler.Register
// ──────────────────────────────────────────────

func TestAuthHandler_Register_Success(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/register", h.Register)

	userID := uuid.New()
	mockSvc.On("Register", mock.Anything, "Ana García", "ana@mail.com", "secreto123").
		Return(&domain.User{
			ID:        userID,
			Nombre:    "Ana García",
			Email:     "ana@mail.com",
			CreatedAt: time.Now(),
		}, nil)

	w := postJSON(router, "/auth/register", map[string]string{
		"nombre":   "Ana García",
		"email":    "ana@mail.com",
		"password": "secreto123",
	})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
	assert.Equal(t, "Cuenta creada exitosamente", resp["message"])

	// Verificar que el password NO está en la respuesta
	data := resp["data"].(map[string]any)
	_, hasPassword := data["password"]
	_, hasPasswordHash := data["password_hash"]
	assert.False(t, hasPassword, "la respuesta no debe incluir password")
	assert.False(t, hasPasswordHash, "la respuesta no debe incluir password_hash")
}

func TestAuthHandler_Register_EmailDuplicado(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/register", h.Register)

	mockSvc.On("Register", mock.Anything, mock.Anything, "existente@mail.com", mock.Anything).
		Return(nil, service.ErrEmailAlreadyExists)

	w := postJSON(router, "/auth/register", map[string]string{
		"nombre":   "Test User",
		"email":    "existente@mail.com",
		"password": "password123",
	})

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
}

func TestAuthHandler_Register_BodyInvalido(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/register", h.Register)

	// Body vacío — debería retornar 400 antes de llamar al service
	w := postJSON(router, "/auth/register", map[string]string{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	// El mock no debería haber sido llamado
	mockSvc.AssertNotCalled(t, "Register")
}

func TestAuthHandler_Register_EmailFormatoInvalido(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/register", h.Register)

	w := postJSON(router, "/auth/register", map[string]string{
		"nombre":   "Test",
		"email":    "no-es-un-email", // formato inválido
		"password": "password123",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "Register")
}

// ──────────────────────────────────────────────
// Tests: AuthHandler.Login
// ──────────────────────────────────────────────

func TestAuthHandler_Login_Success(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/login", h.Login)

	mockSvc.On("Login", mock.Anything, "ana@mail.com", "secreto123").
		Return(&service.AuthTokens{
			AccessToken:  "access.jwt.token",
			RefreshToken: "refresh.jwt.token",
		}, nil)

	w := postJSON(router, "/auth/login", map[string]string{
		"email":    "ana@mail.com",
		"password": "secreto123",
	})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))

	// Verificar que el token está en la respuesta
	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
}

func TestAuthHandler_Login_CredencialesInvalidas(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/login", h.Login)

	mockSvc.On("Login", mock.Anything, "ana@mail.com", "mal_password").
		Return(nil, service.ErrInvalidCredentials)

	w := postJSON(router, "/auth/login", map[string]string{
		"email":    "ana@mail.com",
		"password": "mal_password",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Verificar que el mensaje de error es genérico (no revela qué campo falló)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
	errMsg := resp["error"].(string)
	assert.NotContains(t, errMsg, "password", "el error no debe mencionar 'password'")
	assert.NotContains(t, errMsg, "email", "el error no debe mencionar 'email'")
}

func TestAuthHandler_Login_BodyInvalido(t *testing.T) {
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/login", h.Login)

	// JSON malformado
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("esto no es json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "Login")
}

// ──────────────────────────────────────────────
// Tests: formato de respuesta estandarizado
// ──────────────────────────────────────────────

func TestRespuestaJSON_FormatoEstandar_Exito(t *testing.T) {
	// Verificar que el formato de respuesta es consistente en todos los endpoints exitosos
	// { "success": true, "message": "...", "data": {...} }
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/login", h.Login)

	mockSvc.On("Login", mock.Anything, "test@test.com", "test1234").
		Return(&service.AuthTokens{AccessToken: "t", RefreshToken: "r"}, nil)

	w := postJSON(router, "/auth/login", map[string]string{
		"email": "test@test.com", "password": "test1234",
	})

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "la respuesta debe ser JSON válido")

	// Campos obligatorios en toda respuesta exitosa
	assert.Contains(t, resp, "success")
	assert.Contains(t, resp, "message")
	assert.Equal(t, true, resp["success"])

	// Los campos de error no deben estar en una respuesta exitosa
	_, hasError := resp["error"]
	assert.False(t, hasError)
}

func TestRespuestaJSON_FormatoEstandar_Error(t *testing.T) {
	// Verificar que el formato de error es consistente
	// { "success": false, "error": "...", "code": 4xx }
	mockSvc := new(MockAuthService)
	h := handler.NewAuthHandler(mockSvc)
	router := newTestRouter(http.MethodPost, "/auth/login", h.Login)

	w := postJSON(router, "/auth/login", map[string]string{}) // body vacío = 400

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "los errores también deben ser JSON válido")

	// Campos obligatorios en toda respuesta de error
	assert.Contains(t, resp, "success")
	assert.Contains(t, resp, "error")
	assert.Contains(t, resp, "code")
	assert.Equal(t, false, resp["success"])
	assert.Equal(t, float64(400), resp["code"])
}
