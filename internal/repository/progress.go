package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type LessonProgressRepository interface {
	Upsert(ctx context.Context, p *domain.LessonProgress) error
	GetByUserLesson(ctx context.Context, userID, lessonID string) (*domain.LessonProgress, error)
	ListByUserCourse(ctx context.Context, userID, courseID string) ([]domain.LessonProgress, error)
	CountCompletedPublished(ctx context.Context, userID, courseID string) (int, error)
}

type CourseProgressRepository interface {
	Upsert(ctx context.Context, p *domain.CourseProgress) error
	GetByUserCourse(ctx context.Context, userID, courseID string) (*domain.CourseProgress, error)
	ListByUser(ctx context.Context, userID string) ([]domain.CourseProgress, error)
}
