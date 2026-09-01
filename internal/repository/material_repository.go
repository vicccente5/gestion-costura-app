// Package repository — interfaz e implementación del repositorio de materiales.
// Incluye materiales y su historial de compras (material_purchases).
//
// REGLA DE ORO: todos los métodos que devuelven datos filtran por user_id.
// Esto garantiza que las costureras solo ven sus propios materiales.
package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// MaterialRepository define el contrato de acceso a datos para materiales.
type MaterialRepository interface {
	// Create inserta un nuevo material en la DB.
	Create(ctx context.Context, material *domain.Material) error

	// FindByID busca un material por ID y user_id (previene IDOR).
	FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Material, error)

	// FindByName busca un material por nombre dentro del mismo usuario.
	// Usado para verificar unicidad antes de crear o editar.
	FindByName(ctx context.Context, nombre string, userID uuid.UUID) (*domain.Material, error)

	// FindAll retorna la lista paginada de materiales con búsqueda y filtro por categoría.
	FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, categoria string) ([]domain.Material, int64, error)

	// FindLowStock retorna materiales donde stock_actual <= stock_minimo.
	FindLowStock(ctx context.Context, userID uuid.UUID) ([]domain.Material, error)

	// Update actualiza los campos de un material.
	Update(ctx context.Context, material *domain.Material) error

	// Delete hace soft delete de un material.
	Delete(ctx context.Context, id, userID uuid.UUID) error

	// IsUsedInOrders verifica si el material tiene registros en order_materials.
	// Usado para bloquear eliminaciones que romperían el historial de encargos.
	IsUsedInOrders(ctx context.Context, materialID uuid.UUID) (bool, error)

	// CreatePurchase registra la compra, actualiza el material y opcionalmente crea una transacción financiera.
	CreatePurchase(ctx context.Context, purchase *domain.MaterialPurchase, material *domain.Material, transaction *domain.Transaction) error

	// FindPurchasesByMaterialID retorna el historial de compras de un material.
	FindPurchasesByMaterialID(ctx context.Context, materialID, userID uuid.UUID) ([]domain.MaterialPurchase, error)
}

// materialRepository es la implementación concreta con GORM.
type materialRepository struct {
	db *gorm.DB
}

// NewMaterialRepository crea una nueva instancia del repositorio.
func NewMaterialRepository(db *gorm.DB) MaterialRepository {
	return &materialRepository{db: db}
}

func (r *materialRepository) Create(ctx context.Context, material *domain.Material) error {
	return r.db.WithContext(ctx).Create(material).Error
}

func (r *materialRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Material, error) {
	var material domain.Material
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		First(&material)
	if result.Error != nil {
		return nil, result.Error
	}
	return &material, nil
}

func (r *materialRepository) FindByName(ctx context.Context, nombre string, userID uuid.UUID) (*domain.Material, error) {
	var material domain.Material
	// ILIKE para búsqueda case-insensitive (el nombre podría venir con distintas capitalizaciones)
	result := r.db.WithContext(ctx).
		Where("LOWER(nombre) = LOWER(?) AND user_id = ? AND deleted_at IS NULL", nombre, userID).
		First(&material)
	if result.Error != nil {
		return nil, result.Error
	}
	return &material, nil
}

func (r *materialRepository) FindAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, categoria string) ([]domain.Material, int64, error) {
	var materials []domain.Material
	var total int64

	query := r.db.WithContext(ctx).
		Model(&domain.Material{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	if params.Search != "" {
		query = query.Where("nombre ILIKE ?", "%"+params.Search+"%")
	}
	if categoria != "" {
		query = query.Where("categoria ILIKE ?", "%"+categoria+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("nombre ASC").
		Limit(params.Limit).
		Offset(params.Offset).
		Find(&materials).Error; err != nil {
		return nil, 0, err
	}

	return materials, total, nil
}

func (r *materialRepository) FindLowStock(ctx context.Context, userID uuid.UUID) ([]domain.Material, error) {
	var materials []domain.Material
	// Solo materiales donde el stock actual cayó por debajo o igualó al mínimo
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL AND stock_actual <= stock_minimo", userID).
		Order("nombre ASC").
		Find(&materials).Error
	return materials, err
}

func (r *materialRepository) Update(ctx context.Context, material *domain.Material) error {
	return r.db.WithContext(ctx).Save(material).Error
}

func (r *materialRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&domain.Material{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *materialRepository) IsUsedInOrders(ctx context.Context, materialID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.OrderMaterial{}).
		Where("material_id = ?", materialID).
		Count(&count).Error
	return count > 0, err
}

// CreatePurchase registra la compra Y actualiza el material en una transacción atómica.
// Si cualquier parte falla, todo se revierte — stock y compra siempre son consistentes.
func (r *materialRepository) CreatePurchase(ctx context.Context, purchase *domain.MaterialPurchase, material *domain.Material, transaction *domain.Transaction) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Insertar el registro de compra
		if err := tx.Create(purchase).Error; err != nil {
			return fmt.Errorf("error guardando registro de compra: %w", err)
		}

		// 2. Actualizar stock y costo del material (solo campos específicos para no sobreescribir otros datos sin querer)
		if err := tx.Model(material).Updates(map[string]interface{}{
			"stock_actual":   material.StockActual,
			"costo_unitario": material.CostoUnitario,
		}).Error; err != nil {
			return fmt.Errorf("error actualizando stock del material: %w", err)
		}

		// 3. Crear la transacción financiera (si aplica)
		if transaction != nil {
			if err := tx.Create(transaction).Error; err != nil {
				return fmt.Errorf("error creando transacción financiera: %w", err)
			}
		}

		return nil
	})
}

func (r *materialRepository) FindPurchasesByMaterialID(ctx context.Context, materialID, userID uuid.UUID) ([]domain.MaterialPurchase, error) {
	var purchases []domain.MaterialPurchase
	// Verificar user_id mediante JOIN para prevenir IDOR
	err := r.db.WithContext(ctx).
		Joins("JOIN materials ON materials.id = material_purchases.material_id").
		Where("material_purchases.material_id = ? AND materials.user_id = ? AND materials.deleted_at IS NULL",
			materialID, userID).
		Order("material_purchases.created_at DESC").
		Find(&purchases).Error
	return purchases, err
}
