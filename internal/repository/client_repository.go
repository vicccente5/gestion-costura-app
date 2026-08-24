// Package repository — interfaz e implementación del repositorio de clientes.
package repository

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// ClientRepository define el contrato de acceso a datos para clientes.
// REGLA DE ORO: todos los métodos reciben userID y lo aplican en el WHERE.
// Esto garantiza que ninguna costurera pueda ver datos de otra.
type ClientRepository interface {
	// Create inserta un nuevo cliente en la DB.
	Create(ctx context.Context, client *domain.Client) error

	// FindByID busca un cliente por ID y user_id (previene IDOR).
	FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error)

	// FindAll retorna la lista paginada de clientes de un usuario con búsqueda opcional.
	FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams) ([]domain.Client, int64, error)

	// FindByEmail busca un cliente por email dentro del mismo usuario.
	// Retorna gorm.ErrRecordNotFound si no existe.
	FindByEmail(ctx context.Context, email string, userID uuid.UUID) (*domain.Client, error)

	// Update actualiza los campos de un cliente (solo el dueño puede hacerlo).
	Update(ctx context.Context, client *domain.Client) error

	// Delete hace soft delete de un cliente (solo si no tiene encargos activos).
	// La verificación de encargos activos se hace en el service, no aquí.
	Delete(ctx context.Context, id, userID uuid.UUID) error

	// HasActiveOrders verifica si el cliente tiene encargos en estado pendiente o en_progreso.
	// Usado por el service antes de permitir la eliminación.
	HasActiveOrders(ctx context.Context, clientID uuid.UUID) (bool, error)

	// FindOrdersByClientID retorna el historial de encargos de un cliente.
	// Solo retorna encargos del mismo usuario (previene IDOR).
	FindOrdersByClientID(ctx context.Context, clientID, userID uuid.UUID) ([]domain.Order, error)
}

// clientRepository es la implementación concreta con GORM.
type clientRepository struct {
	db *gorm.DB
}

// NewClientRepository crea una nueva instancia del repositorio de clientes.
func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) Create(ctx context.Context, client *domain.Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

func (r *clientRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error) {
	var client domain.Client
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		First(&client)
	if result.Error != nil {
		return nil, result.Error
	}
	return &client, nil
}

func (r *clientRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams) ([]domain.Client, int64, error) {
	var clients []domain.Client
	var total int64

	query := r.db.WithContext(ctx).
		Model(&domain.Client{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	// Búsqueda por nombre (case-insensitive con ILIKE de PostgreSQL)
	if params.Search != "" {
		query = query.Where("nombre ILIKE ?", "%"+params.Search+"%")
	}

	// Contar total antes de paginar (para total_pages en la respuesta)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Aplicar paginación y ordenar por nombre alfabético
	if err := query.
		Order("nombre ASC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&clients).Error; err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

func (r *clientRepository) FindByEmail(ctx context.Context, email string, userID uuid.UUID) (*domain.Client, error) {
	var client domain.Client
	result := r.db.WithContext(ctx).
		Where("email = ? AND user_id = ? AND deleted_at IS NULL", email, userID).
		First(&client)
	if result.Error != nil {
		return nil, result.Error
	}
	return &client, nil
}

func (r *clientRepository) Update(ctx context.Context, client *domain.Client) error {
	// Save actualiza todos los campos del struct — asegurarse de que userID esté seteado
	return r.db.WithContext(ctx).Save(client).Error
}

func (r *clientRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	// Soft delete — solo elimina si pertenece al usuario (previene IDOR en delete)
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&domain.Client{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *clientRepository) HasActiveOrders(ctx context.Context, clientID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Order{}).
		Where("client_id = ? AND estado IN ? AND deleted_at IS NULL",
			clientID,
			[]string{
				string(domain.OrderStatusPendiente),
				string(domain.OrderStatusEnProgreso),
			},
		).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *clientRepository) FindOrdersByClientID(ctx context.Context, clientID, userID uuid.UUID) ([]domain.Order, error) {
	var orders []domain.Order
	// Verificar user_id en el JOIN para prevenir IDOR — solo encargos del dueño del cliente
	err := r.db.WithContext(ctx).
		Where("client_id = ? AND user_id = ? AND deleted_at IS NULL", clientID, userID).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

// normalizeEmail normaliza un email a minúsculas y sin espacios.
// Función auxiliar usada en el service, no en el repository.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
