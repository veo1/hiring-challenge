package categories

import (
	"encoding/json"
	"net/http"

	"github.com/mytheresa/go-hiring-challenge/models"
)

type CategoryResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CategoryProvider interface {
	GetAllCategories() ([]models.Category, error)
	CreateCategory(category *models.Category) error
}

type CategoryHandler struct {
	repo CategoryProvider
}

func NewCategoryHandler(r CategoryProvider) *CategoryHandler {
	return &CategoryHandler{repo: r}
}

func (h *CategoryHandler) HandleGetAll(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.GetAllCategories()
	if err != nil {
		http.Error(w, "failed to fetch categories", http.StatusInternalServerError)
		return
	}

	response := make([]CategoryResponse, len(categories))
	for i, c := range categories {
		response[i] = CategoryResponse{
			Code: c.Code,
			Name: c.Name,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *CategoryHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if input.Code == "" || input.Name == "" {
		http.Error(w, "Missing code or name", http.StatusBadRequest)
		return
	}

	category := &models.Category{
		Code: input.Code,
		Name: input.Name,
	}

	if err := h.repo.CreateCategory(category); err != nil {
		http.Error(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Category created successfully",
	})
}
