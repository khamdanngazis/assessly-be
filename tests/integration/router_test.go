package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/delivery/http/middleware"
	"github.com/assessly/assessly-be/internal/delivery/http/router"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
)

var jwtSecret = []byte("test-secret-key-for-router-tests")

// generateTestJWT creates a JWT token with specified user ID and role
func generateTestJWT(userID, role string) string {
	claims := &middleware.Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtSecret)
	return tokenString
}

func TestRouterRoleEnforcement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	
	// Create minimal handlers (just for testing routing and auth)
	testHandler := &mockTestHandlerForRouter{}
	reviewHandler := &mockReviewHandlerForRouter{}
	
	// Create JWT middleware
	jwtMiddleware := middleware.JWTAuth(middleware.JWTConfig{
		SecretKey: jwtSecret,
		Logger:    logger,
	})
	
	// Setup router
	r := router.New()
	r.SetupRoutes(
		nil, // healthHandler
		nil, // metricsHandler
		nil, // authHandler
		testHandler,
		nil, // questionHandler
		nil, // submissionHandler
		reviewHandler,
		jwtMiddleware,
		logger,
	)
	
	t.Run("creator endpoints", func(t *testing.T) {
		t.Run("should allow creator to create test", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"title":         "Test Title",
				"description":   "Test Description",
				"allow_retakes": true,
			}
			bodyJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("user-123", "creator"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "creator should be able to create test")
		})
		
		t.Run("should deny reviewer from creating test", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"title":         "Test Title",
				"description":   "Test Description",
				"allow_retakes": true,
			}
			bodyJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("user-123", "reviewer"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusForbidden, w.Code, "reviewer should not be able to create test")
			
			var resp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&resp)
			assert.Contains(t, resp, "error")
		})
		
		t.Run("should deny participant from creating test", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"title":         "Test Title",
				"description":   "Test Description",
				"allow_retakes": true,
			}
			bodyJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("user-123", "participant"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusForbidden, w.Code, "participant should not be able to create test")
		})
		
		t.Run("should deny unauthenticated request", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"title":         "Test Title",
				"description":   "Test Description",
				"allow_retakes": true,
			}
			bodyJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code, "should require authentication")
		})
	})
	
	t.Run("reviewer endpoints", func(t *testing.T) {
		t.Run("should allow reviewer to add manual review", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"manual_score":    85.5,
				"manual_feedback": "Good work",
			}
			bodyJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPut, "/api/v1/reviews/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("reviewer-123", "reviewer"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "reviewer should be able to add manual review")
		})
		
		t.Run("should deny creator from adding manual review", func(t *testing.T) {
			reqBody := map[string]interface{}{
				"manual_score":    85.5,
				"manual_feedback": "Good work",
			}
			bodyJSON, _ := json.Marshal(reqBody)
			
			req := httptest.NewRequest(http.MethodPut, "/api/v1/reviews/550e8400-e29b-41d4-a716-446655440000", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("creator-123", "creator"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusForbidden, w.Code, "creator should not be able to add manual review")
		})
		
		t.Run("should allow reviewer to list submissions", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/550e8400-e29b-41d4-a716-446655440000/submissions", nil)
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("reviewer-123", "reviewer"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code, "reviewer should be able to list submissions")
		})
		
		t.Run("should deny creator from listing submissions", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/550e8400-e29b-41d4-a716-446655440000/submissions", nil)
			req.Header.Set("Authorization", "Bearer "+generateTestJWT("creator-123", "creator"))
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusForbidden, w.Code, "creator should not be able to list submissions")
		})
	})
	
	t.Run("JWT validation", func(t *testing.T) {
		t.Run("should reject invalid JWT", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Authorization", "Bearer invalid-token")
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code, "should reject invalid JWT")
		})
		
		t.Run("should reject expired JWT", func(t *testing.T) {
			claims := &middleware.Claims{
				UserID: "user-123",
				Role:   "creator",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
					IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				},
			}
			
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, _ := token.SignedString(jwtSecret)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Authorization", "Bearer "+tokenString)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code, "should reject expired JWT")
		})
		
		t.Run("should reject JWT with wrong signing method", func(t *testing.T) {
			claims := &middleware.Claims{
				UserID: "user-123",
				Role:   "creator",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}
			
			// Use RS256 instead of HS256
			token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
			tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
			
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tests", bytes.NewReader([]byte(`{}`)))
			req.Header.Set("Authorization", "Bearer "+tokenString)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusUnauthorized, w.Code, "should reject JWT with wrong signing method")
		})
	})
}

// Mock handlers for router testing
type mockTestHandlerForRouter struct{}

func (h *mockTestHandlerForRouter) CreateTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            "test-123",
		"title":         "Test",
		"description":   "Test",
		"allow_retakes": true,
		"is_published":  false,
	})
}

func (h *mockTestHandlerForRouter) PublishTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          "test-123",
		"is_published": true,
	})
}

func (h *mockTestHandlerForRouter) ListTests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tests": []map[string]interface{}{
			{
				"id":            "test-123",
				"title":         "Test",
				"description":   "Test",
				"allow_retakes": true,
				"is_published":  false,
			},
		},
		"pagination": map[string]interface{}{
			"page":      1,
			"page_size": 20,
			"total":     1,
		},
	})
}

type mockReviewHandlerForRouter struct{}

func (h *mockReviewHandlerForRouter) HandleAddManualReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":              "review-123",
		"answer_id":       "answer-123",
		"manual_score":    85.5,
		"manual_feedback": "Good work",
	})
}

func (h *mockReviewHandlerForRouter) HandleGetReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        "review-123",
		"answer_id": "answer-123",
	})
}

func (h *mockReviewHandlerForRouter) HandleListSubmissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"submissions": []interface{}{},
		"total":       0,
	})
}
