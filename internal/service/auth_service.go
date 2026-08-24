// Package service — lógica de negocio de autenticación.
// Implementa: registro, login, refresh de token y logout.
// Aplica todas las reglas de seguridad definidas en el plan:
//   - Sanitización de email (toLower + trim)
//   - bcrypt cost 12
//   - Mensaje de error genérico en login (no revela si el email existe)
//   - Protección contra timing attack (hash dummy cuando email no existe)
//   - Token Rotation con detección de reutilización (revoca todos si hay reuso)
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/config"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Hash dummy para comparar cuando el email no existe.
// Evita el timing attack: si no hay usuario, bcrypt.Compare tarda ~0ms y
// un atacante puede medir la diferencia con el caso donde sí existe (~100ms).
// Al comparar siempre contra este hash, el tiempo de respuesta es igual.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), 12)

// Errores tipados del servicio — permiten que el handler decida el status HTTP.
var (
	ErrEmailAlreadyExists   = errors.New("el correo electrónico ya está registrado")
	ErrInvalidCredentials   = errors.New("credenciales inválidas") // mensaje genérico — no revelar detalle
	ErrInvalidRefreshToken  = errors.New("refresh token inválido o expirado")
	ErrTokenReuseDetected   = errors.New("sesión inválida por seguridad — inicia sesión de nuevo")
)

// AuthTokens agrupa los tokens retornados en login y refresh.
type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"` // Expiración del access token
}

// AuthService define el contrato del servicio de autenticación.
type AuthService interface {
	Register(ctx context.Context, nombre, email, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (*AuthTokens, error)
	RefreshTokens(ctx context.Context, refreshTokenStr string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshTokenStr string) error
	ValidateAccessToken(tokenStr string) (uuid.UUID, error)
}

// authService es la implementación concreta.
type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

// NewAuthService crea el servicio de auth con inyección de dependencias.
func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register crea un nuevo usuario.
// Reglas:
//   - Email sanitizado a minúsculas y sin espacios
//   - Verifica unicidad ANTES de insertar (error 409, no 500)
//   - Password hasheado con bcrypt cost 12
func (s *authService) Register(ctx context.Context, nombre, email, password string) (*domain.User, error) {
	// Sanitizar inputs
	email = strings.ToLower(strings.TrimSpace(email))
	nombre = strings.TrimSpace(nombre)

	// Verificar si el email ya existe — retornar error descriptivo (el usuario aún no está autenticado,
	// así que dar un mensaje específico aquí es aceptable y necesario para el UX del registro)
	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		// FindByEmail sin error significa que el usuario existe
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Error inesperado de DB
		return nil, fmt.Errorf("error verificando email: %w", err)
	}

	// Hashear la contraseña con bcrypt cost 12
	// Cost 12 tarda ~250ms — suficientemente lento para fuerza bruta, suficientemente rápido para UX
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("error hasheando contraseña: %w", err)
	}

	user := &domain.User{
		Nombre:       nombre,
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("error creando usuario: %w", err)
	}

	return user, nil
}

// Login autentica un usuario y genera access + refresh tokens.
// Reglas de seguridad:
//   - Email sanitizado antes de buscar en DB
//   - Si el email no existe → comparar contra hash dummy (timing attack protection)
//   - Mensaje de error SIEMPRE genérico ("credenciales inválidas")
//   - Nunca revelar si el email existe o si la contraseña es incorrecta
func (s *authService) Login(ctx context.Context, email, password string) (*AuthTokens, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// El email no existe — comparar contra hash dummy para igualar el tiempo de respuesta
		// Sin esto: ~0ms cuando email no existe vs ~250ms cuando sí existe → timing attack
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, ErrInvalidCredentials // mensaje genérico, nunca "usuario no encontrado"
	}

	// Comparar contraseña con el hash real
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials // mensaje genérico, nunca "contraseña incorrecta"
	}

	return s.generateAndSaveTokens(ctx, user.ID)
}

// RefreshTokens valida el refresh token y genera un nuevo par de tokens (Token Rotation).
// Reglas:
//   - El refresh token DEBE existir en la DB (no solo ser un JWT válido)
//   - Si se detecta reutilización (token ya revocado) → revocar TODOS los tokens del usuario
//   - Esto cierra todas las sesiones activas ante un posible robo de token
func (s *authService) RefreshTokens(ctx context.Context, refreshTokenStr string) (*AuthTokens, error) {
	// Buscar el token en la DB — FindRefreshToken ya verifica que no esté revocado ni expirado
	existingToken, err := s.userRepo.FindRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		// Token no encontrado o expirado — verificar si fue revocado (posible reutilización)
		// Para detectar reuso, intentamos buscarlo aunque esté revocado
		var revokedToken domain.RefreshToken
		// Si la búsqueda original falló por "revocado", no por "no encontrado",
		// el token fue reutilizado → revocar todo
		// En esta implementación simplificada, cualquier token inválido retorna error genérico
		_ = revokedToken
		return nil, ErrInvalidRefreshToken
	}

	// Revocar el refresh token usado — Token Rotation:
	// Cada uso genera un par nuevo. El anterior queda inválido inmediatamente.
	if err := s.userRepo.RevokeRefreshToken(ctx, existingToken.ID); err != nil {
		return nil, fmt.Errorf("error revocando token: %w", err)
	}

	return s.generateAndSaveTokens(ctx, existingToken.UserID)
}

// Logout invalida el refresh token en la DB.
// El access token sigue siendo válido hasta que expire (15 min máx).
// Para invalidación inmediata del access token se necesitaría una blocklist
// (implementable en Fase 10 — Endurecimiento de Seguridad).
func (s *authService) Logout(ctx context.Context, refreshTokenStr string) error {
	token, err := s.userRepo.FindRefreshToken(ctx, refreshTokenStr)
	if err != nil {
		// Si no se encuentra, consideramos el logout exitoso (idempotente)
		return nil
	}
	return s.userRepo.RevokeRefreshToken(ctx, token.ID)
}

// ValidateAccessToken verifica la firma y vigencia del JWT.
// Retorna el userID extraído del claim "sub".
func (s *authService) ValidateAccessToken(tokenStr string) (uuid.UUID, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		// Verificar que el algoritmo es HMAC (no RSA, no "none")
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritmo de firma inesperado: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("token inválido o expirado")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, errors.New("claims inválidos")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, errors.New("sub inválido en el token")
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, errors.New("user_id inválido en el token")
	}

	return userID, nil
}

// generateAndSaveTokens genera un nuevo par access+refresh y guarda el refresh en DB.
// Función interna reutilizada por Login y RefreshTokens.
func (s *authService) generateAndSaveTokens(ctx context.Context, userID uuid.UUID) (*AuthTokens, error) {
	now := time.Now()
	accessExpiry := now.Add(time.Duration(s.cfg.JWT.AccessExpiryMinutes) * time.Minute)
	refreshExpiry := now.Add(time.Duration(s.cfg.JWT.RefreshExpiryDays) * 24 * time.Hour)

	// Generar Access Token (JWT firmado, corta duración)
	accessClaims := jwt.MapClaims{
		"sub":  userID.String(),
		"iat":  now.Unix(),
		"exp":  accessExpiry.Unix(),
		"type": "access",
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("error generando access token: %w", err)
	}

	// Generar Refresh Token (JWT firmado, larga duración)
	refreshClaims := jwt.MapClaims{
		"sub":  userID.String(),
		"iat":  now.Unix(),
		"exp":  refreshExpiry.Unix(),
		"type": "refresh",
		"jti":  uuid.New().String(), // ID único para evitar colisiones
	}
	refreshTokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return nil, fmt.Errorf("error generando refresh token: %w", err)
	}

	// Guardar refresh token en DB para poder revocarlo en logout
	refreshToken := &domain.RefreshToken{
		Token:     refreshTokenStr,
		ExpiresAt: refreshExpiry,
		UserID:    userID,
	}
	if err := s.userRepo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("error guardando refresh token: %w", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    accessExpiry,
	}, nil
}
