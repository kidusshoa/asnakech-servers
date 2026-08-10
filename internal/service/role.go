package service

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

// RoleService exposes role use-cases.
type RoleService struct {
	roles repository.RoleRepository
}

func NewRoleService(roles repository.RoleRepository) *RoleService {
	return &RoleService{roles: roles}
}

func (s *RoleService) List(ctx context.Context) ([]domain.Role, error) {
	return s.roles.List(ctx)
}

func (s *RoleService) GetByCode(ctx context.Context, code domain.RoleCode) (*domain.Role, error) {
	return s.roles.GetByCode(ctx, code)
}
