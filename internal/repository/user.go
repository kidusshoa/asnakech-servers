package repository

import (
	"context"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	MarkEmailVerified(ctx context.Context, userID string, at time.Time) error
	UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, hash string, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
}

type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, hash string, at time.Time) error
}

type EmailVerificationTokenRepository interface {
	Create(ctx context.Context, token *domain.EmailVerificationToken) error
	GetByHash(ctx context.Context, hash string) (*domain.EmailVerificationToken, error)
	MarkUsed(ctx context.Context, hash string, at time.Time) error
}
