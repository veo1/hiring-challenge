package models

import (
	"gorm.io/gorm"
)

type ProductsRepository struct {
	db *gorm.DB
}

type ProductFilters struct {
	CategoryCode  string
	PriceLessThan *float64
}

func NewProductsRepository(db *gorm.DB) *ProductsRepository {
	return &ProductsRepository{
		db: db,
	}
}

// Unused method, but keeping for potential future use
func (r *ProductsRepository) GetAllProducts() ([]Product, error) {
	var products []Product
	if err := r.db.
		Preload("Variants").
		Preload("Category").
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *ProductsRepository) GetFilteredProducts(offset, limit int, filters ProductFilters) ([]Product, int64, error) {
	var products []Product
	var total int64

	query := r.db.Model(&Product{}).
		Joins("LEFT JOIN categories ON categories.id = products.category_id").
		Preload("Category")

	// Filter
	if filters.CategoryCode != "" {
		query = query.Where("categories.code = ?", filters.CategoryCode)
	}
	if filters.PriceLessThan != nil {
		query = query.Where("products.price < ?", *filters.PriceLessThan)
	}

	// Count total after filtering
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *ProductsRepository) GetByCode(code string) (*Product, error) {
	var product Product
	if err := r.db.
		Preload("Variants").
		Preload("Category").
		Where("code = ?", code).
		First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}
