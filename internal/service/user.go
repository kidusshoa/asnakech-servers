package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type UserService struct {
	users repository.UserRepository
	roles repository.RoleRepository
}

func NewUserService(users repository.UserRepository, roles repository.RoleRepository) *UserService {
	return &UserService{users: users, roles: roles}
}

type AvatarUploadIntent struct {
	Method    string `json:"method"`
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
	Note      string `json:"note"`
}

func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, patch domain.UserProfileUpdate) (*domain.User, error) {
	if patch.Bio != nil && len(*patch.Bio) > 2000 {
		return nil, apperr.Validation("bio must be at most 2000 characters")
	}
	return s.users.UpdateProfile(ctx, userID, patch)
}

func (s *UserService) SetAvatarURL(ctx context.Context, userID, avatarURL string) (*domain.User, error) {
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return nil, apperr.Validation("avatar_url is required")
	}
	if _, err := url.ParseRequestURI(avatarURL); err != nil {
		return nil, apperr.Validation("avatar_url must be a valid URL")
	}
	return s.users.UpdateAvatarURL(ctx, userID, avatarURL)
}

// AvatarUploadIntent returns a placeholder for Stage 14 media uploads.
func (s *UserService) AvatarUploadIntent(userID string) AvatarUploadIntent {
	return AvatarUploadIntent{
		Method:    "PUT",
		UploadURL: "",
		PublicURL: "",
		Note:      "presigned uploads land in Stage 14; use PUT /users/me/avatar with avatar_url for now",
	}
}

func (s *UserService) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int64, domain.UserListFilter, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}
	if filter.Role != "" {
		switch filter.Role {
		case domain.RoleStudent, domain.RoleTeacher, domain.RoleAdmin, domain.RoleParent:
		default:
			return nil, 0, filter, apperr.Validation("invalid role filter")
		}
	}
	users, total, err := s.users.List(ctx, filter)
	return users, total, filter, err
}

func (s *UserService) AdminUpdate(ctx context.Context, actorID, userID string, patch domain.AdminUserUpdate) (*domain.User, error) {
	var roleID *string
	if patch.RoleCode != nil {
		role, err := s.roles.GetByCode(ctx, *patch.RoleCode)
		if err != nil {
			return nil, err
		}
		roleID = &role.ID
	}

	updated, err := s.users.AdminUpdate(ctx, userID, patch, roleID)
	if err != nil {
		return nil, err
	}

	// If admin deactivated themselves or changed own role, still allow — client must re-auth.
	_ = actorID
	return updated, nil
}

func (s *UserService) SoftDelete(ctx context.Context, actorID, userID string) error {
	if actorID == userID {
		return apperr.Validation("cannot delete your own account via admin API")
	}
	return s.users.SoftDelete(ctx, userID, time.Now().UTC())
}
