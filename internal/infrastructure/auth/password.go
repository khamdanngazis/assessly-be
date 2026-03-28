package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher handles password hashing and comparison
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new password hasher
// cost is the bcrypt cost (default: 12, valid range: 4-31)
func NewPasswordHasher(cost int) *PasswordHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &PasswordHasher{cost: cost}
}

// Hash hashes a plaintext password using bcrypt
func (h *PasswordHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// Compare compares a plaintext password with a hashed password
// Returns nil if they match, error otherwise
func (h *PasswordHasher) Compare(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("invalid password")
		}
		return fmt.Errorf("failed to compare passwords: %w", err)
	}
	return nil
}

// NeedsRehash checks if a password hash needs to be updated
// Returns true if the hash was created with a different cost
func (h *PasswordHasher) NeedsRehash(hashedPassword string) bool {
	cost, err := bcrypt.Cost([]byte(hashedPassword))
	if err != nil {
		return false
	}
	return cost != h.cost
}
