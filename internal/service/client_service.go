// Package service — lógica de negocio del módulo de clientes.
// Aplica todas las reglas de validación y unicidad definidas en el plan:
//   - Sanitización de email y nombre antes de guardar
//   - Unicidad de email POR usuario (no global)
//   - Bloqueo de eliminación si hay encargos activos
//   - user_id siempre requerido para prevenir IDOR
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// Errores tipados del servicio de clientes.
var (
	ErrClientNotFound       = errors.New("cliente no encontrado")
	ErrClientEmailDuplicate = errors.New("ya tienes un cliente registrado con ese correo electrónico")
	ErrClientHasActiveOrders = errors.New("no se puede eliminar el cliente porque tiene encargos activos (pendiente o en progreso)")
)

// ClientInput agrupa los datos de entrada para crear o editar un cliente.
// Los campos opcionales usan punteros para distinguir "no enviado" de "vacío".
type ClientInput struct {
	Nombre   string  // Obligatorio
	Telefono *string // Opcional
	Email    *string // Opcional — si se envía, se valida unicidad por usuario
}

// ClientService define el contrato del servicio de clientes.
type ClientService interface {
	Create(ctx context.Context, userID uuid.UUID, input ClientInput) (*domain.Client, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error)
	GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams) ([]domain.Client, int64, error)
	Update(ctx context.Context, id, userID uuid.UUID, input ClientInput) (*domain.Client, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	GetOrders(ctx context.Context, clientID, userID uuid.UUID) ([]domain.Order, error)
}

// clientService es la implementación concreta.
type clientService struct {
	clientRepo repository.ClientRepository
}

// NewClientService crea el servicio con inyección de dependencias.
func NewClientService(clientRepo repository.ClientRepository) ClientService {
	return &clientService{clientRepo: clientRepo}
}

// Create crea un nuevo cliente para la costurera.
// Reglas:
//   - Nombre: trim de espacios, mínimo 2 caracteres
//   - Email: toLower + trim, verificar unicidad por usuario
//   - Teléfono: trim de espacios (si se proporciona)
func (s *clientService) Create(ctx context.Context, userID uuid.UUID, input ClientInput) (*domain.Client, error) {
	// Sanitizar inputs
	input.Nombre = strings.TrimSpace(input.Nombre)
	if input.Telefono != nil {
		trimmed := strings.TrimSpace(*input.Telefono)
		input.Telefono = &trimmed
	}

	// Sanitizar y validar unicidad del email (si se proporciona)
	if input.Email != nil && *input.Email != "" {
		normalized := strings.ToLower(strings.TrimSpace(*input.Email))
		input.Email = &normalized

		if err := s.checkEmailUniqueness(ctx, *input.Email, userID, uuid.Nil); err != nil {
			return nil, err
		}
	} else {
		input.Email = nil // Tratar string vacío como ausente
	}

	client := &domain.Client{
		Nombre:   input.Nombre,
		Telefono: input.Telefono,
		Email:    input.Email,
		UserID:   userID,
	}

	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, fmt.Errorf("error creando cliente: %w", err)
	}

	return client, nil
}

// GetByID retorna un cliente por ID verificando que pertenezca al usuario.
func (s *clientService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Client, error) {
	client, err := s.clientRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("error obteniendo cliente: %w", err)
	}
	return client, nil
}

// GetAll retorna la lista paginada de clientes del usuario.
func (s *clientService) GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams) ([]domain.Client, int64, error) {
	return s.clientRepo.FindAll(ctx, userID, params)
}

// Update modifica los datos de un cliente existente.
// Verifica unicidad de email excluyendo el propio registro.
func (s *clientService) Update(ctx context.Context, id, userID uuid.UUID, input ClientInput) (*domain.Client, error) {
	// Verificar que el cliente existe y pertenece al usuario
	client, err := s.clientRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("error buscando cliente: %w", err)
	}

	// Sanitizar inputs
	client.Nombre = strings.TrimSpace(input.Nombre)
	if input.Telefono != nil {
		trimmed := strings.TrimSpace(*input.Telefono)
		client.Telefono = &trimmed
	} else {
		client.Telefono = nil
	}

	// Sanitizar y validar unicidad del email (excluyendo el propio registro)
	if input.Email != nil && *input.Email != "" {
		normalized := strings.ToLower(strings.TrimSpace(*input.Email))
		client.Email = &normalized
		// Pasar el ID del cliente actual para excluirlo de la verificación
		if err := s.checkEmailUniqueness(ctx, normalized, userID, client.ID); err != nil {
			return nil, err
		}
	} else {
		client.Email = nil
	}

	if err := s.clientRepo.Update(ctx, client); err != nil {
		return nil, fmt.Errorf("error actualizando cliente: %w", err)
	}

	return client, nil
}

// Delete elimina un cliente (soft delete) solo si no tiene encargos activos.
// Encargos activos = estado "pendiente" o "en_progreso".
func (s *clientService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	// Verificar que el cliente existe y pertenece al usuario
	_, err := s.clientRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrClientNotFound
		}
		return fmt.Errorf("error buscando cliente: %w", err)
	}

	// Bloquear eliminación si tiene encargos activos
	hasActive, err := s.clientRepo.HasActiveOrders(ctx, id)
	if err != nil {
		return fmt.Errorf("error verificando encargos: %w", err)
	}
	if hasActive {
		return ErrClientHasActiveOrders
	}

	if err := s.clientRepo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("error eliminando cliente: %w", err)
	}

	return nil
}

// GetOrders retorna el historial de encargos de un cliente.
func (s *clientService) GetOrders(ctx context.Context, clientID, userID uuid.UUID) ([]domain.Order, error) {
	// Verificar que el cliente existe y pertenece al usuario antes de retornar sus encargos
	_, err := s.clientRepo.FindByID(ctx, clientID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("error buscando cliente: %w", err)
	}

	orders, err := s.clientRepo.FindOrdersByClientID(ctx, clientID, userID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo encargos: %w", err)
	}

	return orders, nil
}

// checkEmailUniqueness verifica que el email no esté en uso por otro cliente del mismo usuario.
// excludeID = uuid.Nil en creación, = clientID en edición.
func (s *clientService) checkEmailUniqueness(ctx context.Context, email string, userID, excludeID uuid.UUID) error {
	existing, err := s.clientRepo.FindByEmail(ctx, email, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error verificando email: %w", err)
	}

	// Si encontró un cliente con ese email y NO es el que estamos editando → duplicado
	if existing != nil && existing.ID != excludeID {
		return ErrClientEmailDuplicate
	}

	return nil
}
