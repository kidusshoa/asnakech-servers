package domain

import "time"

// OrgRole is a membership role inside an organization (school).
type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

// Organization is a school / tenant container.
type Organization struct {
	ID          string
	Name        string
	Slug        string
	Description string
	LogoURL     string
	CreatedBy   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// OrganizationMember links a user to an organization.
type OrganizationMember struct {
	ID             string
	OrganizationID string
	UserID         string
	OrgRole        OrgRole
	JoinedAt       time.Time
	CreatedAt      time.Time
	// Optional joined fields for list responses
	UserEmail    string
	UserFullName string
}

// OrganizationInvite is a pending email invite.
type OrganizationInvite struct {
	ID             string
	OrganizationID string
	Email          string
	OrgRole        OrgRole
	TokenHash      string
	InvitedBy      *string
	ExpiresAt      time.Time
	AcceptedAt     *time.Time
	RevokedAt      *time.Time
	CreatedAt      time.Time
}

// OrganizationUpdate patches mutable org fields.
type OrganizationUpdate struct {
	Name        *string
	Description *string
	LogoURL     *string
}

// CanManage reports whether the org role may manage members/settings.
func (r OrgRole) CanManage() bool {
	return r == OrgRoleOwner || r == OrgRoleAdmin
}
