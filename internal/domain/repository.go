package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines methods for user persistence
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}

// TestRepository defines methods for test persistence
type TestRepository interface {
	Create(ctx context.Context, test *Test) error
	FindByID(ctx context.Context, id uuid.UUID) (*Test, error)
	FindByCreatorID(ctx context.Context, creatorID uuid.UUID, limit, offset int) ([]*Test, error)
	FindPublished(ctx context.Context, limit, offset int) ([]*Test, error)
	Update(ctx context.Context, test *Test) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// QuestionRepository defines methods for question persistence
type QuestionRepository interface {
	Create(ctx context.Context, question *Question) error
	FindByID(ctx context.Context, id uuid.UUID) (*Question, error)
	FindByTestID(ctx context.Context, testID uuid.UUID) ([]*Question, error)
	Update(ctx context.Context, question *Question) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByTestID(ctx context.Context, testID uuid.UUID) (int, error)
}

// SubmissionRepository defines methods for submission persistence
type SubmissionRepository interface {
	Create(ctx context.Context, submission *Submission) error
	FindByID(ctx context.Context, id uuid.UUID) (*Submission, error)
	FindByTestID(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*Submission, error)
	CountByTestAndEmail(ctx context.Context, testID uuid.UUID, email string) (int, error)
	Update(ctx context.Context, submission *Submission) error
}

// AnswerRepository defines methods for answer persistence
type AnswerRepository interface {
	Create(ctx context.Context, answer *Answer) error
	CreateBatch(ctx context.Context, answers []*Answer) error
	FindByID(ctx context.Context, id uuid.UUID) (*Answer, error)
	FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*Answer, error)
}

// ReviewRepository defines methods for review persistence
type ReviewRepository interface {
	Create(ctx context.Context, review *Review) error
	FindByAnswerID(ctx context.Context, answerID uuid.UUID) (*Review, error)
	Update(ctx context.Context, review *Review) error
	UpsertAIScore(ctx context.Context, answerID uuid.UUID, score float64, feedback string) error
	UpsertManualScore(ctx context.Context, answerID uuid.UUID, reviewerID uuid.UUID, score float64, feedback string) error
}
