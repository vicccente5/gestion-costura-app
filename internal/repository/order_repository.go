// Package repository — interfaz e implementación del repositorio de encargos.
// Esta es la capa más compleja del proyecto porque los encargos orquestan:
//   1. Descuento de stock de materiales
//   2. Registro de order_materials con snapshot de costos
//   3. Creación automática de transacción al entregar
// Todas estas operaciones ocurren en transacciones SQL atómicas.
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// OrderRepository define el contrato de acceso a datos para encargos.
type OrderRepository interface {
	// CreateWithMaterials crea el encargo y descuenta stock en una transacción atómica.
	// Recibe el encargo + lista de materiales con sus cantidades.
	CreateWithMaterials(ctx context.Context, order *domain.Order, items []OrderItem) error

	// FindByID busca un encargo por ID y user_id (previene IDOR).
	// Precarga el cliente y los materiales del encargo.
	FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Order, error)

	// FindAll retorna la lista paginada con filtro de estado opcional.
	FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, estado string) ([]domain.Order, int64, error)

	// UpdateStatus cambia el estado del encargo.
	// Si el nuevo estado es "entregado", crea automáticamente una Transaction de ingreso.
	// Todo ocurre en una transacción atómica.
	UpdateStatus(ctx context.Context, order *domain.Order, newStatus domain.OrderStatus, transaction *domain.Transaction) error

	// UpdateMetadata actualiza los campos editables de un encargo (solo si está pendiente).
	UpdateMetadata(ctx context.Context, order *domain.Order) error

	// Delete hace soft delete de un encargo (solo si está pendiente).
	Delete(ctx context.Context, id, userID uuid.UUID) error

	// RestoreStock devuelve el stock de los materiales al cancelar un encargo.
	// Incluye los order_materials del encargo para saber qué restaurar.
	RestoreStock(ctx context.Context, order *domain.Order) error

	// AddMaterial añade un material a un encargo existente y descuenta su stock.
	AddMaterial(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, item OrderItem) (*domain.OrderMaterial, error)

	// EditMaterialQuantity ajusta la cantidad de un material en un encargo, restaurando la diferencia de stock.
	EditMaterialQuantity(ctx context.Context, orderMaterialID, userID uuid.UUID, nuevaCantidad float64) error

	// RemoveMaterial elimina un material de un encargo y restaura su stock.
	RemoveMaterial(ctx context.Context, orderMaterialID, userID uuid.UUID) error
}

// OrderItem agrupa un material y su cantidad para crear un encargo.
type OrderItem struct {
	Material *domain.Material // Necesario para el snapshot y para decrementar stock
	Cantidad float64
}

// orderRepository es la implementación concreta con GORM.
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository crea una nueva instancia del repositorio.
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// CreateWithMaterials crea el encargo + order_materials + descuenta stock en una sola TX.
// Si cualquier paso falla (ej: stock insuficiente a nivel de DB), todo se revierte.
func (r *orderRepository) CreateWithMaterials(ctx context.Context, order *domain.Order, items []OrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Crear el encargo
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("error creando encargo: %w", err)
		}

		// 2. Para cada material: crear OrderMaterial + decrementar stock
		for _, item := range items {
			orderMaterial := domain.OrderMaterial{
				Cantidad:             item.Cantidad,
				CostoUnitarioSnapshot: item.Material.CostoUnitario, // snapshot del costo actual
				OrderID:              order.ID,
				MaterialID:           item.Material.ID,
			}

			if err := tx.Create(&orderMaterial).Error; err != nil {
				return fmt.Errorf("error asignando material %s: %w", item.Material.Nombre, err)
			}

			// Decrementar stock del material — la DB tiene CHECK (stock_actual >= 0)
			// Si el stock resulta negativo, la DB lanza un error y se revierte todo
			result := tx.Model(&domain.Material{}).
				Where("id = ? AND user_id = ?", item.Material.ID, order.UserID).
				Update("stock_actual", gorm.Expr("stock_actual - ?", item.Cantidad))
			if result.Error != nil {
				return fmt.Errorf("error actualizando stock de %s: %w", item.Material.Nombre, result.Error)
			}
		}

		return nil
	})
}

// FindByID busca un encargo precargando cliente y materiales (con sus datos de material).
func (r *orderRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	result := r.db.WithContext(ctx).
		Preload("Client").
		Preload("Materials").
		Preload("Materials.Material").
		Where("orders.id = ? AND orders.user_id = ? AND orders.deleted_at IS NULL", id, userID).
		First(&order)
	if result.Error != nil {
		return nil, result.Error
	}
	return &order, nil
}

// FindAll retorna encargos paginados con filtro de estado opcional.
func (r *orderRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, estado string) ([]domain.Order, int64, error) {
	var orders []domain.Order
	var total int64

	query := r.db.WithContext(ctx).
		Model(&domain.Order{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	if estado != "" {
		query = query.Where("estado = ?", estado)
	}
	if params.Search != "" {
		query = query.Where("descripcion ILIKE ?", "%"+params.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("Client").
		Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateStatus cambia el estado del encargo.
// Elimina transacciones previas asociadas (para evitar duplicados) y si newStatus == "entregado", crea la nueva.
func (r *orderRepository) UpdateStatus(ctx context.Context, order *domain.Order, newStatus domain.OrderStatus, transaction *domain.Transaction) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Actualizar estado del encargo
		if err := tx.Model(order).Update("estado", string(newStatus)).Error; err != nil {
			return fmt.Errorf("error actualizando estado: %w", err)
		}

		// Eliminar cualquier transacción previa asociada a este encargo
		// Esto evita duplicados si el encargo pasa a 'entregado' múltiples veces,
		// o elimina el ingreso fantasma si pasa de 'entregado' a 'pendiente'.
		if err := tx.Where("order_id = ?", order.ID).Delete(&domain.Transaction{}).Error; err != nil {
			return fmt.Errorf("error limpiando transacciones previas: %w", err)
		}

		// Si se entrega → crear la transacción de ingreso automática
		if newStatus == domain.OrderStatusEntregado && transaction != nil {
			if err := tx.Create(transaction).Error; err != nil {
				return fmt.Errorf("error creando transacción de ingreso: %w", err)
			}
		}

		return nil
	})
}

// UpdateMetadata actualiza los campos editables del encargo.
func (r *orderRepository) UpdateMetadata(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}

// Delete hace soft delete verificando que pertenezca al usuario, e incluye transacciones.
func (r *orderRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Soft delete al encargo
		result := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&domain.Order{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// Soft delete a las transacciones asociadas (evita ingresos fantasma)
		if err := tx.Where("order_id = ?", id).Delete(&domain.Transaction{}).Error; err != nil {
			return fmt.Errorf("error eliminando transacciones asociadas: %w", err)
		}

		return nil
	})
}

// RestoreStock devuelve el stock de los materiales al cancelar un encargo.
func (r *orderRepository) RestoreStock(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range order.Materials {
			result := tx.Model(&domain.Material{}).
				Where("id = ?", item.MaterialID).
				Update("stock_actual", gorm.Expr("stock_actual + ?", item.Cantidad))
			if result.Error != nil {
				return fmt.Errorf("error restaurando stock: %w", result.Error)
			}
		}
		return nil
	})
}

// AddMaterial añade un material a un encargo existente y descuenta su stock atómicamente.
func (r *orderRepository) AddMaterial(ctx context.Context, orderID uuid.UUID, userID uuid.UUID, item OrderItem) (*domain.OrderMaterial, error) {
	var orderMaterial domain.OrderMaterial
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Verificar que el encargo existe y pertenece al usuario
		var count int64
		if err := tx.Model(&domain.Order{}).Where("id = ? AND user_id = ? AND deleted_at IS NULL", orderID, userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}

		// 2. Crear OrderMaterial
		orderMaterial = domain.OrderMaterial{
			Cantidad:             item.Cantidad,
			CostoUnitarioSnapshot: item.Material.CostoUnitario,
			OrderID:              orderID,
			MaterialID:           item.Material.ID,
		}
		if err := tx.Create(&orderMaterial).Error; err != nil {
			return fmt.Errorf("error guardando material del encargo: %w", err)
		}

		// 3. Descontar stock
		result := tx.Model(&domain.Material{}).
			Where("id = ? AND user_id = ?", item.Material.ID, userID).
			Update("stock_actual", gorm.Expr("stock_actual - ?", item.Cantidad))
		if result.Error != nil {
			return fmt.Errorf("error descontando stock: %w", result.Error)
		}

		return nil
	})
	return &orderMaterial, err
}

// EditMaterialQuantity ajusta la cantidad de un material y el stock atómicamente.
func (r *orderRepository) EditMaterialQuantity(ctx context.Context, orderMaterialID, userID uuid.UUID, nuevaCantidad float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var om domain.OrderMaterial
		// JOIN con orders para validar pertenencia (user_id)
		if err := tx.Joins("JOIN orders ON orders.id = order_materials.order_id").
			Where("order_materials.id = ? AND orders.user_id = ? AND orders.deleted_at IS NULL", orderMaterialID, userID).
			First(&om).Error; err != nil {
			return err
		}

		diferencia := nuevaCantidad - om.Cantidad

		// Actualizar cantidad en el encargo
		if err := tx.Model(&om).Update("cantidad", nuevaCantidad).Error; err != nil {
			return err
		}

		// Actualizar stock (diferencia positiva = resta stock, negativa = suma stock)
		result := tx.Model(&domain.Material{}).
			Where("id = ?", om.MaterialID).
			Update("stock_actual", gorm.Expr("stock_actual - ?", diferencia))
		
		return result.Error
	})
}

// RemoveMaterial elimina el material del encargo y restaura su stock.
func (r *orderRepository) RemoveMaterial(ctx context.Context, orderMaterialID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var om domain.OrderMaterial
		if err := tx.Joins("JOIN orders ON orders.id = order_materials.order_id").
			Where("order_materials.id = ? AND orders.user_id = ? AND orders.deleted_at IS NULL", orderMaterialID, userID).
			First(&om).Error; err != nil {
			return err
		}

		// Restaurar stock
		if err := tx.Model(&domain.Material{}).
			Where("id = ?", om.MaterialID).
			Update("stock_actual", gorm.Expr("stock_actual + ?", om.Cantidad)).Error; err != nil {
			return err
		}

		// Eliminar OrderMaterial (Hard delete porque es tabla pivote sin deleted_at)
		return tx.Delete(&om).Error
	})
}
