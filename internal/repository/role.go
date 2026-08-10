package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

// RoleRepository loads platform roles from persistence.
type RoleRepository interface {
	List(ctx context.Context) ([]domain.Role, error)
	GetByCode(ctx context.Context, code domain.RoleCode) (*domain.Role, error)
}
