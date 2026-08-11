package repository

import (
	"context"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
	GetByID(ctx context.Context, id string) (*domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	Create(ctx context.Context, cat *domain.Category) error
}

type TagRepository interface {
	GetOrCreateByNames(ctx context.Context, names []string) ([]domain.Tag, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.Tag, error)
	ReplaceCourseTags(ctx context.Context, courseID string, tagIDs []string) error
}

type CourseRepository interface {
	Create(ctx context.Context, course *domain.Course) error
	GetByID(ctx context.Context, id string) (*domain.Course, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Course, error)
	Update(ctx context.Context, id string, patch domain.CourseUpdate) (*domain.Course, error)
	SetStatus(ctx context.Context, id string, status domain.CourseStatus, publishedAt *time.Time) (*domain.Course, error)
	SoftDelete(ctx context.Context, id string, at time.Time) error
	List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int64, error)
}
