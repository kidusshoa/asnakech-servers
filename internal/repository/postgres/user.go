package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const q = `
		INSERT INTO users (email, password_hash, full_name, role_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, created_at, updated_at, is_active`

	err := r.pool.QueryRow(ctx, q,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.PasswordHash,
		user.FullName,
		user.RoleID,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.IsActive)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("email already registered")
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `
		SELECT u.id::text, u.email, u.password_hash, u.full_name, u.role_id::text,
		       r.code, u.email_verified_at, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE lower(u.email) = lower($1) AND u.deleted_at IS NULL`

	user, err := scanUser(r.pool.QueryRow(ctx, q, strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("user not found")
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	const q = `
		SELECT u.id::text, u.email, u.password_hash, u.full_name, u.role_id::text,
		       r.code, u.email_verified_at, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL`

	user, err := scanUser(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) MarkEmailVerified(ctx context.Context, userID string, at time.Time) error {
	const q = `
		UPDATE users
		SET email_verified_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, userID, at)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("user not found")
	}
	return nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error {
	const q = `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("user not found")
	}
	return nil
}

func scanUser(row scannable) (*domain.User, error) {
	var user domain.User
	var roleCode string
	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.RoleID,
		&roleCode,
		&user.EmailVerifiedAt,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	user.RoleCode = domain.RoleCode(roleCode)
	return &user, nil
}
