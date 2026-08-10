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

const userSelectCols = `
	u.id::text, u.email, u.password_hash, u.full_name, u.bio, u.avatar_url, u.phone,
	u.locale, u.timezone, u.role_id::text, r.code, u.email_verified_at, u.is_active,
	u.created_at, u.updated_at`

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	const q = `
		INSERT INTO users (email, password_hash, full_name, role_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, created_at, updated_at, is_active, bio, avatar_url, phone, locale, timezone`

	err := r.pool.QueryRow(ctx, q,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.PasswordHash,
		user.FullName,
		user.RoleID,
	).Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.IsActive,
		&user.Bio, &user.AvatarURL, &user.Phone, &user.Locale, &user.Timezone,
	)
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
	q := fmt.Sprintf(`
		SELECT %s
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE lower(u.email) = lower($1) AND u.deleted_at IS NULL`, userSelectCols)

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
	q := fmt.Sprintf(`
		SELECT %s
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL`, userSelectCols)

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

func (r *UserRepository) UpdateProfile(ctx context.Context, userID string, patch domain.UserProfileUpdate) (*domain.User, error) {
	current, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	fullName := current.FullName
	bio := current.Bio
	phone := current.Phone
	locale := current.Locale
	timezone := current.Timezone
	if patch.FullName != nil {
		fullName = strings.TrimSpace(*patch.FullName)
	}
	if patch.Bio != nil {
		bio = strings.TrimSpace(*patch.Bio)
	}
	if patch.Phone != nil {
		phone = strings.TrimSpace(*patch.Phone)
	}
	if patch.Locale != nil {
		locale = strings.TrimSpace(*patch.Locale)
		if locale == "" {
			locale = "en"
		}
	}
	if patch.Timezone != nil {
		timezone = strings.TrimSpace(*patch.Timezone)
		if timezone == "" {
			timezone = "UTC"
		}
	}

	const q = `
		UPDATE users
		SET full_name = $2, bio = $3, phone = $4, locale = $5, timezone = $6
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, userID, fullName, bio, phone, locale, timezone)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("user not found")
	}
	return r.GetByID(ctx, userID)
}

func (r *UserRepository) UpdateAvatarURL(ctx context.Context, userID, avatarURL string) (*domain.User, error) {
	const q = `
		UPDATE users
		SET avatar_url = $2
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, userID, strings.TrimSpace(avatarURL))
	if err != nil {
		return nil, fmt.Errorf("update avatar: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("user not found")
	}
	return r.GetByID(ctx, userID)
}

func (r *UserRepository) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	where := []string{"u.deleted_at IS NULL"}
	args := make([]any, 0, 4)
	argN := 1

	if filter.Role != "" {
		where = append(where, fmt.Sprintf("r.code = $%d", argN))
		args = append(args, string(filter.Role))
		argN++
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		where = append(where, fmt.Sprintf("(u.email ILIKE $%d OR u.full_name ILIKE $%d)", argN, argN))
		args = append(args, "%"+q+"%")
		argN++
	}
	whereSQL := strings.Join(where, " AND ")

	countQ := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE %s`, whereSQL)

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	listArgs := append(append([]any{}, args...), perPage, offset)
	listQ := fmt.Sprintf(`
		SELECT %s
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE %s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d`, userSelectCols, whereSQL, argN, argN+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, perPage)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list users iterate: %w", err)
	}
	return users, total, nil
}

func (r *UserRepository) AdminUpdate(ctx context.Context, userID string, patch domain.AdminUserUpdate, roleID *string) (*domain.User, error) {
	current, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	fullName := current.FullName
	isActive := current.IsActive
	rid := current.RoleID
	if patch.FullName != nil {
		fullName = strings.TrimSpace(*patch.FullName)
	}
	if patch.IsActive != nil {
		isActive = *patch.IsActive
	}
	if roleID != nil {
		rid = *roleID
	}

	const q = `
		UPDATE users
		SET full_name = $2, is_active = $3, role_id = $4
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, userID, fullName, isActive, rid)
	if err != nil {
		return nil, fmt.Errorf("admin update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("user not found")
	}
	return r.GetByID(ctx, userID)
}

func (r *UserRepository) SoftDelete(ctx context.Context, userID string, at time.Time) error {
	const q = `
		UPDATE users
		SET deleted_at = $2, is_active = FALSE
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, userID, at)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
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
		&user.Bio,
		&user.AvatarURL,
		&user.Phone,
		&user.Locale,
		&user.Timezone,
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
