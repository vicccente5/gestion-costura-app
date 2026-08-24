// Package repository — interfaz e implementación del repositorio de usuarios.
// La interfaz permite hacer mocks en los tests sin depender de la DB real.
// La implementación usa GORM con PostgreSQL.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"gorm.io/gorm"
)

// UserRepository define el contrato de acceso a datos para usuarios.
// El service solo conoce esta interfaz, nunca la implementación concreta.
// Esto desacopla la lógica de negocio del motor de base de datos.
type UserRepository interface {
	// Create crea un nuevo usuario en la DB.
	Create(ctx context.Context, user *domain.User) error

	// FindByEmail busca un usuario por email (ya normalizado a minúsculas).
	// Retorna gorm.ErrRecordNotFound si no existe.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindByID busca un usuario por su UUID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// SaveRefreshToken guarda un nuevo refresh token en la DB.
	SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error

	// FindRefreshToken busca un refresh token por su valor string.
	// Retorna gorm.ErrRecordNotFound si no existe o está revocado.
	FindRefreshToken(ctx context.Context, tokenStr string) (*domain.RefreshToken, error)

	// RevokeRefreshToken marca un refresh token como revocado (logout).
	RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error

	// RevokeAllUserRefreshTokens revoca TODOS los tokens de un usuario.
	// Se usa en detección de Token Rotation (reutilización sospechosa).
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

// userRepository es la implementación concreta con GORM + PostgreSQL.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository crea una nueva instancia del repositorio.
// Recibe *gorm.DB por inyección de dependencias desde el router/main.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	result := r.db.WithContext(ctx).Create(user)
	return result.Error
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *userRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *userRepository) FindRefreshToken(ctx context.Context, tokenStr string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	result := r.db.WithContext(ctx).
		Where("token = ? AND revoked_at IS NULL", tokenStr).
		First(&token)
	if result.Error != nil {
		return nil, result.Error
	}
	// Verificar que no haya expirado
	if token.IsExpired() {
		return nil, errors.New("refresh token expirado")
	}
	return &token, nil
}

func (r *userRepository) RevokeRefreshToken(ctx context.Context, tokenID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ?", tokenID).
		Update("revoked_at", gorm.Expr("NOW()")).
		Error
}

func (r *userRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", gorm.Expr("NOW()")).
		Error
}
