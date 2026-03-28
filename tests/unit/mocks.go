package unit

import (
	"context"

	"github.com/assessly/assessly-be/internal/domain"
	"github.com/assessly/assessly-be/internal/usecase/scoring"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// ============================================================================
// Repository Mocks
// ============================================================================

// MockUserRepository mocks the UserRepository interface
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockTestRepository mocks the TestRepository interface
type MockTestRepository struct {
	mock.Mock
}

func (m *MockTestRepository) Create(ctx context.Context, test *domain.Test) error {
	args := m.Called(ctx, test)
	return args.Error(0)
}

func (m *MockTestRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Test, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Test), args.Error(1)
}

func (m *MockTestRepository) Update(ctx context.Context, test *domain.Test) error {
	args := m.Called(ctx, test)
	return args.Error(0)
}

func (m *MockTestRepository) FindByCreatorID(ctx context.Context, creatorID uuid.UUID, limit, offset int) ([]*domain.Test, error) {
	args := m.Called(ctx, creatorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Test), args.Error(1)
}

func (m *MockTestRepository) FindPublished(ctx context.Context, limit, offset int) ([]*domain.Test, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Test), args.Error(1)
}

func (m *MockTestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockQuestionRepository mocks the QuestionRepository interface
type MockQuestionRepository struct {
	mock.Mock
}

func (m *MockQuestionRepository) Create(ctx context.Context, question *domain.Question) error {
	args := m.Called(ctx, question)
	return args.Error(0)
}

func (m *MockQuestionRepository) FindByTestID(ctx context.Context, testID uuid.UUID) ([]*domain.Question, error) {
	args := m.Called(ctx, testID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Question), args.Error(1)
}

func (m *MockQuestionRepository) CountByTestID(ctx context.Context, testID uuid.UUID) (int, error) {
	args := m.Called(ctx, testID)
	return args.Int(0), args.Error(1)
}

func (m *MockQuestionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Question, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Question), args.Error(1)
}

func (m *MockQuestionRepository) Update(ctx context.Context, question *domain.Question) error {
	args := m.Called(ctx, question)
	return args.Error(0)
}

func (m *MockQuestionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockSubmissionRepository mocks the SubmissionRepository interface
type MockSubmissionRepository struct {
	mock.Mock
}

func (m *MockSubmissionRepository) Create(ctx context.Context, submission *domain.Submission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

func (m *MockSubmissionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Submission, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Submission), args.Error(1)
}

func (m *MockSubmissionRepository) FindByTestID(ctx context.Context, testID uuid.UUID, limit, offset int) ([]*domain.Submission, error) {
	args := m.Called(ctx, testID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Submission), args.Error(1)
}

func (m *MockSubmissionRepository) CountByTestAndEmail(ctx context.Context, testID uuid.UUID, email string) (int, error) {
	args := m.Called(ctx, testID, email)
	return args.Int(0), args.Error(1)
}

func (m *MockSubmissionRepository) Update(ctx context.Context, submission *domain.Submission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

// MockAnswerRepository mocks the AnswerRepository interface
type MockAnswerRepository struct {
	mock.Mock
}

func (m *MockAnswerRepository) Create(ctx context.Context, answer *domain.Answer) error {
	args := m.Called(ctx, answer)
	return args.Error(0)
}

func (m *MockAnswerRepository) CreateBatch(ctx context.Context, answers []*domain.Answer) error {
	args := m.Called(ctx, answers)
	return args.Error(0)
}

func (m *MockAnswerRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Answer), args.Error(1)
}

func (m *MockAnswerRepository) FindBySubmissionID(ctx context.Context, submissionID uuid.UUID) ([]*domain.Answer, error) {
	args := m.Called(ctx, submissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Answer), args.Error(1)
}

// MockReviewRepository mocks the ReviewRepository interface
type MockReviewRepository struct {
	mock.Mock
}

func (m *MockReviewRepository) Create(ctx context.Context, review *domain.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockReviewRepository) FindByAnswerID(ctx context.Context, answerID uuid.UUID) (*domain.Review, error) {
	args := m.Called(ctx, answerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Review), args.Error(1)
}

func (m *MockReviewRepository) Update(ctx context.Context, review *domain.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockReviewRepository) UpsertAIScore(ctx context.Context, answerID uuid.UUID, score float64, feedback string) error {
	args := m.Called(ctx, answerID, score, feedback)
	return args.Error(0)
}

func (m *MockReviewRepository) UpsertManualScore(ctx context.Context, answerID uuid.UUID, reviewerID uuid.UUID, score float64, feedback string) error {
	args := m.Called(ctx, answerID, reviewerID, score, feedback)
	return args.Error(0)
}

// ============================================================================
// Service Mocks
// ============================================================================

// MockPasswordHasher mocks the PasswordHasher interface
type MockPasswordHasher struct {
	mock.Mock
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordHasher) Compare(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

// MockTokenGenerator mocks the TokenGenerator interface
type MockTokenGenerator struct {
	mock.Mock
}

func (m *MockTokenGenerator) GenerateToken(userID uuid.UUID, email, role string) (string, error) {
	args := m.Called(userID, email, role)
	return args.String(0), args.Error(1)
}

// MockTokenValidator mocks the TokenValidator interface
type MockTokenValidator struct {
	mock.Mock
}

func (m *MockTokenValidator) ValidateToken(tokenString string) (testID string, email string, role string, err error) {
	args := m.Called(tokenString)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

// MockResetTokenGenerator mocks the ResetTokenGenerator interface
type MockResetTokenGenerator struct {
	mock.Mock
}

func (m *MockResetTokenGenerator) GenerateResetToken(userID string, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

// MockEmailSender mocks the EmailSender interface
type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendPasswordReset(to, token, resetURL string) error {
	args := m.Called(to, token, resetURL)
	return args.Error(0)
}

func (m *MockEmailSender) SendTestAccessToken(to, testTitle, accessToken, accessURL string) error {
	args := m.Called(to, testTitle, accessToken, accessURL)
	return args.Error(0)
}

// MockAccessTokenGenerator mocks the AccessTokenGenerator interface
type MockAccessTokenGenerator struct {
	mock.Mock
}

func (m *MockAccessTokenGenerator) GenerateAccessToken(testID uuid.UUID, email string, expiryHours int) (string, error) {
	args := m.Called(testID, email, expiryHours)
	return args.String(0), args.Error(1)
}

// MockScoringQueuer mocks the ScoringQueuer interface
type MockScoringQueuer struct {
	mock.Mock
}

func (m *MockScoringQueuer) Enqueue(ctx context.Context, submissionID uuid.UUID) error {
	args := m.Called(ctx, submissionID)
	return args.Error(0)
}

// MockQueueEnqueuer mocks the QueueEnqueuer interface
type MockQueueEnqueuer struct {
	mock.Mock
}

func (m *MockQueueEnqueuer) Enqueue(ctx context.Context, submissionID uuid.UUID) error {
	args := m.Called(ctx, submissionID)
	return args.Error(0)
}

// MockAIScorer mocks the AIScorer interface
type MockAIScorer struct {
	mock.Mock
}

func (m *MockAIScorer) ScoreAnswer(ctx context.Context, question, expectedAnswer, actualAnswer string) (*scoring.ScoreResult, error) {
	args := m.Called(ctx, question, expectedAnswer, actualAnswer)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scoring.ScoreResult), args.Error(1)
}
