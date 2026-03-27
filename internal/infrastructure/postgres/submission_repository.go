package postgres

import (
	"context"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SubmissionRepository implements domain.SubmissionRepository using PostgreSQL
type SubmissionRepository struct {
	pool *pgxpool.Pool
}

// NewSubmissionRepository creates a new PostgreSQL Submission repository
func NewSubmissionRepository(pool *pgxpool.Pool) *SubmissionRepository {
	return &SubmissionRepository{pool: pool}
}

// Create creates a new submission in the database
func (r *SubmissionRepository) Create(ctx context.Context, submission *domain.Submission) error {
	query := `
		INSERT INTO submissions (id, test_id, access_email, submitted_at, ai_total_score, manual_total_score)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		submission.ID,
		submission.TestID,
		submission.AccessEmail,
		submission.SubmittedAt,
		submission.AITotalScore,
		submission.ManualTotalScore,
	)

	if err != nil {
		return domain.ErrInternal{
			Message: "failed to create submission",
			Err:     err,
		}
	}

	return nil
}

// FindByID finds a submission by ID
func (r *SubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	query := `
		SELECT id, test_id, access_email, submitted_at, ai_total_score, manual_total_score
		FROM submissions
		WHERE id = $1
	`

	var submission domain.Submission
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&submission.ID,
		&submission.TestID,
		&submission.AccessEmail,
		&submission.SubmittedAt,
		&submission.AITotalScore,
		&submission.ManualTotalScore,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound{
				Resource: "submission",
				ID:       id.String(),
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find submission by ID",
			Err:     err,
		}
	}

	return &submission, nil
}

// FindByTestID finds submissions for a test with pagination
func (r *SubmissionRepository) FindByTestID(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error) {
	query := `
		SELECT id, test_id, access_email, submitted_at, ai_total_score, manual_total_score
		FROM submissions
		WHERE test_id = $1
		ORDER BY submitted_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, testID, limit, offset)
	if err != nil {
		return nil, domain.ErrInternal{
			Message: "failed to find submissions by test ID",
			Err:     err,
		}
	}
	defer rows.Close()

	var submissions []*domain.Submission
	for rows.Next() {
		var submission domain.Submission
		err := rows.Scan(
			&submission.ID,
			&submission.TestID,
			&submission.AccessEmail,
			&submission.SubmittedAt,
			&submission.AITotalScore,
			&submission.ManualTotalScore,
		)
		if err != nil {
			return nil, domain.ErrInternal{
				Message: "failed to scan submission row",
				Err:     err,
			}
		}
		submissions = append(submissions, &submission)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal{
			Message: "error iterating submission rows",
			Err:     err,
		}
	}

	return submissions, nil
}

// CountByTestAndEmail counts how many times an email has submitted for a test
func (r *SubmissionRepository) CountByTestAndEmail(ctx context.Context, testID uuid.UUID, email string) (int, error) {
	query := `
		SELECT COUNT(*) FROM submissions 
		WHERE test_id = $1 AND access_email = $2
	`

	var count int
	err := r.pool.QueryRow(ctx, query, testID, email).Scan(&count)
	if err != nil {
		return 0, domain.ErrInternal{
			Message: "failed to count submissions",
			Err:     err,
		}
	}

	return count, nil
}

// Update updates an existing submission
func (r *SubmissionRepository) Update(ctx context.Context, submission *domain.Submission) error {
	query := `
		UPDATE submissions
		SET ai_total_score = $1, manual_total_score = $2
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query,
		submission.AITotalScore,
		submission.ManualTotalScore,
		submission.ID,
	)

	if err != nil {
		return domain.ErrInternal{
			Message: "failed to update submission",
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "submission",
			ID:       submission.ID.String(),
		}
	}

	return nil
}
