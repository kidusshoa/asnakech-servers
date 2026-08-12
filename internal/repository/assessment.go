package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type QuizRepository interface {
	Create(ctx context.Context, q *domain.Quiz) error
	GetByID(ctx context.Context, id string) (*domain.Quiz, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.Quiz, error)
	Update(ctx context.Context, q *domain.Quiz) (*domain.Quiz, error)
	SetStatus(ctx context.Context, id string, status domain.QuizStatus) (*domain.Quiz, error)
	Delete(ctx context.Context, id string) error
}

type QuizQuestionRepository interface {
	Create(ctx context.Context, q *domain.QuizQuestion) error
	GetByID(ctx context.Context, id string) (*domain.QuizQuestion, error)
	ListByQuiz(ctx context.Context, quizID string) ([]domain.QuizQuestion, error)
	Update(ctx context.Context, q *domain.QuizQuestion) (*domain.QuizQuestion, error)
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, quizID string, orderedIDs []string) error
	NextPosition(ctx context.Context, quizID string) (int, error)
}

type QuizAttemptRepository interface {
	Create(ctx context.Context, a *domain.QuizAttempt) error
	GetByID(ctx context.Context, id string) (*domain.QuizAttempt, error)
	ListByQuizUser(ctx context.Context, quizID, userID string) ([]domain.QuizAttempt, error)
	CountByQuizUser(ctx context.Context, quizID, userID string) (int, error)
	Update(ctx context.Context, a *domain.QuizAttempt) error
	GetInProgress(ctx context.Context, quizID, userID string) (*domain.QuizAttempt, error)
	BestGradedByCourse(ctx context.Context, courseID string) ([]domain.QuizAttempt, error)
}

type QuizAttemptAnswerRepository interface {
	Upsert(ctx context.Context, a *domain.QuizAttemptAnswer) error
	ListByAttempt(ctx context.Context, attemptID string) ([]domain.QuizAttemptAnswer, error)
	ReplaceAll(ctx context.Context, attemptID string, answers []domain.QuizAttemptAnswer) error
}

type AssignmentRepository interface {
	Create(ctx context.Context, a *domain.Assignment) error
	GetByID(ctx context.Context, id string) (*domain.Assignment, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.Assignment, error)
	Update(ctx context.Context, a *domain.Assignment) (*domain.Assignment, error)
	SetStatus(ctx context.Context, id string, status domain.AssignmentStatus) (*domain.Assignment, error)
	Delete(ctx context.Context, id string) error
}

type AssignmentSubmissionRepository interface {
	Upsert(ctx context.Context, s *domain.AssignmentSubmission) error
	GetByAssignmentUser(ctx context.Context, assignmentID, userID string) (*domain.AssignmentSubmission, error)
	GetByID(ctx context.Context, id string) (*domain.AssignmentSubmission, error)
	ListByAssignment(ctx context.Context, assignmentID string) ([]domain.AssignmentSubmission, error)
	ListGradedByCourse(ctx context.Context, courseID string) ([]domain.AssignmentSubmission, error)
}
