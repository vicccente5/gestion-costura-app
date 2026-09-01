// Package service — lógica de negocio de gestión de encargos.
//
// Máquina de estados de un encargo:
//
//	pendiente ──→ en_progreso ──→ completado ──→ entregado (genera Transaction automática)
//	    │               │               │
//	    └───────────────┴───────────────┴──→ cancelado (restaura stock si aplica)
//
// Reglas de negocio clave:
//  1. Al crear: se valida stock suficiente por material, se descuenta stock, se guarda snapshot de costos
//  2. Solo se puede editar metadata si el encargo está en "pendiente"
//  3. Al cancelar desde pendiente/en_progreso: se restaura el stock de materiales
//  4. Al entregar: se crea automáticamente una Transaction de ingreso (source="order")
//  5. Margen de ganancia: null si precio_venta=0 (no dividir por cero)
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// Errores tipados del servicio de encargos.
var (
	ErrOrderNotFound             = errors.New("encargo no encontrado")
	ErrOrderNotEditable          = errors.New("solo se pueden editar encargos en estado pendiente")
	ErrOrderNotDeletable         = errors.New("solo se pueden eliminar encargos en estado pendiente")
	ErrOrderInvalidStatusChange  = errors.New("cambio de estado no permitido por la máquina de estados")
	ErrOrderAlreadyDelivered     = errors.New("el encargo ya fue entregado")
	ErrInsufficientStock         = errors.New("stock insuficiente para uno o más materiales")
	ErrOrderClientNotFound       = errors.New("el cliente del encargo no pertenece a este usuario")
	ErrOrderMaterialNotFound     = errors.New("uno o más materiales no pertenecen a este usuario")
	ErrOrderNoPrice              = errors.New("no se puede entregar un encargo con precio de venta en 0")
)

// OrderMaterialInput define un material y su cantidad en la creación del encargo.
type OrderMaterialInput struct {
	MaterialID uuid.UUID
	Cantidad   float64
}

// OrderCreateInput agrupa los datos para crear un encargo.
type OrderCreateInput struct {
	Descripcion  string
	PrecioVenta  int64
	Horas        float64
	TarifaHora   int64
	FechaEntrega *time.Time
	Notas        *string
	ClientID     uuid.UUID
	Materials    []OrderMaterialInput
}

// OrderUpdateInput permite editar metadata de un encargo pendiente.
type OrderUpdateInput struct {
	Descripcion  string
	PrecioVenta  int64
	Horas        float64
	TarifaHora   int64
	FechaEntrega *time.Time
	Notas        *string
}

// OrderService define el contrato del servicio de encargos.
type OrderService interface {
	Create(ctx context.Context, userID uuid.UUID, input OrderCreateInput) (*domain.Order, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Order, error)
	GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, estado string) ([]domain.Order, int64, error)
	UpdateMetadata(ctx context.Context, id, userID uuid.UUID, input OrderUpdateInput) (*domain.Order, error)
	ChangeStatus(ctx context.Context, id, userID uuid.UUID, newStatus domain.OrderStatus) (*domain.Order, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error

	// Gestión de materiales del encargo
	AddMaterial(ctx context.Context, orderID, userID uuid.UUID, input OrderMaterialInput) (*domain.OrderMaterial, error)
	EditMaterialQuantity(ctx context.Context, orderID, orderMaterialID, userID uuid.UUID, nuevaCantidad float64) error
	RemoveMaterial(ctx context.Context, orderID, orderMaterialID, userID uuid.UUID) error
}

// orderService es la implementación concreta.
type orderService struct {
	orderRepo    repository.OrderRepository
	clientRepo   repository.ClientRepository
	materialRepo repository.MaterialRepository
}

// NewOrderService crea el service con inyección de dependencias.
// Necesita acceso a client y material repos para validaciones.
func NewOrderService(
	orderRepo repository.OrderRepository,
	clientRepo repository.ClientRepository,
	materialRepo repository.MaterialRepository,
) OrderService {
	return &orderService{
		orderRepo:    orderRepo,
		clientRepo:   clientRepo,
		materialRepo: materialRepo,
	}
}

// Create crea un encargo validando stock y guardando snapshots de costos.
func (s *orderService) Create(ctx context.Context, userID uuid.UUID, input OrderCreateInput) (*domain.Order, error) {
	// 1. Verificar que el cliente pertenece al usuario
	_, err := s.clientRepo.FindByID(ctx, input.ClientID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderClientNotFound
		}
		return nil, fmt.Errorf("error verificando cliente: %w", err)
	}

	// 2. Cargar materiales, validar pertenencia y stock suficiente
	orderItems, err := s.buildOrderItems(ctx, userID, input.Materials)
	if err != nil {
		return nil, err
	}

	// 3. Construir el encargo
	order := &domain.Order{
		Descripcion:  input.Descripcion,
		PrecioVenta:  input.PrecioVenta,
		Horas:        input.Horas,
		TarifaHora:   input.TarifaHora,
		FechaEntrega: input.FechaEntrega,
		Notas:        input.Notas,
		ClientID:     input.ClientID,
		UserID:       userID,
		Estado:       domain.OrderStatusPendiente,
	}

	// 4. Persistir en transacción atómica (crea encargo + order_materials + descuenta stock)
	if err := s.orderRepo.CreateWithMaterials(ctx, order, orderItems); err != nil {
		return nil, fmt.Errorf("error creando encargo: %w", err)
	}

	return order, nil
}

func (s *orderService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("error obteniendo encargo: %w", err)
	}
	return order, nil
}

func (s *orderService) GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, estado string) ([]domain.Order, int64, error) {
	return s.orderRepo.FindAll(ctx, userID, params, estado)
}

// UpdateMetadata edita los campos de un encargo — solo si está en estado "pendiente".
func (s *orderService) UpdateMetadata(ctx context.Context, id, userID uuid.UUID, input OrderUpdateInput) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("error buscando encargo: %w", err)
	}

	if order.Estado != domain.OrderStatusPendiente {
		return nil, ErrOrderNotEditable
	}

	order.Descripcion = input.Descripcion
	order.PrecioVenta = input.PrecioVenta
	order.Horas = input.Horas
	order.TarifaHora = input.TarifaHora
	order.FechaEntrega = input.FechaEntrega
	order.Notas = input.Notas

	if err := s.orderRepo.UpdateMetadata(ctx, order); err != nil {
		return nil, fmt.Errorf("error actualizando encargo: %w", err)
	}

	return order, nil
}

// ChangeStatus aplica la máquina de estados y maneja los efectos secundarios.
//
// Transiciones válidas:
//
//	pendiente   → en_progreso, cancelado
//	en_progreso → completado, cancelado
//	completado  → entregado, cancelado
//	entregado   → (estado terminal — no se puede cambiar)
//	cancelado   → (estado terminal — no se puede cambiar)
func (s *orderService) ChangeStatus(ctx context.Context, id, userID uuid.UUID, newStatus domain.OrderStatus) (*domain.Order, error) {
	order, err := s.orderRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("error buscando encargo: %w", err)
	}

	// Validar la transición de estado — se permite saltar a cualquier estado
	// principal (pendiente, en_progreso, completado, entregado).
	// Solo restricción: si ya está cancelado, no se puede cambiar.
	if order.Estado == domain.OrderStatusCancelado {
		return nil, fmt.Errorf("%w: un encargo cancelado no puede cambiar de estado", ErrOrderInvalidStatusChange)
	}
	// Validar que el nuevo estado sea uno de los valores aceptados
	validStates := map[domain.OrderStatus]bool{
		domain.OrderStatusPendiente:  true,
		domain.OrderStatusEnProgreso: true,
		domain.OrderStatusCompletado: true,
		domain.OrderStatusEntregado:  true,
		domain.OrderStatusCancelado:  true,
	}
	if !validStates[newStatus] {
		return nil, fmt.Errorf("%w: estado inválido '%s'", ErrOrderInvalidStatusChange, newStatus)
	}

	// Efecto secundario: cancelado → restaurar stock
	if newStatus == domain.OrderStatusCancelado {
		// Solo restaurar stock si el encargo tenía materiales asignados
		// y estaba en un estado donde el stock ya fue descontado
		if len(order.Materials) > 0 {
			if err := s.orderRepo.RestoreStock(ctx, order); err != nil {
				return nil, fmt.Errorf("error restaurando stock: %w", err)
			}
		}
	}

	// Efecto secundario: entregado → crear Transaction de ingreso automática
	var autoTransaction *domain.Transaction
	if newStatus == domain.OrderStatusEntregado {
		if order.PrecioVenta <= 0 {
			return nil, ErrOrderNoPrice
		}
		autoTransaction = s.buildDeliveryTransaction(order)
	}

	// Persistir cambio de estado (+ transaction si aplica)
	if err := s.orderRepo.UpdateStatus(ctx, order, newStatus, autoTransaction); err != nil {
		return nil, fmt.Errorf("error actualizando estado: %w", err)
	}

	order.Estado = newStatus
	return order, nil
}

// Delete elimina un encargo — solo si está en estado "pendiente".
func (s *orderService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	order, err := s.orderRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("error buscando encargo: %w", err)
	}

	// Ya no restringimos borrar solo encargos pendientes.
	// Se puede borrar en cualquier estado, y siempre restauramos stock.
	// Restaurar stock antes de eliminar
	if len(order.Materials) > 0 {
		if err := s.orderRepo.RestoreStock(ctx, order); err != nil {
			return fmt.Errorf("error restaurando stock: %w", err)
		}
	}

	return s.orderRepo.Delete(ctx, id, userID)
}

// ──────────────────────────────────────────────
// Helpers privados
// ──────────────────────────────────────────────

// buildOrderItems valida y carga los materiales del encargo.
// Verifica: pertenencia al usuario, stock suficiente.
func (s *orderService) buildOrderItems(ctx context.Context, userID uuid.UUID, inputs []OrderMaterialInput) ([]repository.OrderItem, error) {
	items := make([]repository.OrderItem, 0, len(inputs))

	for _, input := range inputs {
		if input.Cantidad <= 0 {
			return nil, fmt.Errorf("la cantidad del material debe ser mayor a 0")
		}

		material, err := s.materialRepo.FindByID(ctx, input.MaterialID, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrOrderMaterialNotFound
			}
			return nil, fmt.Errorf("error verificando material: %w", err)
		}

		// Verificar stock suficiente en el service (antes de llegar a la DB)
		if material.StockActual < input.Cantidad {
			return nil, fmt.Errorf("%w: %s (disponible: %.2f, requerido: %.2f)",
				ErrInsufficientStock, material.Nombre, material.StockActual, input.Cantidad)
		}

		items = append(items, repository.OrderItem{
			Material: material,
			Cantidad: input.Cantidad,
		})
	}

	return items, nil
}

// validateStatusTransition verifica que la transición de estado sea válida.
func validateStatusTransition(current, next domain.OrderStatus) error {
	// Estados terminales — no se puede cambiar desde aquí
	if current == domain.OrderStatusEntregado || current == domain.OrderStatusCancelado {
		if current == domain.OrderStatusEntregado {
			return ErrOrderAlreadyDelivered
		}
		return ErrOrderInvalidStatusChange
	}

	// Matriz de transiciones válidas
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPendiente:  {domain.OrderStatusEnProgreso, domain.OrderStatusCancelado},
		domain.OrderStatusEnProgreso: {domain.OrderStatusCompletado, domain.OrderStatusCancelado},
		domain.OrderStatusCompletado: {domain.OrderStatusEntregado, domain.OrderStatusCancelado},
	}

	for _, valid := range validTransitions[current] {
		if next == valid {
			return nil
		}
	}

	return fmt.Errorf("%w: no se puede pasar de %s a %s", ErrOrderInvalidStatusChange, current, next)
}

// buildDeliveryTransaction construye la Transaction automática al entregar un encargo.
// Refleja la ganancia normal (precio de venta) y la ganancia neta.
func (s *orderService) buildDeliveryTransaction(order *domain.Order) *domain.Transaction {
	// Costo total de materiales con los snapshots históricos
	var costoMateriales float64
	for _, item := range order.Materials {
		costoMateriales += item.Cantidad * float64(item.CostoUnitarioSnapshot)
	}

	// Costo de mano de obra
	costoManoObra := float64(order.Horas) * float64(order.TarifaHora)

	costoTotal := costoMateriales + costoManoObra
	gananciaNeta := float64(order.PrecioVenta) - costoTotal

	descripcion := fmt.Sprintf("Encargo entregado: %s | Ganancia Normal (Ingreso): $%d | Ganancia Neta: $%.0f", 
		order.Descripcion, order.PrecioVenta, gananciaNeta)

	orderID := order.ID
	return &domain.Transaction{
		Tipo:        domain.TransactionTypeIngreso,
		Monto:       order.PrecioVenta,
		Descripcion: descripcion,
		Fecha:       time.Now(),
		Source:      domain.TransactionSourceOrder,
		OrderID:     &orderID,
		UserID:      order.UserID,
	}
}

// AddMaterial añade un material a un encargo si está en estado pendiente.
func (s *orderService) AddMaterial(ctx context.Context, orderID, userID uuid.UUID, input OrderMaterialInput) (*domain.OrderMaterial, error) {
	order, err := s.orderRepo.FindByID(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("error buscando encargo: %w", err)
	}

	if order.Estado != domain.OrderStatusPendiente {
		return nil, ErrOrderNotEditable
	}

	if input.Cantidad <= 0 {
		return nil, fmt.Errorf("la cantidad debe ser mayor a 0")
	}

	material, err := s.materialRepo.FindByID(ctx, input.MaterialID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderMaterialNotFound
		}
		return nil, fmt.Errorf("error buscando material: %w", err)
	}

	if material.StockActual < input.Cantidad {
		return nil, fmt.Errorf("%w: %s (disponible: %.2f, requerido: %.2f)",
			ErrInsufficientStock, material.Nombre, material.StockActual, input.Cantidad)
	}

	item := repository.OrderItem{
		Material: material,
		Cantidad: input.Cantidad,
	}

	return s.orderRepo.AddMaterial(ctx, orderID, userID, item)
}

// EditMaterialQuantity edita la cantidad de un material en un encargo pendiente.
func (s *orderService) EditMaterialQuantity(ctx context.Context, orderID, orderMaterialID, userID uuid.UUID, nuevaCantidad float64) error {
	order, err := s.orderRepo.FindByID(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("error buscando encargo: %w", err)
	}

	if order.Estado != domain.OrderStatusPendiente {
		return ErrOrderNotEditable
	}

	if nuevaCantidad <= 0 {
		return fmt.Errorf("la cantidad debe ser mayor a 0")
	}

	// Buscar el material específico en el encargo
	var om *domain.OrderMaterial
	for _, m := range order.Materials {
		if m.ID == orderMaterialID {
			om = &m
			break
		}
	}
	if om == nil {
		return fmt.Errorf("el material no pertenece a este encargo")
	}

	// Verificar si hay stock suficiente si la cantidad aumenta
	if nuevaCantidad > om.Cantidad {
		diferencia := nuevaCantidad - om.Cantidad
		if om.Material.StockActual < diferencia {
			return fmt.Errorf("%w: %s (disponible: %.2f, requerido extra: %.2f)",
				ErrInsufficientStock, om.Material.Nombre, om.Material.StockActual, diferencia)
		}
	}

	return s.orderRepo.EditMaterialQuantity(ctx, orderMaterialID, userID, nuevaCantidad)
}

// RemoveMaterial quita un material de un encargo pendiente.
func (s *orderService) RemoveMaterial(ctx context.Context, orderID, orderMaterialID, userID uuid.UUID) error {
	order, err := s.orderRepo.FindByID(ctx, orderID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("error buscando encargo: %w", err)
	}

	if order.Estado != domain.OrderStatusPendiente {
		return ErrOrderNotEditable
	}

	return s.orderRepo.RemoveMaterial(ctx, orderMaterialID, userID)
}
