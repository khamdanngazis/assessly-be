package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReviewRepository implements domain.ReviewRepository using PostgreSQL
type ReviewRepository struct {
	pool *pgxpool.Pool
}

// NewReviewRepository creates a new review repository
func NewReviewRepository(pool *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{pool: pool}
}

// Create inserts a new review into the database
func (r *ReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	query := `
		INSERT INTO reviews (
			id, answer_id, reviewer_id, ai_score, ai_feedback, ai_scored_at,
			manual_score, manual_feedback, manual_scored_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(ctx, query,
		review.ID,
		review.AnswerID,
		review.ReviewerID,
		review.AIScore,
		review.AIFeedback,
		review.AIScoredAt,
		review.ManualScore,
		review.ManualFeedback,
		review.ManualScoredAt,
	)

	if err != nil {
		return domain.ErrInternal{
			Message: "failed to create review",
			Err:     err,
		}
	}

	return nil
}

// FindByAnswerID retrieves a review by answer ID
func (r *ReviewRepository) FindByAnswerID(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
	query := `
		SELECT id, answer_id, reviewer_id, ai_score, ai_feedback, ai_scored_at,
		       manual_score, manual_feedback, manual_scored_at
		FROM reviews
		WHERE answer_id = $1
	`

	review := &domain.Review{}
	err := r.pool.QueryRow(ctx, query, answerID).Scan(
		&review.ID,
		&review.AnswerID,
		&review.ReviewerID,
		&review.AIScore,
		&review.AIFeedback,
		&review.AIScoredAt,
		&review.ManualScore,
		&review.ManualFeedback,
		&review.ManualScoredAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound{
				Resource: "review",
				ID:       answerID.String(),
			}
		}
		return nil, domain.ErrInternal{
			Message: "failed to find review",
			Err:     err,
		}
	}

	return review, nil
}

// Update updates an existing review
func (r *ReviewRepository) Update(ctx context.Context, review *domain.Review) error {
	query := `
		UPDATE reviews
		SET reviewer_id = $2, ai_score = $3, ai_feedback = $4, ai_scored_at = $5,
		    manual_score = $6, manual_feedback = $7, manual_scored_at = $8
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query,
		review.ID,
		review.ReviewerID,
		review.AIScore,
		review.AIFeedback,
		review.AIScoredAt,
		review.ManualScore,
		review.ManualFeedback,
		review.ManualScoredAt,
	)

	if err != nil {
		return domain.ErrInternal{
			Message: "failed to update review",
			Err:     err,
		}
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound{
			Resource: "review",
			ID:       "",
		}
	}

	return nil
}

// UpsertAIScore creates or updates AI score for an answer
func (r *ReviewRepository) UpsertAIScore(ctx context.Context, answerID uuid.UUID, score float64, feedback string) error {
	now := time.Now()
	query := `
		INSERT INTO reviews (id, answer_id, ai_score, ai_feedback, ai_scored_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (answer_id)
		DO UPDATE SET
			ai_score = EXCLUDED.ai_score,
			ai_feedback = EXCLUDED.ai_feedback,
			ai_scored_at = EXCLUDED.ai_scored_at
	`

	_, err := r.pool.Exec(ctx, query, uuid.New(), answerID, score, feedback, now)
	if err != nil {
		return domain.ErrInternal{
			Message: "failed to upsert AI score",
			Err:     err,
		}
	}

	return nil
}

// UpsertManualScore creates or updates manual score for an answer
func (r *ReviewRepository) UpsertManualScore(ctx context.Context, answerID uuid.UUID, reviewerID uuid.UUID, score float64, feedback string) error {
	now := time.Now()
	query := `
		INSERT INTO reviews (id, answer_id, reviewer_id, manual_score, manual_feedback, manual_scored_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (answer_id)
		DO UPDATE SET
			reviewer_id = EXCLUDED.reviewer_id,
			manual_score = EXCLUDED.manual_score,
			manual_feedback = EXCLUDED.manual_feedback,
			manual_scored_at = EXCLUDED.manual_scored_at
	`

	_, err := r.pool.Exec(ctx, query, uuid.New(), answerID, reviewerID, score, feedback, now)
	if err != nil {
		return domain.ErrInternal{
			Message: "failed to upsert manual score",
			Err:     err,
		}
	}

	return nil
}
