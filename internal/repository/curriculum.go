package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type ModuleRepository interface {
	Create(ctx context.Context, m *domain.CourseModule) error
	GetByID(ctx context.Context, id string) (*domain.CourseModule, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.CourseModule, error)
	Update(ctx context.Context, id, title string) (*domain.CourseModule, error)
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, courseID string, orderedIDs []string) error
	NextPosition(ctx context.Context, courseID string) (int, error)
}

type LessonRepository interface {
	Create(ctx context.Context, l *domain.Lesson) error
	GetByID(ctx context.Context, id string) (*domain.Lesson, error)
	ListByModule(ctx context.Context, moduleID string) ([]domain.Lesson, error)
	Update(ctx context.Context, id string, title, summary string, minutes int, prereq *string) (*domain.Lesson, error)
	SetStatus(ctx context.Context, id string, status domain.LessonStatus) (*domain.Lesson, error)
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, moduleID string, orderedIDs []string) error
	NextPosition(ctx context.Context, moduleID string) (int, error)
	CountPublishedByCourse(ctx context.Context, courseID string) (int, error)
}

type ContentBlockRepository interface {
	Create(ctx context.Context, b *domain.ContentBlock) error
	GetByID(ctx context.Context, id string) (*domain.ContentBlock, error)
	ListByLesson(ctx context.Context, lessonID string) ([]domain.ContentBlock, error)
	Update(ctx context.Context, b *domain.ContentBlock) (*domain.ContentBlock, error)
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, lessonID string, orderedIDs []string) error
	NextPosition(ctx context.Context, lessonID string) (int, error)
}
