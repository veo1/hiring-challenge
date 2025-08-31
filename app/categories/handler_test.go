package categories

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/stretchr/testify/assert"
)

// --- Mock Repository ---

type MockCategoryRepo struct {
	Categories []models.Category
	CreateErr  error
	ListErr    error
	LastSaved  *models.Category
}

func (m *MockCategoryRepo) GetAllCategories() ([]models.Category, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	return m.Categories, nil
}

func (m *MockCategoryRepo) CreateCategory(cat *models.Category) error {
	m.LastSaved = cat
	return m.CreateErr
}

// --- Tests: GET /categories ---

func TestHandleGetAll(t *testing.T) {
	testCases := []struct {
		name               string
		mockRepoSetup      func() *MockCategoryRepo
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "Success with multiple categories",
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{
					Categories: []models.Category{
						{Code: "clothing", Name: "Clothing"},
						{Code: "shoes", Name: "Shoes"},
					},
				}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp []CategoryResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Len(t, resp, 2)
				assert.Equal(t, "clothing", resp[0].Code)
				assert.Equal(t, "Shoes", resp[1].Name)
			},
		},
		{
			name: "Success with empty list",
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{
					Categories: []models.Category{},
				}
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp []CategoryResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Len(t, resp, 0)
			},
		},
		{
			name: "Repository error",
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{
					ListErr: errors.New("db down"),
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "failed to fetch categories", errResp["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := tc.mockRepoSetup()
			handler := NewCategoryHandler(mockRepo)
			req := httptest.NewRequest("GET", "/categories", nil)
			rec := httptest.NewRecorder()

			// Act
			handler.HandleGetAll(rec, req)

			// Assert
			assert.Equal(t, tc.expectedStatusCode, rec.Code)
			if tc.checkResponse != nil {
				tc.checkResponse(t, rec)
			}
		})
	}
}

// --- Tests: POST /categories ---

func TestHandleCreate(t *testing.T) {
	testCases := []struct {
		name               string
		requestBody        string
		mockRepoSetup      func() *MockCategoryRepo
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
		checkRepoCall      func(t *testing.T, repo *MockCategoryRepo)
	}{
		{
			name:        "Success",
			requestBody: `{"code":"accessories","name":"Accessories"}`,
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{}
			},
			expectedStatusCode: http.StatusCreated,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "Category created successfully", resp["message"])
			},
			checkRepoCall: func(t *testing.T, repo *MockCategoryRepo) {
				assert.NotNil(t, repo.LastSaved)
				assert.Equal(t, "accessories", repo.LastSaved.Code)
				assert.Equal(t, "Accessories", repo.LastSaved.Name)
			},
		},
		{
			name:        "Invalid JSON body",
			requestBody: `{invalid json`,
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{}
			},
			expectedStatusCode: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "Invalid JSON body", errResp["error"])
			},
			checkRepoCall: func(t *testing.T, repo *MockCategoryRepo) {
				assert.Nil(t, repo.LastSaved, "CreateCategory should not be called with invalid JSON")
			},
		},
		{
			name:        "Missing required fields (code)",
			requestBody: `{"name":"MissingCode"}`,
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{}
			},
			expectedStatusCode: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "Missing code or name", errResp["error"])
			},
			checkRepoCall: func(t *testing.T, repo *MockCategoryRepo) {
				assert.Nil(t, repo.LastSaved, "CreateCategory should not be called with missing fields")
			},
		},
		{
			name:        "Repository error on create",
			requestBody: `{"code":"toys","name":"Toys"}`,
			mockRepoSetup: func() *MockCategoryRepo {
				return &MockCategoryRepo{CreateErr: errors.New("insert failed")}
			},
			expectedStatusCode: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var errResp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&errResp)
				assert.NoError(t, err)
				assert.Equal(t, "Failed to create category", errResp["error"])
			},
			checkRepoCall: func(t *testing.T, repo *MockCategoryRepo) {
				assert.NotNil(t, repo.LastSaved, "CreateCategory should have been called")
				assert.Equal(t, "toys", repo.LastSaved.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			mockRepo := tc.mockRepoSetup()
			handler := NewCategoryHandler(mockRepo)
			req := httptest.NewRequest("POST", "/categories", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Act
			handler.HandleCreate(rec, req)

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
