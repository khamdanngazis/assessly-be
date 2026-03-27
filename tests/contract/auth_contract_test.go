package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/delivery/http/handler"
	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/infrastructure/auth"
	authUC "github.com/assessly/assessly-be/internal/usecase/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthRegisterContract validates the HTTP contract for POST /api/v1/auth/register
func TestAuthRegisterContract(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepository{}
	mockPasswordHasher := &MockPasswordHasher{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	registerUC := authUC.NewRegisterUserUseCase(mockUserRepo, mockPasswordHasher, logger)
	authHandler := handler.NewAuthHandler(registerUC, nil, nil, nil, logger)

	t.Run("should return 201 with correct response schema on successful registration", func(t *testing.T) {
		// Prepare mock responses
		expectedUser := &domain.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			Role:      domain.RoleCreator,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		mockUserRepo.FindByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound{Resource: "user", ID: email}
		}
		mockUserRepo.CreateFunc = func(ctx context.Context, user *domain.User) error {
			user.ID = expectedUser.ID
			user.CreatedAt = expectedUser.CreatedAt
			user.UpdatedAt = expectedUser.UpdatedAt
			return nil
		}
		mockPasswordHasher.HashFunc = func(password string) (string, error) {
			return "hashed_password", nil
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
			"role":     "creator",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		authHandler.Register(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusCreated, w.Code, "should return 201 Created")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate required fields exist
		assert.Contains(t, resp, "id", "response should contain id field")
		assert.Contains(t, resp, "email", "response should contain email field")
		assert.Contains(t, resp, "role", "response should contain role field")
		assert.Contains(t, resp, "created_at", "response should contain created_at field")

		// Validate field types
		assert.IsType(t, "", resp["id"], "id should be string")
		assert.IsType(t, "", resp["email"], "email should be string")
		assert.IsType(t, "", resp["role"], "role should be string")
		assert.IsType(t, "", resp["created_at"], "created_at should be string")

		// Validate field values
		assert.Equal(t, "test@example.com", resp["email"], "email should match request")
		assert.Equal(t, "creator", resp["role"], "role should match request")

		// Validate that password is NOT in response
		assert.NotContains(t, resp, "password", "response should NOT contain password field")
	})

	t.Run("should return 400 on invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when role is invalid", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
			"role":     "invalid_role",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
		assert.Contains(t, resp["error"], "creator", "error message should mention valid roles")
	})

	t.Run("should return 409 when user already exists", func(t *testing.T) {
		// Mock user exists
		mockUserRepo.FindByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{Email: email}, nil
		}

		reqBody := map[string]interface{}{
			"email":    "existing@example.com",
			"password": "password123",
			"role":     "creator",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Register(w, req)

		assert.Equal(t, http.StatusConflict, w.Code, "should return 409 Conflict")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when validation fails", func(t *testing.T) {
		// Mock validation - empty email
		mockUserRepo.FindByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
			return nil, nil
		}

		reqBody := map[string]interface{}{
			"email":    "",  // Empty email
			"password": "password123",
			"role":     "creator",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Register(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// TestAuthLoginContract validates the HTTP contract for POST /api/v1/auth/login
func TestAuthLoginContract(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepository{}
	mockPasswordHasher := &MockPasswordHasher{}
	mockJWTService := auth.NewJWTService("test-secret-key", "assessly-test", 24)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create real use case with mocks
	loginUC := authUC.NewLoginUserUseCase(mockUserRepo, mockPasswordHasher, mockJWTService, logger)
	authHandler := handler.NewAuthHandler(nil, loginUC, nil, nil, logger)

	t.Run("should return 200 with correct response schema on successful login", func(t *testing.T) {
		// Prepare mock user with hashed password
		userID := uuid.New()
		mockUserRepo.FindByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{
				ID:           userID,
				Email:        "test@example.com",
				PasswordHash: "hashed_password",
				Role:         domain.RoleCreator,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}, nil
		}
		mockPasswordHasher.CompareFunc = func(hashedPassword, password string) error {
			return nil // Password matches
		}

		// Prepare request
		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		authHandler.Login(w, req)

		// Validate HTTP status code
		assert.Equal(t, http.StatusOK, w.Code, "should return 200 OK")

		// Validate response Content-Type
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"), "should return JSON content type")

		// Validate response body schema
		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "response should be valid JSON")

		// Validate top-level fields
		assert.Contains(t, resp, "token", "response should contain token field")
		assert.Contains(t, resp, "user", "response should contain user field")

		// Validate token field type
		assert.IsType(t, "", resp["token"], "token should be string")
		assert.NotEmpty(t, resp["token"], "token should not be empty")

		// Validate user object schema
		user, ok := resp["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Contains(t, user, "id", "user should contain id field")
		assert.Contains(t, user, "email", "user should contain email field")
		assert.Contains(t, user, "role", "user should contain role field")

		// Validate user field types
		assert.IsType(t, "", user["id"], "user.id should be string")
		assert.IsType(t, "", user["email"], "user.email should be string")
		assert.IsType(t, "", user["role"], "user.role should be string")

		// Validate user field values
		assert.Equal(t, "test@example.com", user["email"], "email should match")
		assert.Equal(t, "creator", user["role"], "role should match")

		// Validate that password is NOT in response
		assert.NotContains(t, user, "password", "user object should NOT contain password field")
		assert.NotContains(t, user, "password_hash", "user object should NOT contain password_hash field")
	})

	t.Run("should return 400 on invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 401 on invalid credentials", func(t *testing.T) {
		// Mock user not found or password mismatch
		mockUserRepo.FindByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound{Resource: "user", ID: email}
		}

		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "wrongpassword",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Login(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "should return 401 Unauthorized")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})

	t.Run("should return 400 when validation fails", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "",  // Empty email
			"password": "password123",
		}
		reqJSON, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(reqJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		authHandler.Login(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "should return 400 Bad Request")

		var resp map[string]interface{}
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err, "error response should be valid JSON")
		assert.Contains(t, resp, "error", "error response should contain error field")
	})
}

// Mock implementations for contract tests

type MockUserRepository struct {
	FindByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	CreateFunc      func(ctx context.Context, user *domain.User) error
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.FindByEmailFunc != nil {
		return m.FindByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	return nil
}

type MockPasswordHasher struct {
	HashFunc    func(password string) (string, error)
	CompareFunc func(hashedPassword, password string) error
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	if m.HashFunc != nil {
		return m.HashFunc(password)
	}
	return "", nil
}

func (m *MockPasswordHasher) Compare(hashedPassword, password string) error {
	if m.CompareFunc != nil {
		return m.CompareFunc(hashedPassword, password)
	}
	return nil
}
