package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at`

	err := r.pool.QueryRow(ctx, q, token.UserID, token.TokenHash, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	const q = `
		SELECT id::text, user_id::text, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var token domain.RefreshToken
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.Unauthorized("invalid refresh token")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &token, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, hash string, at time.Time) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL`

	_, err := r.pool.Exec(ctx, q, hash, at)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string, at time.Time) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.pool.Exec(ctx, q, userID, at)
	if err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}

type PasswordResetTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordResetTokenRepository(pool *pgxpool.Pool) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{pool: pool}
}

func (r *PasswordResetTokenRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	const q = `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at`

	err := r.pool.QueryRow(ctx, q, token.UserID, token.TokenHash, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *PasswordResetTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error) {
	const q = `
		SELECT id::text, user_id::text, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1`

	var token domain.PasswordResetToken
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.Unauthorized("invalid or expired reset token")
		}
		return nil, fmt.Errorf("get password reset token: %w", err)
	}
	return &token, nil
}

func (r *PasswordResetTokenRepository) MarkUsed(ctx context.Context, hash string, at time.Time) error {
	const q = `
		UPDATE password_reset_tokens
		SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, hash, at)
	if err != nil {
		return fmt.Errorf("mark reset token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.Unauthorized("invalid or expired reset token")
	}
	return nil
}

type EmailVerificationTokenRepository struct {
	pool *pgxpool.Pool
}

func NewEmailVerificationTokenRepository(pool *pgxpool.Pool) *EmailVerificationTokenRepository {
	return &EmailVerificationTokenRepository{pool: pool}
}

func (r *EmailVerificationTokenRepository) Create(ctx context.Context, token *domain.EmailVerificationToken) error {
	const q = `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at`

	err := r.pool.QueryRow(ctx, q, token.UserID, token.TokenHash, token.ExpiresAt).
		Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create email verification token: %w", err)
	}
	return nil
}

func (r *EmailVerificationTokenRepository) GetByHash(ctx context.Context, hash string) (*domain.EmailVerificationToken, error) {
	const q = `
		SELECT id::text, user_id::text, token_hash, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token_hash = $1`

	var token domain.EmailVerificationToken
	err := r.pool.QueryRow(ctx, q, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.Unauthorized("invalid or expired verification token")
		}
		return nil, fmt.Errorf("get email verification token: %w", err)
	}
	return &token, nil
}

func (r *EmailVerificationTokenRepository) MarkUsed(ctx context.Context, hash string, at time.Time) error {
	const q = `
		UPDATE email_verification_tokens
		SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, hash, at)
	if err != nil {
		return fmt.Errorf("mark verification token used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.Unauthorized("invalid or expired verification token")
	}
	return nil
}
