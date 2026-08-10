package repository

import (
	"context"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id string) (*domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	Update(ctx context.Context, id string, patch domain.OrganizationUpdate) (*domain.Organization, error)
	SoftDelete(ctx context.Context, id string, at time.Time) error
	ListForUser(ctx context.Context, userID string) ([]domain.Organization, error)
}

type OrganizationMemberRepository interface {
	Add(ctx context.Context, member *domain.OrganizationMember) error
	Get(ctx context.Context, orgID, userID string) (*domain.OrganizationMember, error)
	ListByOrg(ctx context.Context, orgID string) ([]domain.OrganizationMember, error)
	UpdateRole(ctx context.Context, orgID, userID string, role domain.OrgRole) error
	Remove(ctx context.Context, orgID, userID string) error
	CountOwners(ctx context.Context, orgID string) (int, error)
}

type OrganizationInviteRepository interface {
	Create(ctx context.Context, invite *domain.OrganizationInvite) error
	GetByHash(ctx context.Context, hash string) (*domain.OrganizationInvite, error)
	GetByID(ctx context.Context, id string) (*domain.OrganizationInvite, error)
	ListPendingByOrg(ctx context.Context, orgID string) ([]domain.OrganizationInvite, error)
	MarkAccepted(ctx context.Context, id string, at time.Time) error
	Revoke(ctx context.Context, id string, at time.Time) error
}
