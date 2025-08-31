package categories

import (
	"encoding/json"
	"net/http"

	"github.com/mytheresa/go-hiring-challenge/app/api"
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
		api.ErrorResponse(w, http.StatusInternalServerError, "failed to fetch categories")
		return
	}

	response := make([]CategoryResponse, len(categories))
	for i, c := range categories {
		response[i] = CategoryResponse{
			Code: c.Code,
			Name: c.Name,
		}
	}

	api.OKResponse(w, response)
}

func (h *CategoryHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if input.Code == "" || input.Name == "" {
		api.ErrorResponse(w, http.StatusBadRequest, "Missing code or name")
		return
	}

	category := &models.Category{
		Code: input.Code,
		Name: input.Name,
	}

	if err := h.repo.CreateCategory(category); err != nil {
		api.ErrorResponse(w, http.StatusInternalServerError, "Failed to create category")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Category created successfully",
	})
}
