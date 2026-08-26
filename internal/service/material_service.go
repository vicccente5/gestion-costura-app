// Package service — lógica de negocio del módulo de inventario de materiales.
//
// La pieza central de este módulo es el cálculo del PROMEDIO PONDERADO MÓVIL:
// Cuando se registra una compra, el nuevo costo unitario se calcula considerando
// tanto el stock existente como el stock nuevo, no simplemente el último precio pagado.
//
// Ejemplo:
//   - Tengo 10 metros de tela a $500/metro (stock existente)
//   - Compro 5 metros más a $600/metro (nueva compra)
//   - Costo promedio = (10*500 + 5*600) / (10+5) = (5000+3000)/15 = $533/metro
//   - Si usara solo el último precio ($600), sobreestimaría el costo real del stock
package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vicccente5/gestion-costura-app/internal/domain"
	"github.com/vicccente5/gestion-costura-app/internal/repository"
	"github.com/vicccente5/gestion-costura-app/internal/utils"
	"gorm.io/gorm"
)

// Errores tipados del servicio de materiales.
var (
	ErrMaterialNotFound       = errors.New("material no encontrado")
	ErrMaterialNameDuplicate  = errors.New("ya tienes un material con ese nombre")
	ErrMaterialUsedInOrders   = errors.New("no se puede eliminar el material porque está en uso en encargos")
	ErrPurchaseQuantityInvalid = errors.New("la cantidad de la compra debe ser mayor a 0")
	ErrPurchasePriceInvalid    = errors.New("el precio unitario debe ser mayor a 0")
)

// MaterialInput agrupa los datos para crear o editar un material.
type MaterialInput struct {
	Nombre        string
	Categoria     string
	Unidad        string
	StockMinimo   float64
	CostoUnitario int64 // Precio inicial estimado (se actualiza con compras)
}

// PurchaseInput agrupa los datos para registrar una compra.
type PurchaseInput struct {
	Cantidad       float64
	PrecioUnitario int64
	Fecha          time.Time
	Notas          *string
}

// MaterialService define el contrato del servicio de inventario.
type MaterialService interface {
	Create(ctx context.Context, userID uuid.UUID, input MaterialInput) (*domain.Material, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Material, error)
	GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, categoria string) ([]domain.Material, int64, error)
	GetLowStock(ctx context.Context, userID uuid.UUID) ([]domain.Material, error)
	Update(ctx context.Context, id, userID uuid.UUID, input MaterialInput) (*domain.Material, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	RegisterPurchase(ctx context.Context, materialID, userID uuid.UUID, input PurchaseInput) (*domain.MaterialPurchase, *domain.Material, error)
	GetPurchases(ctx context.Context, materialID, userID uuid.UUID) ([]domain.MaterialPurchase, error)
}

// materialService es la implementación concreta.
type materialService struct {
	materialRepo repository.MaterialRepository
}

// NewMaterialService crea el service con inyección de dependencias.
func NewMaterialService(materialRepo repository.MaterialRepository) MaterialService {
	return &materialService{materialRepo: materialRepo}
}

// Create crea un nuevo material verificando unicidad de nombre por usuario.
func (s *materialService) Create(ctx context.Context, userID uuid.UUID, input MaterialInput) (*domain.Material, error) {
	// Sanitizar inputs
	input.Nombre = strings.TrimSpace(input.Nombre)
	input.Categoria = strings.TrimSpace(input.Categoria)
	input.Unidad = strings.TrimSpace(input.Unidad)

	// Verificar unicidad de nombre por usuario (case-insensitive)
	if err := s.checkNameUniqueness(ctx, input.Nombre, userID, uuid.Nil); err != nil {
		return nil, err
	}

	material := &domain.Material{
		Nombre:        input.Nombre,
		Categoria:     input.Categoria,
		Unidad:        input.Unidad,
		StockActual:   0,                  // El stock inicial siempre es 0 — se alimenta con compras
		StockMinimo:   input.StockMinimo,
		CostoUnitario: input.CostoUnitario, // Estimación inicial — se corrige con la primera compra
		UserID:        userID,
	}

	if err := s.materialRepo.Create(ctx, material); err != nil {
		return nil, fmt.Errorf("error creando material: %w", err)
	}

	return material, nil
}

func (s *materialService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Material, error) {
	material, err := s.materialRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, fmt.Errorf("error obteniendo material: %w", err)
	}
	return material, nil
}

func (s *materialService) GetAll(ctx context.Context, userID uuid.UUID, params utils.PaginationParams, categoria string) ([]domain.Material, int64, error) {
	return s.materialRepo.FindAll(ctx, userID, params, categoria)
}

func (s *materialService) GetLowStock(ctx context.Context, userID uuid.UUID) ([]domain.Material, error) {
	return s.materialRepo.FindLowStock(ctx, userID)
}

// Update modifica los metadatos del material (nombre, categoría, unidad, stock mínimo).
// NO permite modificar stock_actual ni costo_unitario directamente — eso se hace con compras.
func (s *materialService) Update(ctx context.Context, id, userID uuid.UUID, input MaterialInput) (*domain.Material, error) {
	material, err := s.materialRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, fmt.Errorf("error buscando material: %w", err)
	}

	// Sanitizar
	input.Nombre = strings.TrimSpace(input.Nombre)
	input.Categoria = strings.TrimSpace(input.Categoria)
	input.Unidad = strings.TrimSpace(input.Unidad)

	// Verificar unicidad de nombre excluyendo el propio registro
	if !strings.EqualFold(material.Nombre, input.Nombre) {
		if err := s.checkNameUniqueness(ctx, input.Nombre, userID, material.ID); err != nil {
			return nil, err
		}
	}

	material.Nombre = input.Nombre
	material.Categoria = input.Categoria
	material.Unidad = input.Unidad
	material.StockMinimo = input.StockMinimo
	// stock_actual y costo_unitario NO se modifican aquí

	if err := s.materialRepo.Update(ctx, material); err != nil {
		return nil, fmt.Errorf("error actualizando material: %w", err)
	}

	return material, nil
}

// Delete elimina un material si no está siendo usado en ningún encargo.
func (s *materialService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.materialRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMaterialNotFound
		}
		return fmt.Errorf("error buscando material: %w", err)
	}

	isUsed, err := s.materialRepo.IsUsedInOrders(ctx, id)
	if err != nil {
		return fmt.Errorf("error verificando uso del material: %w", err)
	}
	if isUsed {
		return ErrMaterialUsedInOrders
	}

	return s.materialRepo.Delete(ctx, id, userID)
}

// RegisterPurchase registra una compra y actualiza el inventario.
//
// Dos operaciones en una transacción atómica:
//  1. Crear el registro en material_purchases
//  2. Actualizar stock_actual y recalcular costo_unitario con promedio ponderado
//
// Fórmula del promedio ponderado móvil:
//
//	nuevo_costo = (stock_actual * costo_anterior + cantidad * precio_unitario) / (stock_actual + cantidad)
//
// Se usa math.Round para convertir el float a int64 (CLP son enteros).
func (s *materialService) RegisterPurchase(ctx context.Context, materialID, userID uuid.UUID, input PurchaseInput) (*domain.MaterialPurchase, *domain.Material, error) {
	// Validaciones de negocio
	if input.Cantidad <= 0 {
		return nil, nil, ErrPurchaseQuantityInvalid
	}
	if input.PrecioUnitario <= 0 {
		return nil, nil, ErrPurchasePriceInvalid
	}

	material, err := s.materialRepo.FindByID(ctx, materialID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrMaterialNotFound
		}
		return nil, nil, fmt.Errorf("error buscando material: %w", err)
	}

	// Calcular el nuevo costo unitario con promedio ponderado móvil
	stockAnterior := material.StockActual
	costoAnterior := float64(material.CostoUnitario)
	stockNuevo := stockAnterior + input.Cantidad

	var nuevoCosto int64
	if stockNuevo > 0 {
		promedioPonderado := (stockAnterior*costoAnterior + input.Cantidad*float64(input.PrecioUnitario)) / stockNuevo
		nuevoCosto = int64(math.Round(promedioPonderado))
	}

	// Calcular el precio total de la compra
	precioTotal := int64(math.Round(input.Cantidad * float64(input.PrecioUnitario)))

	// Preparar el registro de compra
	purchase := &domain.MaterialPurchase{
		Cantidad:       input.Cantidad,
		PrecioUnitario: input.PrecioUnitario,
		PrecioTotal:    precioTotal,
		Fecha:          input.Fecha,
		Notas:          input.Notas,
		MaterialID:     materialID,
	}

	// Actualizar el material con los nuevos valores calculados
	material.StockActual = stockNuevo
	material.CostoUnitario = nuevoCosto

	// Persistir ambas operaciones en una transacción atómica
	if err := s.materialRepo.CreatePurchase(ctx, purchase, material); err != nil {
		return nil, nil, fmt.Errorf("error registrando compra: %w", err)
	}

	return purchase, material, nil
}

func (s *materialService) GetPurchases(ctx context.Context, materialID, userID uuid.UUID) ([]domain.MaterialPurchase, error) {
	// Verificar que el material existe y pertenece al usuario
	if _, err := s.materialRepo.FindByID(ctx, materialID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, fmt.Errorf("error buscando material: %w", err)
	}

	return s.materialRepo.FindPurchasesByMaterialID(ctx, materialID, userID)
}

// checkNameUniqueness verifica que el nombre no esté en uso por otro material del mismo usuario.
// excludeID = uuid.Nil en creación, = materialID en edición.
func (s *materialService) checkNameUniqueness(ctx context.Context, nombre string, userID, excludeID uuid.UUID) error {
	existing, err := s.materialRepo.FindByName(ctx, nombre, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error verificando nombre: %w", err)
	}
	if existing != nil && existing.ID != excludeID {
		return ErrMaterialNameDuplicate
	}
	return nil
}
