package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type SearchRepository interface {
	Search(ctx context.Context, filter domain.SearchFilter) (*domain.SearchResults, error)
	Recommendations(ctx context.Context, userID, locale string, limit int) ([]domain.CourseRecommendation, error)
}

type ParentLinkRepository interface {
	Create(ctx context.Context, link *domain.ParentStudentLink) error
	ListByParent(ctx context.Context, parentID string) ([]domain.ParentStudentLink, error)
	Revoke(ctx context.Context, parentID, studentID string) error
	GetActive(ctx context.Context, parentID, studentID string) (*domain.ParentStudentLink, error)
}
