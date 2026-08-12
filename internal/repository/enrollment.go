package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type EnrollmentRepository interface {
	Create(ctx context.Context, e *domain.Enrollment) error
	GetByCourseUser(ctx context.Context, courseID, userID string) (*domain.Enrollment, error)
	GetByID(ctx context.Context, id string) (*domain.Enrollment, error)
	UpdateStatus(ctx context.Context, e *domain.Enrollment) error
	CountByCourseStatus(ctx context.Context, courseID string, status domain.EnrollmentStatus) (int64, error)
	ListByUser(ctx context.Context, userID string, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int64, error)
	ListByCourse(ctx context.Context, courseID string, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int64, error)
	NextWaitlisted(ctx context.Context, courseID string) (*domain.Enrollment, error)
	AppendEvent(ctx context.Context, ev *domain.EnrollmentEvent) error
}

type EnrollmentInviteCodeRepository interface {
	Create(ctx context.Context, code *domain.EnrollmentInviteCode) error
	GetByID(ctx context.Context, id string) (*domain.EnrollmentInviteCode, error)
	GetByCourseCode(ctx context.Context, courseID, code string) (*domain.EnrollmentInviteCode, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.EnrollmentInviteCode, error)
	Revoke(ctx context.Context, id string) error
	IncrementUses(ctx context.Context, id string) error
}
