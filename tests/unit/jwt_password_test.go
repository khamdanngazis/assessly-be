package unit

import (
	"testing"
	"time"

	"github.com/assessly/assessly-be/internal/infrastructure/auth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// JWT Service Tests

func TestJWTService_GenerateAndValidate(t *testing.T) {
	// Arrange
	secret := "test-secret-key-must-be-at-least-32-characters-long"
	issuer := "assessly-test"
	expiryHours := 1

	jwtService := auth.NewJWTService(secret, issuer, expiryHours)

	userID := uuid.New()
	email := "test@example.com"
	role := "creator"

	// Act - Generate token
	token, err := jwtService.GenerateToken(userID, email, role)

	// Assert - Token generation
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Act - Validate token
	claims, err := jwtService.ValidateToken(token)

	// Assert - Token validation
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestJWTService_GenerateResetToken(t *testing.T) {
	// Arrange
	secret := "test-secret-key-must-be-at-least-32-characters-long"
	issuer := "assessly-test"
	expiryHours := 1

	jwtService := auth.NewJWTService(secret, issuer, expiryHours)

	userID := uuid.New()
	email := "test@example.com"

	// Act - Generate reset token
	token, err := jwtService.GenerateResetToken(userID, email)

	// Assert - Token generation
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Act - Validate reset token
	claims, err := jwtService.ValidateToken(token)

	// Assert - Token validation
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "reset", claims.Role) // Reset tokens have "reset" role
}

func TestJWTService_ValidateInvalidToken(t *testing.T) {
	// Arrange
	secret := "test-secret-key-must-be-at-least-32-characters-long"
	issuer := "assessly-test"
	expiryHours := 1

	jwtService := auth.NewJWTService(secret, issuer, expiryHours)

	invalidTokens := []string{
		"",
		"invalid-token",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
	}

	for _, invalidToken := range invalidTokens {
		t.Run("Invalid token: "+invalidToken, func(t *testing.T) {
			// Act
			_, err := jwtService.ValidateToken(invalidToken)

			// Assert
			assert.Error(t, err)
		})
	}
}

func TestJWTService_ValidateExpiredToken(t *testing.T) {
	// Arrange
	secret := "test-secret-key-must-be-at-least-32-characters-long"
	issuer := "assessly-test"
	expiryHours := -1 // Negative expiry to force immediate expiration

	jwtService := auth.NewJWTService(secret, issuer, expiryHours)

	userID := uuid.New()
	email := "test@example.com"
	role := "creator"

	// Act - Generate token
	token, err := jwtService.GenerateToken(userID, email, role)
	assert.NoError(t, err)

	// Wait a moment to ensure token is expired
	time.Sleep(100 * time.Millisecond)

	// Act - Validate expired token
	_, err = jwtService.ValidateToken(token)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestJWTService_ValidateTokenWithWrongSecret(t *testing.T) {
	// Arrange
	secret1 := "test-secret-key-must-be-at-least-32-characters-long"
	secret2 := "different-secret-key-must-be-at-least-32-chars"
	issuer := "assessly-test"
	expiryHours := 1

	jwtService1 := auth.NewJWTService(secret1, issuer, expiryHours)
	jwtService2 := auth.NewJWTService(secret2, issuer, expiryHours)

	userID := uuid.New()
	email := "test@example.com"
	role := "creator"

	// Act - Generate token with service1
	token, err := jwtService1.GenerateToken(userID, email, role)
	assert.NoError(t, err)

	// Act - Try to validate with service2 (wrong secret)
	_, err = jwtService2.ValidateToken(token)

	// Assert
	assert.Error(t, err)
}

// Password Hasher Tests

func TestPasswordHasher_HashAndCompare(t *testing.T) {
	// Arrange
	hasher := auth.NewPasswordHasher(10) // Lower cost for faster tests
	password := "mySecurePassword123"

	// Act - Hash password
	hashedPassword, err := hasher.Hash(password)

	// Assert - Hash generation
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, password, hashedPassword) // Hashed should be different from plain
	assert.True(t, len(hashedPassword) > 50)     // bcrypt hashes are long

	// Act - Compare correct password
	err = hasher.Compare(hashedPassword, password)

	// Assert - Correct password
	assert.NoError(t, err)

	// Act - Compare wrong password
	err = hasher.Compare(hashedPassword, "wrongPassword")

	// Assert - Wrong password
	assert.Error(t, err)
}

func TestPasswordHasher_HashDeterminism(t *testing.T) {
	// Arrange
	hasher := auth.NewPasswordHasher(10)
	password := "samePassword"

	// Act - Hash same password twice
	hash1, err1 := hasher.Hash(password)
	hash2, err2 := hasher.Hash(password)

	// Assert - Both hashes successful
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Assert - Hashes are different (bcrypt uses salt)
	assert.NotEqual(t, hash1, hash2)

	// Assert - But both verify against same password
	assert.NoError(t, hasher.Compare(hash1, password))
	assert.NoError(t, hasher.Compare(hash2, password))
}

func TestPasswordHasher_EmptyPassword(t *testing.T) {
	// Arrange
	hasher := auth.NewPasswordHasher(10)

	// Act
	hash, err := hasher.Hash("")

	// Assert - Empty password should be rejected
	assert.Error(t, err)
	assert.Empty(t, hash)
	assert.Contains(t, err.Error(), "empty")
}

func TestPasswordHasher_LongPassword(t *testing.T) {
	// Arrange
	hasher := auth.NewPasswordHasher(10)
	// bcrypt has a 72-byte limit, test that it properly handles this
	longPassword := string(make([]byte, 80)) // 80 bytes - exceeds bcrypt limit

	// Act
	hash, err := hasher.Hash(longPassword)

	// Assert - Should return error for password over 72 bytes
	assert.Error(t, err)
	assert.Empty(t, hash)
	assert.Contains(t, err.Error(), "exceeds 72 bytes")
}

func TestPasswordHasher_SpecialCharacters(t *testing.T) {
	// Arrange
	hasher := auth.NewPasswordHasher(10)
	password := "p@$$w0rd!#%^&*()+{}[]|\\:;<>?,./~`"

	// Act
	hash, err := hasher.Hash(password)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify comparison works with special characters
	err = hasher.Compare(hash, password)
	assert.NoError(t, err)
}

func TestPasswordHasher_CompareInvalidHash(t *testing.T) {
	// Arrange
	hasher := auth.NewPasswordHasher(10)
	password := "testPassword"
	invalidHash := "not-a-valid-bcrypt-hash"

	// Act
	err := hasher.Compare(invalidHash, password)

	// Assert
	assert.Error(t, err)
}
