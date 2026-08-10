package service

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/auth"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

const inviteTTL = 7 * 24 * time.Hour

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

type OrganizationService struct {
	orgs     repository.OrganizationRepository
	members  repository.OrganizationMemberRepository
	invites  repository.OrganizationInviteRepository
	users    repository.UserRepository
	exposeDev bool
}

func NewOrganizationService(
	orgs repository.OrganizationRepository,
	members repository.OrganizationMemberRepository,
	invites repository.OrganizationInviteRepository,
	users repository.UserRepository,
	exposeDev bool,
) *OrganizationService {
	return &OrganizationService{
		orgs:      orgs,
		members:   members,
		invites:   invites,
		users:     users,
		exposeDev: exposeDev,
	}
}

type CreateOrganizationInput struct {
	Name        string
	Slug        string
	Description string
}

type InviteResult struct {
	Invite     *domain.OrganizationInvite
	TokenDev   string
}

func (s *OrganizationService) Create(ctx context.Context, actorID string, in CreateOrganizationInput) (*domain.Organization, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperr.Validation("name is required")
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugify(name)
	} else {
		slug = slugify(slug)
	}
	if slug == "" {
		return nil, apperr.Validation("slug is required")
	}

	org := &domain.Organization{
		Name:        name,
		Slug:        slug,
		Description: strings.TrimSpace(in.Description),
		CreatedBy:   &actorID,
	}
	if err := s.orgs.Create(ctx, org); err != nil {
		return nil, err
	}

	member := &domain.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         actorID,
		OrgRole:        domain.OrgRoleOwner,
	}
	if err := s.members.Add(ctx, member); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) ListMine(ctx context.Context, userID string) ([]domain.Organization, error) {
	return s.orgs.ListForUser(ctx, userID)
}

func (s *OrganizationService) Get(ctx context.Context, actorID string, orgID string, platformAdmin bool) (*domain.Organization, *domain.OrganizationMember, error) {
	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	if platformAdmin {
		member, _ := s.members.Get(ctx, orgID, actorID)
		return org, member, nil
	}
	member, err := s.members.Get(ctx, orgID, actorID)
	if err != nil {
		return nil, nil, err
	}
	return org, member, nil
}

func (s *OrganizationService) Update(ctx context.Context, actorID, orgID string, patch domain.OrganizationUpdate, platformAdmin bool) (*domain.Organization, error) {
	if err := s.requireManage(ctx, actorID, orgID, platformAdmin); err != nil {
		return nil, err
	}
	if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
		return nil, apperr.Validation("name cannot be empty")
	}
	return s.orgs.Update(ctx, orgID, patch)
}

func (s *OrganizationService) Delete(ctx context.Context, actorID, orgID string, platformAdmin bool) error {
	if !platformAdmin {
		member, err := s.members.Get(ctx, orgID, actorID)
		if err != nil {
			return err
		}
		if member.OrgRole != domain.OrgRoleOwner {
			return apperr.Forbidden("only owners can delete an organization")
		}
	}
	return s.orgs.SoftDelete(ctx, orgID, time.Now().UTC())
}

func (s *OrganizationService) ListMembers(ctx context.Context, actorID, orgID string, platformAdmin bool) ([]domain.OrganizationMember, error) {
	if _, _, err := s.Get(ctx, actorID, orgID, platformAdmin); err != nil {
		return nil, err
	}
	return s.members.ListByOrg(ctx, orgID)
}

func (s *OrganizationService) UpdateMemberRole(ctx context.Context, actorID, orgID, targetUserID string, role domain.OrgRole, platformAdmin bool) error {
	if err := s.requireManage(ctx, actorID, orgID, platformAdmin); err != nil {
		return err
	}
	if role != domain.OrgRoleAdmin && role != domain.OrgRoleMember && role != domain.OrgRoleOwner {
		return apperr.Validation("invalid org role")
	}

	target, err := s.members.Get(ctx, orgID, targetUserID)
	if err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == apperr.CodeForbidden {
			return apperr.NotFound("member not found")
		}
		return err
	}

	if target.OrgRole == domain.OrgRoleOwner && role != domain.OrgRoleOwner {
		owners, err := s.members.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return apperr.Validation("cannot demote the last owner")
		}
	}
	return s.members.UpdateRole(ctx, orgID, targetUserID, role)
}

func (s *OrganizationService) RemoveMember(ctx context.Context, actorID, orgID, targetUserID string, platformAdmin bool) error {
	if err := s.requireManage(ctx, actorID, orgID, platformAdmin); err != nil {
		return err
	}
	if actorID == targetUserID {
		return apperr.Validation("use leave flow or transfer ownership before removing yourself")
	}

	target, err := s.members.Get(ctx, orgID, targetUserID)
	if err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == apperr.CodeForbidden {
			return apperr.NotFound("member not found")
		}
		return err
	}
	if target.OrgRole == domain.OrgRoleOwner {
		owners, err := s.members.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return apperr.Validation("cannot remove the last owner")
		}
	}
	return s.members.Remove(ctx, orgID, targetUserID)
}

func (s *OrganizationService) CreateInvite(ctx context.Context, actorID, orgID, email string, role domain.OrgRole, platformAdmin bool) (*InviteResult, error) {
	if err := s.requireManage(ctx, actorID, orgID, platformAdmin); err != nil {
		return nil, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, apperr.Validation("a valid email is required")
	}
	if role != domain.OrgRoleAdmin && role != domain.OrgRoleMember {
		return nil, apperr.Validation("invite role must be admin or member")
	}

	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		return nil, apperr.Internal("could not create invite token")
	}
	invite := &domain.OrganizationInvite{
		OrganizationID: orgID,
		Email:          email,
		OrgRole:        role,
		TokenHash:      hash,
		InvitedBy:      &actorID,
		ExpiresAt:      time.Now().UTC().Add(inviteTTL),
	}
	if err := s.invites.Create(ctx, invite); err != nil {
		return nil, err
	}
	result := &InviteResult{Invite: invite}
	if s.exposeDev {
		result.TokenDev = raw
	}
	return result, nil
}

func (s *OrganizationService) ListInvites(ctx context.Context, actorID, orgID string, platformAdmin bool) ([]domain.OrganizationInvite, error) {
	if err := s.requireManage(ctx, actorID, orgID, platformAdmin); err != nil {
		return nil, err
	}
	return s.invites.ListPendingByOrg(ctx, orgID)
}

func (s *OrganizationService) RevokeInvite(ctx context.Context, actorID, orgID, inviteID string, platformAdmin bool) error {
	if err := s.requireManage(ctx, actorID, orgID, platformAdmin); err != nil {
		return err
	}
	invite, err := s.invites.GetByID(ctx, inviteID)
	if err != nil {
		return err
	}
	if invite.OrganizationID != orgID {
		return apperr.NotFound("invite not found")
	}
	return s.invites.Revoke(ctx, inviteID, time.Now().UTC())
}

func (s *OrganizationService) AcceptInvite(ctx context.Context, actorID, rawToken string) (*domain.Organization, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, apperr.Validation("token is required")
	}
	invite, err := s.invites.GetByHash(ctx, auth.HashToken(rawToken))
	if err != nil {
		return nil, err
	}
	if invite.RevokedAt != nil || invite.AcceptedAt != nil || time.Now().UTC().After(invite.ExpiresAt) {
		return nil, apperr.Unauthorized("invalid or expired invite")
	}

	user, err := s.users.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(user.Email, invite.Email) {
		return nil, apperr.Forbidden("invite email does not match your account")
	}

	org, err := s.orgs.GetByID(ctx, invite.OrganizationID)
	if err != nil {
		return nil, err
	}

	member := &domain.OrganizationMember{
		OrganizationID: invite.OrganizationID,
		UserID:         actorID,
		OrgRole:        invite.OrgRole,
	}
	if err := s.members.Add(ctx, member); err != nil {
		if appErr, ok := apperr.As(err); ok && appErr.Code == apperr.CodeConflict {
			// already a member — still mark invite accepted
		} else {
			return nil, err
		}
	}
	if err := s.invites.MarkAccepted(ctx, invite.ID, time.Now().UTC()); err != nil {
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) requireManage(ctx context.Context, actorID, orgID string, platformAdmin bool) error {
	if platformAdmin {
		return nil
	}
	member, err := s.members.Get(ctx, orgID, actorID)
	if err != nil {
		return err
	}
	if !member.OrgRole.CanManage() {
		return apperr.Forbidden("insufficient organization role")
	}
	return nil
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == ' ' {
			b.WriteRune(r)
		}
	}
	s = nonSlug.ReplaceAllString(b.String(), "-")
	s = strings.Trim(s, "-")
	if len(s) > 64 {
		s = s[:64]
		s = strings.Trim(s, "-")
	}
	return s
}

// Slugify normalizes an organization name/slug for uniqueness.
func Slugify(s string) string { return slugify(s) }
