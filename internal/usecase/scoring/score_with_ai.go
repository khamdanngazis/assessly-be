package scoring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/infrastructure/metrics"
	"github.com/google/uuid"
)

// AIScorer interface for AI scoring
type AIScorer interface {
	ScoreAnswer(ctx context.Context, question, expectedAnswer, actualAnswer string) (*ScoreResult, error)
}

// ScoreResult represents AI scoring result
type ScoreResult struct {
	Score    float64
	Feedback string
}

// ScoreWithAIRequest holds the data for AI scoring
type ScoreWithAIRequest struct {
	SubmissionID uuid.UUID
}

// ScoreWithAIUseCase handles AI scoring of submissions
type ScoreWithAIUseCase struct {
	submissionRepo domain.SubmissionRepository
	answerRepo     domain.AnswerRepository
	questionRepo   domain.QuestionRepository
	reviewRepo     domain.ReviewRepository
	aiScorer       AIScorer
	logger         *slog.Logger
}

// NewScoreWithAIUseCase creates a new ScoreWithAI use case
func NewScoreWithAIUseCase(
	submissionRepo domain.SubmissionRepository,
	answerRepo domain.AnswerRepository,
	questionRepo domain.QuestionRepository,
	reviewRepo domain.ReviewRepository,
	aiScorer AIScorer,
	logger *slog.Logger,
) *ScoreWithAIUseCase {
	return &ScoreWithAIUseCase{
		submissionRepo: submissionRepo,
		answerRepo:     answerRepo,
		questionRepo:   questionRepo,
		reviewRepo:     reviewRepo,
		aiScorer:       aiScorer,
		logger:         logger,
	}
}

// Execute scores all answers in a submission using AI
// T118: Records AI scoring duration metrics
func (uc *ScoreWithAIUseCase) Execute(ctx context.Context, req ScoreWithAIRequest) error {
	// T118: Start timing for metrics
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		metrics.AIScoringDuration.Observe(duration.Seconds())
	}()

	// Find submission
	submission, err := uc.submissionRepo.FindByID(ctx, req.SubmissionID)
	if err != nil {
		uc.logger.Error("failed to find submission", "submission_id", req.SubmissionID, "error", err)
		metrics.AIScoringTotal.WithLabelValues("error").Inc()
		metrics.AIScoringErrors.Inc()
		return err
	}

	// Get all answers for the submission
	answers, err := uc.answerRepo.FindBySubmissionID(ctx, req.SubmissionID)
	if err != nil {
		uc.logger.Error("failed to get answers", "submission_id", req.SubmissionID, "error", err)
		metrics.AIScoringTotal.WithLabelValues("error").Inc()
		metrics.AIScoringErrors.Inc()
		return err
	}

	if len(answers) == 0 {
		uc.logger.Warn("no answers found for submission", "submission_id", req.SubmissionID)
		metrics.AIScoringTotal.WithLabelValues("error").Inc()
		return fmt.Errorf("no answers found for submission %s", req.SubmissionID)
	}

	// Score each answer
	var totalScore float64
	var scoredCount int

	for _, answer := range answers {
		// Get the question for context
		question, err := uc.questionRepo.FindByID(ctx, answer.QuestionID)
		if err != nil {
			uc.logger.Error("failed to get question", "question_id", answer.QuestionID, "error", err)
			continue // Skip this answer but continue with others
		}

		// Score the answer using AI
		result, err := uc.aiScorer.ScoreAnswer(ctx, question.Text, question.ExpectedAnswer, answer.Text)
		if err != nil {
			uc.logger.Error("failed to score answer with AI",
				"answer_id", answer.ID,
				"question_id", question.ID,
				"error", err,
			)
			continue // Skip this answer but continue with others
		}

		// Store the AI score in review
		if err := uc.reviewRepo.UpsertAIScore(ctx, answer.ID, result.Score, result.Feedback); err != nil {
			uc.logger.Error("failed to store AI score",
				"answer_id", answer.ID,
				"score", result.Score,
				"error", err,
			)
			continue
		}

		totalScore += result.Score
		scoredCount++

		uc.logger.Info("answer scored with AI",
			"answer_id", answer.ID,
			"question_id", question.ID,
			"score", result.Score,
		)
	}

	// Calculate average score and update submission
	if scoredCount > 0 {
		avgScore := totalScore / float64(scoredCount)
		submission.AITotalScore = &avgScore

		if err := uc.submissionRepo.Update(ctx, submission); err != nil {
			uc.logger.Error("failed to update submission with AI score",
				"submission_id", req.SubmissionID,
				"score", avgScore,
				"error", err,
			)
			metrics.AIScoringTotal.WithLabelValues("error").Inc()
			metrics.AIScoringErrors.Inc()
			return err
		}

		// T118: Record successful scoring
		metrics.AIScoringTotal.WithLabelValues("success").Inc()

		uc.logger.Info("submission scored with AI",
			"submission_id", req.SubmissionID,
			"avg_score", avgScore,
			"scored_answers", scoredCount,
			"total_answers", len(answers),
		)
	} else {
		metrics.AIScoringTotal.WithLabelValues("error").Inc()
	}

	return nil
}
