package service

import (
	"context"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/i18n"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type DiscoveryService struct {
	search repository.SearchRepository
	cfg    *config.Config
}

func NewDiscoveryService(search repository.SearchRepository, cfg *config.Config) *DiscoveryService {
	return &DiscoveryService{search: search, cfg: cfg}
}

func (s *DiscoveryService) Search(ctx context.Context, filter domain.SearchFilter) (*domain.SearchResults, error) {
	return s.search.Search(ctx, filter)
}

func (s *DiscoveryService) Recommendations(ctx context.Context, userID, locale string, limit int) ([]domain.CourseRecommendation, error) {
	if userID == "" {
		return nil, apperr.Unauthorized("authentication required")
	}
	return s.search.Recommendations(ctx, userID, locale, limit)
}

func (s *DiscoveryService) FeatureFlags() map[string]bool {
	flags := map[string]bool{
		"payments":      true,
		"live":          true,
		"certificates":  true,
		"communication": true,
		"search":        true,
		"parent_links":  true,
	}
	for _, name := range s.cfg.FeatureFlags {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "!") || strings.HasPrefix(name, "-") {
			flags[strings.TrimPrefix(strings.TrimPrefix(name, "!"), "-")] = false
		} else {
			flags[name] = true
		}
	}
	return flags
}

func (s *DiscoveryService) Locales() []i18n.LocaleInfo {
	return i18n.Supported
}

func (s *DiscoveryService) Messages(locale string) map[string]string {
	return i18n.Messages(locale)
}

type ParentService struct {
	users repository.UserRepository
	links repository.ParentLinkRepository
}

func NewParentService(
	users repository.UserRepository,
	links repository.ParentLinkRepository,
) *ParentService {
	return &ParentService{users: users, links: links}
}

func (s *ParentService) LinkChild(ctx context.Context, parentID, studentEmail string) (*domain.ParentStudentLink, error) {
	studentEmail = strings.TrimSpace(strings.ToLower(studentEmail))
	if studentEmail == "" {
		return nil, apperr.Validation("student_email is required")
	}

	parent, err := s.users.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parent.RoleCode != domain.RoleParent && parent.RoleCode != domain.RoleAdmin {
		return nil, apperr.Forbidden("only parents or admins can link students")
	}

	student, err := s.users.GetByEmail(ctx, studentEmail)
	if err != nil {
		return nil, err
	}
	if student.RoleCode != domain.RoleStudent {
		return nil, apperr.Validation("linked user must be a student")
	}
	if student.ID == parentID {
		return nil, apperr.Validation("cannot link yourself")
	}

	link := &domain.ParentStudentLink{
		ParentUserID:  parentID,
		StudentUserID: student.ID,
	}
	if err := s.links.Create(ctx, link); err != nil {
		return nil, err
	}
	link.StudentEmail = student.Email
	link.StudentFullName = student.FullName
	return link, nil
}

func (s *ParentService) ListChildren(ctx context.Context, parentID string) ([]domain.ParentStudentLink, error) {
	return s.links.ListByParent(ctx, parentID)
}

func (s *ParentService) UnlinkChild(ctx context.Context, parentID, studentID string) error {
	return s.links.Revoke(ctx, parentID, studentID)
}
