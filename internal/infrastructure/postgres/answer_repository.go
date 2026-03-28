package postgres

import (
	"context"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnswerRepository implements domain.AnswerRepository using PostgreSQL
type AnswerRepository struct {
	pool *pgxpool.Pool
}

// NewAnswerRepository creates a new PostgreSQL Answer repository
func NewAnswerRepository(pool *pgxpool.Pool) *AnswerRepository {
	return &AnswerRepository{pool: pool}
}

// Create creates a new answer in the database
func (r *AnswerRepository) Create(ctx context.Context, answer *domain.Answer) error {
	query := `
		INSERT INTO answers (id, submission_id, question_id, text, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.pool.Exec(ctx, query,
		answer.ID,
		answer.SubmissionID,
		answer.QuestionID,
		answer.Text,
		answer.CreatedAt,
	)

	if err != nil {
		// Check for unique constraint violation on (submission_id, question_id)
		if isPgUniqueViolation(err) {
			return domain.ErrConflict{
				Resource: "answer",
				Message:  "answer for this question already exists in this submission",
			}
		}
		return domain.ErrInternal{
			Message: "failed to create answer",
			Err:     err,
		}
	}

	return nil
}

// CreateBatch creates multiple answers in a transaction
func (r *AnswerRepository) CreateBatch(ctx context.Context, answers []*domain.Answer) error {
	// Start transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.ErrInternal{
			Message: "failed to begin transaction",
			Err:     err,
		}
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO answers (id, submission_id, question_id, text, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, answer := range answers {
		_, err := tx.Exec(ctx, query,
			answer.ID,
			answer.SubmissionID,
			answer.QuestionID,
			answer.Text,
			answer.CreatedAt,
		)
		if err != nil {
			return domain.ErrInternal{
				Message: "failed to create answer in batch",
				Err:     err,
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return domain.ErrInternal{
			Message: "failed to commit transaction",
			Err:     err,
		}
	}

	return nil
}

// FindByID finds an answer by ID
func (r *AnswerRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
	query := `
		SELECT id, submission_id, question_id, text, created_at
		FROM answers
		WHERE id = $1
	`

	var answer domain.Answer
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&answer.ID,
		&answer.SubmissionID,
		&answer.QuestionID,
		&answer.Text,
		&answer.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound{
				Resource: "answer",
				ID:       id.String(),
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find answer by ID",
			Err:     err,
		}
	}

	return &answer, nil
}

// FindBySubmissionID finds all answers for a submission
func (r *AnswerRepository) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error) {
	query := `
		SELECT id, submission_id, question_id, text, created_at
		FROM answers
		WHERE submission_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, submissionID)
	if err != nil {
		return nil, domain.ErrInternal{
			Message: "failed to find answers by submission ID",
			Err:     err,
		}
	}
	defer rows.Close()

	var answers []*domain.Answer
	for rows.Next() {
		var answer domain.Answer
		err := rows.Scan(
			&answer.ID,
			&answer.SubmissionID,
			&answer.QuestionID,
			&answer.Text,
			&answer.CreatedAt,
		)
		if err != nil {
			return nil, domain.ErrInternal{
				Message: "failed to scan answer row",
				Err:     err,
			}
		}
		answers = append(answers, &answer)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal{
			Message: "error iterating answer rows",
			Err:     err,
		}
	}

	return answers, nil
}
