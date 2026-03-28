package postgres

import (
	"context"
	"fmt"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implements domain.UserRepository using PostgreSQL
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new PostgreSQL User repository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if isPgUniqueViolation(err) {
			return domain.ErrConflict{
				Resource: "user",
				Message:  fmt.Sprintf("user with email %s already exists", user.Email),
			}
		}
		return domain.ErrInternal{
			Message: "failed to create user",
			Err:     err,
		}
	}

	return nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound{
				Resource: "user",
				ID:       id.String(),
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find user by ID",
			Err:     err,
		}
	}

	return &user, nil
}

// FindByEmail finds a user by email address
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound{
				Resource: "user",
				ID:       email,
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find user by email",
			Err:     err,
		}
	}

	return &user, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, password_hash = $2, role = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := r.pool.Exec(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		if isPgUniqueViolation(err) {
			return domain.ErrConflict{
				Resource: "user",
				Message:  fmt.Sprintf("user with email %s already exists", user.Email),
			}
		}
		return domain.ErrInternal{
			Message: "failed to update user",
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "user",
			ID:       user.ID.String(),
		}
	}

	return nil
}
