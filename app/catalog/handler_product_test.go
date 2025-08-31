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

// --- Response Struct ---

// ProductDetailResponse defines the structure for a single product's JSON response.
type ProductDetailResponse struct {
	Code     string    `json:"code"`
	Price    float64   `json:"price"`
	Category Category  `json:"category"`
	Variants []Variant `json:"variants"`
}

// --- Tests ---

func TestHandleGetProduct(t *testing.T) {
	allMockProducts := []models.Product{
		{
			Code:     "PROD001",
			Price:    decimal.NewFromFloat(15.50),
			Category: models.Category{Code: "clothing", Name: "Clothing"},
			Variants: []models.Variant{
				{Name: "Red Small", SKU: "SKU001-A", Price: decimal.Decimal{}}, // empty, should inherit
				{Name: "Red Medium", SKU: "SKU001-B", Price: decimal.NewFromFloat(17.75)},
			},
		},
		{
			Code:     "PROD100",
			Price:    decimal.NewFromFloat(30.00),
			Category: models.Category{Code: "shoes", Name: "Shoes"},
			Variants: []models.Variant{},
		},
		{
			Code:  "PROD300",
			Price: decimal.NewFromFloat(0),
			Variants: []models.Variant{
				{Name: "No Price", SKU: "SKU-ZERO", Price: decimal.Decimal{}},
			},
		},
	}

	testCases := []struct {
		name               string
		productCode        string
		mockRepoSetup      func() *MockProductRepo
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
		checkRepoCall      func(t *testing.T, repo *MockProductRepo)
	}{
		{
			name:        "Success with variants and price inheritance",
			productCode: "PROD001",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProductDetailResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "PROD001", resp.Code)
				assert.Equal(t, 15.50, resp.Price)
				assert.Equal(t, "clothing", resp.Category.Code)
				assert.Len(t, resp.Variants, 2)
				assert.Equal(t, 15.50, resp.Variants[0].Price, "Variant should inherit product price")
				assert.Equal(t, 17.75, resp.Variants[1].Price, "Variant should have its own price")
			},
			checkRepoCall: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "PROD001", repo.lastCalledCode)
			},
		},
		{
			name:        "Product not found",
			productCode: "NONEXISTENT",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "Product not found", errResp["error"])
			},
			checkRepoCall: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "NONEXISTENT", repo.lastCalledCode)
			},
		},
		{
			name:        "Repository internal error",
			productCode: "PROD-ERR",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{Err: errors.New("db connection lost")}
			},
			expectedStatusCode: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to retrieve product", errResp["error"])
			},
			checkRepoCall: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "PROD-ERR", repo.lastCalledCode)
			},
		},
		{
			name:        "Product with no variants",
			productCode: "PROD100",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProductDetailResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "PROD100", resp.Code)
				assert.Len(t, resp.Variants, 0)
			},
		},
		{
			name:        "Product with zero price fallback",
			productCode: "PROD300",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp ProductDetailResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Len(t, resp.Variants, 1)
				assert.Equal(t, 0.0, resp.Variants[0].Price)
			},
		},
		{
			name:        "Empty product code in path",
			productCode: "",
			mockRepoSetup: func() *MockProductRepo {
				return &MockProductRepo{SourceProducts: allMockProducts}
			},
			expectedStatusCode: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "Product not found", errResp["error"])
			},
			checkRepoCall: func(t *testing.T, repo *MockProductRepo) {
				assert.Equal(t, "", repo.lastCalledCode)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := tc.mockRepoSetup()
			handler := NewCatalogHandler(mockRepo)
			req := httptest.NewRequest("GET", "/catalog/"+tc.productCode, nil)
			req.SetPathValue("code", tc.productCode)
			rec := httptest.NewRecorder()

			// Act
			handler.HandleGetProduct(rec, req)

			// Assert
			assert.Equal(t, tc.expectedStatusCode, rec.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}

			if tc.checkRepoCall != nil {
				tc.checkRepoCall(t, mockRepo)
			}
		})
	}
}
