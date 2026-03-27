package unit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test Setup
func getTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// RegisterUserUseCase Tests

func TestRegisterUser_Success(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockHasher := new(MockPasswordHasher)
	logger := getTestLogger()

	useCase := auth.NewRegisterUserUseCase(mockUserRepo, mockHasher, logger)

	req := auth.RegisterUserRequest{
		Email:    "john@example.com",
		Password: "password123",
		Role:     domain.RoleCreator,
	}

	hashedPassword := "$2a$12$hashed"

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(nil, domain.ErrNotFound{Resource: "user", ID: req.Email})
	mockHasher.On("Hash", req.Password).Return(hashedPassword, nil)
	mockUserRepo.On("Create", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
		return user.Email == req.Email &&
			user.PasswordHash == hashedPassword &&
			user.Role == req.Role
	})).Return(nil)

	// Act
	user, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, hashedPassword, user.PasswordHash)
	assert.Equal(t, req.Role, user.Role)
	mockUserRepo.AssertExpectations(t)
	mockHasher.AssertExpectations(t)
}

func TestRegisterUser_EmailAlreadyExists(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockHasher := new(MockPasswordHasher)
	logger := getTestLogger()

	useCase := auth.NewRegisterUserUseCase(mockUserRepo, mockHasher, logger)

	req := auth.RegisterUserRequest{
		Email:    "existing@example.com",
		Password: "password123",
		Role:     domain.RoleCreator,
	}

	existingUser := &domain.User{
		ID:    uuid.New(),
		Email: req.Email,
		Role:  domain.RoleCreator,
	}

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(existingUser, nil)

	// Act
	user, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.IsType(t, domain.ErrConflict{}, err)
	mockUserRepo.AssertExpectations(t)
	mockHasher.AssertNotCalled(t, "Hash")
	mockUserRepo.AssertNotCalled(t, "Create")
}

func TestRegisterUser_ValidationFailure(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockHasher := new(MockPasswordHasher)
	logger := getTestLogger()

	useCase := auth.NewRegisterUserUseCase(mockUserRepo, mockHasher, logger)

	tests := []struct {
		name    string
		request auth.RegisterUserRequest
		errType interface{}
	}{
		{
			name: "Empty email",
			request: auth.RegisterUserRequest{
				Email:    "",
				Password: "password123",
				Role:     domain.RoleCreator,
			},
			errType: domain.ErrValidation{},
		},
		{
			name: "Short password",
			request: auth.RegisterUserRequest{
				Email:    "john@example.com",
				Password: "short",
				Role:     domain.RoleCreator,
			},
			errType: domain.ErrValidation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			user, err := useCase.Execute(context.Background(), tt.request)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, user)
			assert.IsType(t, tt.errType, err)
		})
	}
}

// LoginUserUseCase Tests

func TestLoginUser_Success(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockHasher := new(MockPasswordHasher)
	mockTokenGen := new(MockTokenGenerator)
	logger := getTestLogger()

	useCase := auth.NewLoginUserUseCase(mockUserRepo, mockHasher, mockTokenGen, logger)

	req := auth.LoginUserRequest{
		Email:    "john@example.com",
		Password: "password123",
	}

	userID := uuid.New()
	hashedPassword := "$2a$12$hashed"
	expectedToken := "jwt-token-here"

	existingUser := &domain.User{
		ID:             userID,
		Email:          req.Email,
		PasswordHash:   hashedPassword,
		Role:           domain.RoleCreator,
	}

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(existingUser, nil)
	mockHasher.On("Compare", hashedPassword, req.Password).Return(nil)
	mockTokenGen.On("GenerateToken", userID, req.Email, string(domain.RoleCreator)).Return(expectedToken, nil)

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedToken, response.Token)
	assert.Equal(t, existingUser.ID, response.User.ID)
	assert.Equal(t, existingUser.Email, response.User.Email)
	mockUserRepo.AssertExpectations(t)
	mockHasher.AssertExpectations(t)
	mockTokenGen.AssertExpectations(t)
}

func TestLoginUser_InvalidCredentials(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockHasher := new(MockPasswordHasher)
	mockTokenGen := new(MockTokenGenerator)
	logger := getTestLogger()

	useCase := auth.NewLoginUserUseCase(mockUserRepo, mockHasher, mockTokenGen, logger)

	req := auth.LoginUserRequest{
		Email:    "john@example.com",
		Password: "wrongpassword",
	}

	hashedPassword := "$2a$12$hashed"
	existingUser := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         domain.RoleCreator,
	}

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(existingUser, nil)
	mockHasher.On("Compare", hashedPassword, req.Password).Return(errors.New("password mismatch"))

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.IsType(t, domain.ErrUnauthorized{}, err)
	mockUserRepo.AssertExpectations(t)
	mockHasher.AssertExpectations(t)
	mockTokenGen.AssertNotCalled(t, "GenerateToken")
}

func TestLoginUser_UserNotFound(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockHasher := new(MockPasswordHasher)
	mockTokenGen := new(MockTokenGenerator)
	logger := getTestLogger()

	useCase := auth.NewLoginUserUseCase(mockUserRepo, mockHasher, mockTokenGen, logger)

	req := auth.LoginUserRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(nil, domain.ErrNotFound{Resource: "user", ID: req.Email})

	// Act
	response, err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.IsType(t, domain.ErrUnauthorized{}, err)
	mockUserRepo.AssertExpectations(t)
	mockHasher.AssertNotCalled(t, "Compare")
	mockTokenGen.AssertNotCalled(t, "GenerateToken")
}

// RequestPasswordResetUseCase Tests

func TestRequestPasswordReset_Success(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockTokenGen := new(MockResetTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := getTestLogger()

	useCase := auth.NewRequestPasswordResetUseCase(mockUserRepo, mockTokenGen, mockEmailSender, logger)

	req := auth.RequestPasswordResetRequest{
		Email:    "john@example.com",
		ResetURL: "https://app.assessly.com/reset",
	}

	existingUser := &domain.User{
		ID:    uuid.New(),
		Email: req.Email,
		Role:  domain.RoleCreator,
	}

	resetToken := "reset-token-123"

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(existingUser, nil)
	mockTokenGen.On("GenerateResetToken", existingUser.ID.String(), req.Email).Return(resetToken, nil)
	mockEmailSender.On("SendPasswordReset", req.Email, resetToken, req.ResetURL).Return(nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	mockUserRepo.AssertExpectations(t)
	mockTokenGen.AssertExpectations(t)
	mockEmailSender.AssertExpectations(t)
}

func TestRequestPasswordReset_UserNotFound(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockTokenGen := new(MockResetTokenGenerator)
	mockEmailSender := new(MockEmailSender)
	logger := getTestLogger()

	useCase := auth.NewRequestPasswordResetUseCase(mockUserRepo, mockTokenGen, mockEmailSender, logger)

	req := auth.RequestPasswordResetRequest{
		Email:    "nonexistent@example.com",
		ResetURL: "https://app.assessly.com/reset",
	}

	// Mock expectations
	mockUserRepo.On("FindByEmail", mock.Anything, req.Email).Return(nil, domain.ErrNotFound{Resource: "user", ID: req.Email})

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err) // Should not return error for security (don't reveal if email exists)
	mockUserRepo.AssertExpectations(t)
	mockTokenGen.AssertNotCalled(t, "GenerateResetToken")
	mockEmailSender.AssertNotCalled(t, "SendPasswordReset")
}

// ResetPasswordUseCase Tests

func TestResetPassword_Success(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockTokenValidator := new(MockTokenValidator)
	mockHasher := new(MockPasswordHasher)
	logger := getTestLogger()

	useCase := auth.NewResetPasswordUseCase(mockUserRepo, mockTokenValidator, mockHasher, logger)

	email := "john@example.com"
	userID := uuid.New()
	req := auth.ResetPasswordRequest{
		Token:       "valid-reset-token",
		NewPassword: "newpassword123",
	}

	existingUser := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: "$2a$12$oldhash",
		Role:         domain.RoleCreator,
	}

	newHashedPassword := "$2a$12$newhash"

	// Mock expectations
	mockTokenValidator.On("ValidateToken", req.Token).Return(userID.String(), email, "reset", nil)
	mockUserRepo.On("FindByID", mock.Anything, userID).Return(existingUser, nil)
	mockHasher.On("Hash", req.NewPassword).Return(newHashedPassword, nil)
	mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
		return user.Email == email && user.PasswordHash == newHashedPassword
	})).Return(nil)

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	mockTokenValidator.AssertExpectations(t)
	mockUserRepo.AssertExpectations(t)
	mockHasher.AssertExpectations(t)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockTokenValidator := new(MockTokenValidator)
	mockHasher := new(MockPasswordHasher)
	logger := getTestLogger()

	useCase := auth.NewResetPasswordUseCase(mockUserRepo, mockTokenValidator, mockHasher, logger)

	req := auth.ResetPasswordRequest{
		Token:       "invalid-token",
		NewPassword: "newpassword123",
	}

	// Mock expectations
	mockTokenValidator.On("ValidateToken", req.Token).Return("", "", "", errors.New("invalid token"))

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.IsType(t, domain.ErrUnauthorized{}, err)
	mockTokenValidator.AssertExpectations(t)
	mockUserRepo.AssertNotCalled(t, "FindByEmail")
	mockHasher.AssertNotCalled(t, "Hash")
	mockUserRepo.AssertNotCalled(t, "Update")
}

func TestResetPassword_ShortPassword(t *testing.T) {
	// Arrange
	mockUserRepo := new(MockUserRepository)
	mockTokenValidator := new(MockTokenValidator)
	mockHasher := new(MockPasswordHasher)
	logger := getTestLogger()

	useCase := auth.NewResetPasswordUseCase(mockUserRepo, mockTokenValidator, mockHasher, logger)

	req := auth.ResetPasswordRequest{
		Token:       "valid-token",
		NewPassword: "short",
	}

	// Act
	err := useCase.Execute(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.IsType(t, domain.ErrValidation{}, err)
	mockTokenValidator.AssertNotCalled(t, "ValidateToken")
}
