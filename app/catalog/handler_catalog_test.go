package catalog

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// --- Mock Repo ---

type MockProductRepo struct {
	SourceProducts []models.Product
	Err            error

	// Fields to capture call arguments
	lastCalledOffset  int
	lastCalledLimit   int
	lastCalledFilters models.ProductFilters
	lastCalledCode    string
}

func (m *MockProductRepo) GetAllProducts() ([]models.Product, error) {
	return nil, nil
}

func (m *MockProductRepo) GetFilteredProducts(offset, limit int, filters models.ProductFilters) ([]models.Product, int64, error) {
	m.lastCalledOffset = offset
	m.lastCalledLimit = limit
	m.lastCalledFilters = filters

	if m.Err != nil {
		return nil, 0, m.Err
	}

	// Simulate filtering
	var filteredProducts []models.Product
	for _, p := range m.SourceProducts {
		match := true
		// Category filter
		if filters.CategoryCode != "" && p.Category.Code != filters.CategoryCode {
			match = false
		}
		// Price filter
		if filters.PriceLessThan != nil && p.Price.InexactFloat64() >= *filters.PriceLessThan {
			match = false
		}

		if match {
			filteredProducts = append(filteredProducts, p)
		}
	}

	total := int64(len(filteredProducts))

	// Simulate pagination
	start := offset
	if start > len(filteredProducts) {
		start = len(filteredProducts)
	}
	end := offset + limit
	if end > len(filteredProducts) {
		end = len(filteredProducts)
	}

	return filteredProducts[start:end], total, nil
}

func (m *MockProductRepo) GetByCode(code string) (*models.Product, error) {
	m.lastCalledCode = code

	if m.Err != nil {
		return nil, m.Err
	}

	for _, p := range m.SourceProducts {
		if p.Code == code {
			product := p
			return &product, nil
		}
	}
	return nil, models.ErrProductNotFound
}

// --- Helpers ---

func newTestProduct(code, categoryCode, categoryName string, price float64) models.Product {
	return models.Product{
		Code:  code,
		Price: decimal.NewFromFloat(price),
		Category: models.Category{
			Code: categoryCode,
			Name: categoryName,
		},
	}
}

// --- Tests ---

func TestHandleGet(t *testing.T) {
	allMockProducts := []models.Product{
		newTestProduct("PROD001", "shoes", "Shoes", 19.99),
		newTestProduct("PROD002", "clothing", "Clothing", 24.99),
		newTestProduct("PROD003", "accessories", "Accessories", 10.00),
		newTestProduct("PROD004", "clothing", "Clothing", 95.50),
	}

	testCases := []struct {
		name               string
		url                string
		mockRepoSetup      func() *MockProductRepo
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
		checkRepoCalls     func(t *testing.T, repo *MockProductRepo)
	}{
		{
			name: "Success with default pagination",
			url:  "/catalog",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{
					SourceProducts: allMockProducts,
				}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 4, resp.Total)
				assert.Len(t, resp.Products, 4)
				assert.Equal(t, "PROD001", resp.Products[0].Code)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, 0, repo.lastCalledOffset, "Expected default offset 0")
				assert.Equal(t, 10, repo.lastCalledLimit, "Expected default limit 10")
				assert.Empty(t, repo.lastCalledFilters.CategoryCode)
				assert.Nil(t, repo.lastCalledFilters.PriceLessThan)
			},
		},
		{
			name: "Success with custom pagination",
			url:  "/catalog?offset=1&limit=2",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 4, resp.Total)
				assert.Len(t, resp.Products, 2)
				assert.Equal(t, "PROD002", resp.Products[0].Code)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, 1, repo.lastCalledOffset)
				assert.Equal(t, 2, repo.lastCalledLimit)
			},
		},
		{
			name: "Pagination with out-of-bounds values",
			url:  "/catalog?offset=-10&limit=200",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, 0, repo.lastCalledOffset, "Offset should be clamped to 0")
				assert.Equal(t, 100, repo.lastCalledLimit, "Limit should be clamped to 100")
			},
		},
		{
			name: "Pagination with lower bound limit",
			url:  "/catalog?limit=0",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, 1, repo.lastCalledLimit, "Limit should be clamped to 1")
			},
		},
		{
			name: "Filter by category and check response",
			url:  "/catalog?category=clothing",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{
					SourceProducts: allMockProducts,
				}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 2, resp.Total)
				assert.Len(t, resp.Products, 2)
				assert.Equal(t, "PROD002", resp.Products[0].Code)
				assert.Equal(t, "PROD004", resp.Products[1].Code)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "clothing", repo.lastCalledFilters.CategoryCode)
				assert.Nil(t, repo.lastCalledFilters.PriceLessThan)
			},
		},
		{
			name: "Filter by price less than",
			url:  "/catalog?price_lt=20",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 2, resp.Total)
				assert.Len(t, resp.Products, 2)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.NotNil(t, repo.lastCalledFilters.PriceLessThan)
				assert.Equal(t, 20.0, *repo.lastCalledFilters.PriceLessThan)
				assert.Empty(t, repo.lastCalledFilters.CategoryCode)
			},
		},
		{
			name: "Combined filters",
			url:  "/catalog?category=clothing&price_lt=30",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 1, resp.Total)
				assert.Len(t, resp.Products, 1)
				assert.Equal(t, "PROD002", resp.Products[0].Code)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "clothing", repo.lastCalledFilters.CategoryCode)
				assert.NotNil(t, repo.lastCalledFilters.PriceLessThan)
				assert.Equal(t, 30.0, *repo.lastCalledFilters.PriceLessThan)
			},
		},
		{
			name: "Empty result from repo",
			url:  "/catalog?category=nonexistent",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 0, resp.Total)
				assert.Len(t, resp.Products, 0)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "nonexistent", repo.lastCalledFilters.CategoryCode)
			},
		},
		{
			name: "Repository error",
			url:  "/catalog",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{Err: errors.New("db down")}
			},
			expectedStatusCode: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "failed to get products", errResp["error"])
			},
		},
		{
			name: "Invalid query param values are ignored",
			url:  "/catalog?offset=abc&limit=xyz&price_lt=def",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp Response
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, 4, resp.Total)
				assert.Len(t, resp.Products, 4)
			},
			checkRepoCalls: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, 0, repo.lastCalledOffset, "Expected default offset for invalid value")
				assert.Equal(t, 10, repo.lastCalledLimit, "Expected default limit for invalid value")
				assert.Nil(t, repo.lastCalledFilters.PriceLessThan, "Expected nil price filter for invalid value")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := tc.mockRepoSetup()
			handler := NewCatalogHandler(mockRepo)
			req := httptest.NewRequest("GET", tc.url, nil)
			rec := httptest.NewRecorder()

			// Act
			handler.HandleGet(rec, req)

			// Assert
			assert.Equal(t, tc.expectedStatusCode, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}

			if tc.checkRepoCalls != nil {
				tc.checkRepoCalls(t, mockRepo)
			}
		})
	}
}
