package domain

import "time"

// User is an authenticated platform account.
type User struct {
	ID              string
	Email           string
	PasswordHash    string
	FullName        string
	Bio             string
	AvatarURL       string
	Phone           string
	Locale          string
	Timezone        string
	RoleID          string
	RoleCode        RoleCode
	EmailVerifiedAt *time.Time
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// UserListFilter controls admin user listing.
type UserListFilter struct {
	Role    RoleCode
	Query   string
	Page    int
	PerPage int
}

// UserProfileUpdate is a self-service profile patch.
type UserProfileUpdate struct {
	FullName *string
	Bio      *string
	Phone    *string
	Locale   *string
	Timezone *string
}

// AdminUserUpdate is an admin-managed user patch.
type AdminUserUpdate struct {
	FullName *string
	RoleCode *RoleCode
	IsActive *bool
}

// RefreshToken is a persisted refresh session (hash only in storage).
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// PasswordResetToken is a one-time password reset credential.
type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// EmailVerificationToken is a one-time email verification credential.
type EmailVerificationToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
