package domain

import "time"

// User is an authenticated platform account.
type User struct {
	ID              string
	Email           string
	PasswordHash    string
	FullName        string
	RoleID          string
	RoleCode        RoleCode
	EmailVerifiedAt *time.Time
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
