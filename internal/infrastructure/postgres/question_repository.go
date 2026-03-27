package postgres

import (
	"context"
	"fmt"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuestionRepository implements domain.QuestionRepository using PostgreSQL
type QuestionRepository struct {
	pool *pgxpool.Pool
}

// NewQuestionRepository creates a new PostgreSQL Question repository
func NewQuestionRepository(pool *pgxpool.Pool) *QuestionRepository {
	return &QuestionRepository{pool: pool}
}

// Create creates a new question in the database
func (r *QuestionRepository) Create(ctx context.Context, question *domain.Question) error {
	query := `
		INSERT INTO questions (id, test_id, text, expected_answer, order_num, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pool.Exec(ctx, query,
		question.ID,
		question.TestID,
		question.Text,
		question.ExpectedAnswer,
		question.OrderNum,
		question.CreatedAt,
	)

	if err != nil {
		// Check for unique constraint violation on (test_id, order_num)
		if isPgUniqueViolation(err) {
			return domain.ErrConflict{
				Resource: "question",
				Message:  fmt.Sprintf("question with order number %d already exists for this test", question.OrderNum),
			}
		}
		return domain.ErrInternal{
			Message: "failed to create question",
			Err:     err,
		}
	}

	return nil
}

// FindByID finds a question by ID
func (r *QuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Question, error) {
	query := `
		SELECT id, test_id, text, expected_answer, order_num, created_at
		FROM questions
		WHERE id = $1
	`

	var question domain.Question
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&question.ID,
		&question.TestID,
		&question.Text,
		&question.ExpectedAnswer,
		&question.OrderNum,
		&question.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound{
				Resource: "question",
				ID:       id.String(),
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find question by ID",
			Err:     err,
		}
	}

	return &question, nil
}

// FindByTestID finds all questions for a test, ordered by order_num
func (r *QuestionRepository) FindByTestID(ctx context.Context, testID uuid.UUID) ([]*domain.Question, error) {
	query := `
		SELECT id, test_id, text, expected_answer, order_num, created_at
		FROM questions
		WHERE test_id = $1
		ORDER BY order_num ASC
	`

	rows, err := r.pool.Query(ctx, query, testID)
	if err != nil {
		return nil, domain.ErrInternal{
			Message: "failed to find questions by test ID",
			Err:     err,
		}
	}
	defer rows.Close()

	var questions []*domain.Question
	for rows.Next() {
		var question domain.Question
		err := rows.Scan(
			&question.ID,
			&question.TestID,
			&question.Text,
			&question.ExpectedAnswer,
			&question.OrderNum,
			&question.CreatedAt,
		)
		if err != nil {
			return nil, domain.ErrInternal{
				Message: "failed to scan question row",
				Err:     err,
			}
		}
		questions = append(questions, &question)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.ErrInternal{
			Message: "error iterating question rows",
			Err:     err,
		}
	}

	return questions, nil
}

// Update updates an existing question
func (r *QuestionRepository) Update(ctx context.Context, question *domain.Question) error {
	query := `
		UPDATE questions
		SET text = $1, expected_answer = $2, order_num = $3
		WHERE id = $4
	`

	result, err := r.pool.Exec(ctx, query,
		question.Text,
		question.ExpectedAnswer,
		question.OrderNum,
		question.ID,
	)

	if err != nil {
		// Check for unique constraint violation on (test_id, order_num)
		if isPgUniqueViolation(err) {
			return domain.ErrConflict{
				Resource: "question",
				Message:  fmt.Sprintf("question with order number %d already exists for this test", question.OrderNum),
			}
		}
		return domain.ErrInternal{
			Message: "failed to update question",
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "question",
			ID:       question.ID.String(),
		}
	}

	return nil
}

// Delete deletes a question (hard delete - be careful!)
func (r *QuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM questions
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return domain.ErrInternal{
			Message: fmt.Sprintf("failed to delete question: %v", err),
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "question",
			ID:       id.String(),
		}
	}

	return nil
}

// CountByTestID counts the number of questions for a test
func (r *QuestionRepository) CountByTestID(ctx context.Context, testID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM questions WHERE test_id = $1
	`

	var count int
	err := r.pool.QueryRow(ctx, query, testID).Scan(&count)
	if err != nil {
		return 0, domain.ErrInternal{
			Message: "failed to count questions",
			Err:     err,
		}
	}

	return count, nil
}
