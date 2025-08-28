package catalog

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mytheresa/go-hiring-challenge/models"
)

type Response struct {
	Total    int       `json:"total"`
	Products []Product `json:"products"`
}

type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Product struct {
	Code     string   `json:"code"`
	Price    float64  `json:"price"`
	Category Category `json:"category"`
}

type Variant struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

type ProductProvider interface {
	GetAllProducts() ([]models.Product, error)
	GetFilteredProducts(offset, limit int, filters models.ProductFilters) ([]models.Product, int64, error)
	GetByCode(code string) (*models.Product, error)
}

type CatalogHandler struct {
	repo ProductProvider
}

func NewCatalogHandler(r ProductProvider) *CatalogHandler {
	return &CatalogHandler{
		repo: r,
	}
}

func (h *CatalogHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	// Parse pagination query params
	offset := 0
	limit := 10

	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil {
			if l < 1 {
				limit = 1
			} else if l > 100 {
				limit = 100
			} else {
				limit = l
			}
		}
	}

	// Parse filters
	categoryCode := r.URL.Query().Get("category")

	var priceFilter *float64
	if priceStr := r.URL.Query().Get("price_lt"); priceStr != "" {
		if val, err := strconv.ParseFloat(priceStr, 64); err == nil {
			priceFilter = &val
		}
	}

	filters := models.ProductFilters{
		CategoryCode:  categoryCode,
		PriceLessThan: priceFilter,
	}

	res, total, err := h.repo.GetFilteredProducts(offset, limit, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	products := make([]Product, len(res))
	for i, p := range res {
		products[i] = Product{
			Code:  p.Code,
			Price: p.Price.InexactFloat64(),
			Category: Category{
				Code: p.Category.Code,
				Name: p.Category.Name,
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Total:    int(total),
		Products: products,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *CatalogHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	product, err := h.repo.GetByCode(code)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Map response
	variants := make([]Variant, len(product.Variants))
	for i, v := range product.Variants {
		price := v.Price
		if price.IsZero() {
			price = product.Price
		}
		variants[i] = Variant{
			Name:  v.Name,
			SKU:   v.SKU,
			Price: price.InexactFloat64(),
		}
	}

	response := struct {
		Code     string    `json:"code"`
		Price    float64   `json:"price"`
		Category Category  `json:"category"`
		Variants []Variant `json:"variants"`
	}{
		Code:  product.Code,
		Price: product.Price.InexactFloat64(),
		Category: Category{
			Code: product.Category.Code,
			Name: product.Category.Name,
		},
		Variants: variants,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
