package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type MediaRepository interface {
	Create(ctx context.Context, a *domain.MediaAsset) error
	GetByID(ctx context.Context, id string) (*domain.MediaAsset, error)
	Update(ctx context.Context, a *domain.MediaAsset) (*domain.MediaAsset, error)
	ListByOwner(ctx context.Context, ownerID string, page, perPage int) ([]domain.MediaAsset, int64, error)
	SoftDelete(ctx context.Context, id string) error
}
