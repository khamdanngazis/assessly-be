package postgres

import (
	"context"
	"fmt"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestRepository implements domain.TestRepository using PostgreSQL
type TestRepository struct {
	pool *pgxpool.Pool
}

// NewTestRepository creates a new PostgreSQL Test repository
func NewTestRepository(pool *pgxpool.Pool) *TestRepository {
	return &TestRepository{pool: pool}
}

// Create creates a new test in the database
func (r *TestRepository) Create(ctx context.Context, test *domain.Test) error {
	query := `
		INSERT INTO tests (id, creator_id, title, description, allow_retakes, is_published, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		test.ID,
		test.CreatorID,
		test.Title,
		test.Description,
		test.AllowRetakes,
		test.IsPublished,
		test.CreatedAt,
		test.UpdatedAt,
	)

	if err != nil {
		return domain.ErrInternal{
			Message: "failed to create test",
			Err:     err,
		}
	}

	return nil
}

// FindByID finds a test by ID
func (r *TestRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
	query := `
		SELECT id, creator_id, title, description, allow_retakes, is_published, created_at, updated_at
		FROM tests
		WHERE id = $1
	`

	var test domain.Test
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&test.ID,
		&test.CreatorID,
		&test.Title,
		&test.Description,
		&test.AllowRetakes,
		&test.IsPublished,
		&test.CreatedAt,
		&test.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound{
				Resource: "test",
				ID:       id.String(),
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find test by ID",
			Err:     err,
		}
	}

	return &test, nil
}

// FindByCreatorID finds tests by creator ID with pagination
func (r *TestRepository) FindByCreatorID(ctx context.Context, creatorID uuid.UUID, limit, offset int) ([]*domain.Test, error) {
	query := `
		SELECT id, creator_id, title, description, allow_retakes, is_published, created_at, updated_at
		FROM tests
		WHERE creator_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, creatorID, limit, offset)
	if err != nil {
		return nil, domain.ErrInternal{
			Message: "failed to find tests by creator ID",
			Err:     err,
		}
	}
	defer rows.Close()

	var tests []*domain.Test
	for rows.Next() {
		var test domain.Test
		err := rows.Scan(
			&test.ID,
			&test.CreatorID,
			&test.Title,
			&test.Description,
			&test.AllowRetakes,
			&test.IsPublished,
			&test.CreatedAt,
			&test.UpdatedAt,
		)
		if err != nil {
			return nil, domain.ErrInternal{
				Message: "failed to scan test row",
				Err:     err,
			}
		}
		tests = append(tests, &test)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal{
			Message: "error iterating test rows",
			Err:     err,
		}
	}

	return tests, nil
}

// FindPublished finds published tests with pagination
func (r *TestRepository) FindPublished(ctx context.Context, limit, offset int) ([]*domain.Test, error) {
	query := `
		SELECT id, creator_id, title, description, allow_retakes, is_published, created_at, updated_at
		FROM tests
		WHERE is_published = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, domain.ErrInternal{
			Message: "failed to find published tests",
			Err:     err,
		}
	}
	defer rows.Close()

	var tests []*domain.Test
	for rows.Next() {
		var test domain.Test
		err := rows.Scan(
			&test.ID,
			&test.CreatorID,
			&test.Title,
			&test.Description,
			&test.AllowRetakes,
			&test.IsPublished,
			&test.CreatedAt,
			&test.UpdatedAt,
		)
		if err != nil {
			return nil, domain.ErrInternal{
				Message: "failed to scan test row",
				Err:     err,
			}
		}
		tests = append(tests, &test)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal{
			Message: "error iterating test rows",
			Err:     err,
		}
	}

	return tests, nil
}

// Update updates an existing test
func (r *TestRepository) Update(ctx context.Context, test *domain.Test) error {
	query := `
		UPDATE tests
		SET title = $1, description = $2, allow_retakes = $3, is_published = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := r.pool.Exec(ctx, query,
		test.Title,
		test.Description,
		test.AllowRetakes,
		test.IsPublished,
		test.UpdatedAt,
		test.ID,
	)

	if err != nil {
		return domain.ErrInternal{
			Message: "failed to update test",
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "test",
			ID:       test.ID.String(),
		}
	}

	return nil
}

// Delete deletes a test (hard delete - be careful!)
func (r *TestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM tests
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return domain.ErrInternal{
			Message: fmt.Sprintf("failed to delete test: %v", err),
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "test",
			ID:       id.String(),
		}
	}

	return nil
}
