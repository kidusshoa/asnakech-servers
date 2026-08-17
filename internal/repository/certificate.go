package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type CertificateRepository interface {
	Create(ctx context.Context, c *domain.Certificate) error
	GetByID(ctx context.Context, id string) (*domain.Certificate, error)
	GetByVerificationCode(ctx context.Context, code string) (*domain.Certificate, error)
	GetByCourseUser(ctx context.Context, courseID, userID string) (*domain.Certificate, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Certificate, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.Certificate, error)
	Revoke(ctx context.Context, id string) (*domain.Certificate, error)
	UpdateStorage(ctx context.Context, id, storageKey, publicURL string) error
}
