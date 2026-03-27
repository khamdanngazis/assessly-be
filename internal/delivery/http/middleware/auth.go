package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// UserRoleKey is the context key for user role
	UserRoleKey contextKey = "user_role"
)

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	SecretKey []byte
	Logger    *slog.Logger
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth middleware validates JWT tokens and extracts user information
func JWTAuth(config JWTConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				config.Logger.Warn("missing authorization header",
					"path", r.URL.Path,
					"method", r.Method,
				)
				unauthorized(w, "missing authorization header")
				return
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				config.Logger.Warn("invalid authorization header format",
					"path", r.URL.Path,
					"method", r.Method,
				)
				unauthorized(w, "invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return config.SecretKey, nil
			})

			if err != nil {
				config.Logger.Warn("invalid token",
					"error", err.Error(),
					"path", r.URL.Path,
					"method", r.Method,
				)
				unauthorized(w, "invalid token")
				return
			}

			// Extract claims
			claims, ok := token.Claims.(*Claims)
			if !ok || !token.Valid {
				config.Logger.Warn("invalid token claims",
					"path", r.URL.Path,
					"method", r.Method,
				)
				unauthorized(w, "invalid token")
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(requiredRole string, logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(UserRoleKey).(string)
			if !ok || role != requiredRole {
				logger.Warn("insufficient permissions",
					"required_role", requiredRole,
					"user_role", role,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"insufficient permissions"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// unauthorized sends 401 response
func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// GetUserID extracts user ID from request context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// GetUserRole extracts user role from request context
func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok
}
