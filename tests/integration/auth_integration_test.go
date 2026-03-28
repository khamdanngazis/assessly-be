package integration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/infrastructure/auth"
	"github.com/assessly/assessly-be/internal/infrastructure/postgres"
	authUC "github.com/assessly/assessly-be/internal/usecase/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthIntegration_RegisterLoginReset tests the complete auth flow:
// 1. Register a new user
// 2. Login with credentials
// 3. Request password reset
// 4. Reset password with token
// 5. Login with new password
func TestAuthIntegration_RegisterLoginReset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test environment
	loadTestEnv(t)

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Setup test database (auto-create and run migrations)
	pool := setupTestDatabase(t, ctx)
	defer pool.Close()

	// Setup repositories and services
	userRepo := postgres.NewUserRepository(pool)
	passwordHasher := auth.NewPasswordHasher(10) // Lower cost for faster tests
	
	// Get JWT config from environment
	jwtSecret := getEnv("JWT_SECRET", "test-secret-key")
	jwtIssuer := getEnv("JWT_ISSUER", "assessly-test")
	tokenGen := auth.NewJWTService(jwtSecret, jwtIssuer, 24)
	
	// Mock email sender (doesn't actually send emails in tests)
	emailSender := &mockEmailSender{}
	resetTokenGen := &mockResetTokenGenerator{jwtService: tokenGen}

	// Create use cases
	registerUC := authUC.NewRegisterUserUseCase(userRepo, passwordHasher, logger)
	loginUC := authUC.NewLoginUserUseCase(userRepo, passwordHasher, tokenGen, logger)
	requestResetUC := authUC.NewRequestPasswordResetUseCase(userRepo, resetTokenGen, emailSender, logger)
	resetPasswordUC := authUC.NewResetPasswordUseCase(userRepo, resetTokenGen, passwordHasher, logger)

	// Test data
	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "SecurePassword123!"
	newPassword := "NewSecurePassword456!"

	// Step 1: Register new user
	t.Run("Register", func(t *testing.T) {
		req := authUC.RegisterUserRequest{
			Email:    email,
			Password: password,
			Role:     domain.UserRole("creator"),
		}

		user, err := registerUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, domain.UserRole("creator"), user.Role)
		assert.NotEqual(t, password, user.PasswordHash) // Should be hashed
	})

	// Step 2: Login with correct credentials
	var loginToken string
	var userID uuid.UUID
	t.Run("Login_Success", func(t *testing.T) {
		req := authUC.LoginUserRequest{
			Email:    email,
			Password: password,
		}

		resp, err := loginUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, email, resp.User.Email)
		loginToken = resp.Token
		userID = resp.User.ID

		// Validate the token
		claims, err := tokenGen.ValidateToken(resp.Token)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "creator", claims.Role)
	})

	// Step 3: Login with wrong password should fail
	t.Run("Login_WrongPassword", func(t *testing.T) {
		req := authUC.LoginUserRequest{
			Email:    email,
			Password: "WrongPassword123!",
		}

		resp, err := loginUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	// Step 4: Request password reset
	var resetToken string
	t.Run("RequestPasswordReset", func(t *testing.T) {
		req := authUC.RequestPasswordResetRequest{
			Email:    email,
			ResetURL: "https://assessly.com/reset",
		}

		err := requestResetUC.Execute(ctx, req)
		require.NoError(t, err)

		// Check that email was "sent"
		assert.Equal(t, 1, emailSender.CallCount)
		assert.Equal(t, email, emailSender.LastRecipient)
		resetToken = emailSender.LastToken
		assert.NotEmpty(t, resetToken)
	})

	// Step 5: Reset password with valid token
	t.Run("ResetPassword_Success", func(t *testing.T) {
		req := authUC.ResetPasswordRequest{
			Token:       resetToken,
			NewPassword: newPassword,
		}

		err := resetPasswordUC.Execute(ctx, req)
		require.NoError(t, err)

		// Verify old password no longer works
		loginReq := authUC.LoginUserRequest{
			Email:    email,
			Password: password,
		}
		resp, err := loginUC.Execute(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	// Step 6: Login with new password
	t.Run("Login_WithNewPassword", func(t *testing.T) {
		req := authUC.LoginUserRequest{
			Email:    email,
			Password: newPassword,
		}

		resp, err := loginUC.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, email, resp.User.Email)
		assert.Equal(t, userID, resp.User.ID)
	})

	// Step 7: Duplicate registration should fail
	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		req := authUC.RegisterUserRequest{
			Email:    email,
			Password: "AnotherPassword123!",
			Role:     domain.UserRole("reviewer"),
		}

		user, err := registerUC.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	// Step 8: Token validation
	t.Run("ValidateToken", func(t *testing.T) {
		claims, err := tokenGen.ValidateToken(loginToken)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, "creator", claims.Role)
	})

	// Step 9: Invalid token should fail
	t.Run("ValidateToken_Invalid", func(t *testing.T) {
		claims, err := tokenGen.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	// Cleanup: Delete test user
	t.Run("Cleanup", func(t *testing.T) {
		_, err := pool.Exec(ctx, "DELETE FROM users WHERE email = $1", email)
		require.NoError(t, err)
	})
}

// Mock implementations for testing

type mockEmailSender struct {
	CallCount     int
	LastRecipient string
	LastToken     string
}

func (m *mockEmailSender) SendPasswordReset(to, token, resetURL string) error {
	m.CallCount++
	m.LastRecipient = to
	m.LastToken = token
	return nil
}

func (m *mockEmailSender) SendTestAccessToken(to, testTitle, accessToken, accessURL string) error {
	return nil
}

type mockResetTokenGenerator struct {
	jwtService *auth.JWTService
}

func (m *mockResetTokenGenerator) GenerateResetToken(userID, email string) (string, error) {
	// Generate a JWT token with role "reset"
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}
	return m.jwtService.GenerateToken(parsedID, email, "reset")
}

func (m *mockResetTokenGenerator) ValidateToken(tokenString string) (string, string, string, error) {
	// Validate JWT token
	claims, err := m.jwtService.ValidateToken(tokenString)
	if err != nil {
		return "", "", "", domain.ErrUnauthorized{Message: "invalid token"}
	}
	return claims.UserID, claims.Email, claims.Role, nil
}
