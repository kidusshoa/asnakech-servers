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

type EnrollmentRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) *EnrollmentRepository {
	return &EnrollmentRepository{pool: pool}
}

func (r *EnrollmentRepository) Create(ctx context.Context, e *domain.Enrollment) error {
	const q = `
		INSERT INTO enrollments (
			course_id, user_id, status, source, invite_code_id,
			enrolled_at, waitlisted_at, cancelled_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		e.CourseID, e.UserID, string(e.Status), string(e.Source), e.InviteCodeID,
		e.EnrolledAt, e.WaitlistedAt, e.CancelledAt,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("already enrolled in this course")
		}
		return fmt.Errorf("create enrollment: %w", err)
	}
	return nil
}

func (r *EnrollmentRepository) GetByID(ctx context.Context, id string) (*domain.Enrollment, error) {
	const q = `
		SELECT e.id::text, e.course_id::text, e.user_id::text, e.status, e.source,
		       e.invite_code_id::text, e.enrolled_at, e.waitlisted_at, e.cancelled_at,
		       e.created_at, e.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, '')
		FROM enrollments e
		LEFT JOIN users u ON u.id = e.user_id
		LEFT JOIN courses c ON c.id = e.course_id
		WHERE e.id = $1`

	en, err := scanEnrollment(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("enrollment not found")
		}
		return nil, fmt.Errorf("get enrollment: %w", err)
	}
	return en, nil
}

func (r *EnrollmentRepository) GetByCourseUser(ctx context.Context, courseID, userID string) (*domain.Enrollment, error) {
	const q = `
		SELECT e.id::text, e.course_id::text, e.user_id::text, e.status, e.source,
		       e.invite_code_id::text, e.enrolled_at, e.waitlisted_at, e.cancelled_at,
		       e.created_at, e.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, '')
		FROM enrollments e
		LEFT JOIN users u ON u.id = e.user_id
		LEFT JOIN courses c ON c.id = e.course_id
		WHERE e.course_id = $1 AND e.user_id = $2`

	en, err := scanEnrollment(r.pool.QueryRow(ctx, q, courseID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("enrollment not found")
		}
		return nil, fmt.Errorf("get enrollment by course user: %w", err)
	}
	return en, nil
}

func (r *EnrollmentRepository) UpdateStatus(ctx context.Context, e *domain.Enrollment) error {
	const q = `
		UPDATE enrollments
		SET status = $2, source = $3, invite_code_id = $4,
		    enrolled_at = $5, waitlisted_at = $6, cancelled_at = $7
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q,
		e.ID, string(e.Status), string(e.Source), e.InviteCodeID,
		e.EnrolledAt, e.WaitlistedAt, e.CancelledAt,
	)
	if err != nil {
		return fmt.Errorf("update enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("enrollment not found")
	}
	return nil
}

func (r *EnrollmentRepository) CountByCourseStatus(ctx context.Context, courseID string, status domain.EnrollmentStatus) (int64, error) {
	const q = `SELECT COUNT(*) FROM enrollments WHERE course_id = $1 AND status = $2`
	var n int64
	if err := r.pool.QueryRow(ctx, q, courseID, string(status)).Scan(&n); err != nil {
		return 0, fmt.Errorf("count enrollments: %w", err)
	}
	return n, nil
}

func (r *EnrollmentRepository) ListByUser(ctx context.Context, userID string, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int64, error) {
	return r.list(ctx, "e.user_id = $1", []any{userID}, filter)
}

func (r *EnrollmentRepository) ListByCourse(ctx context.Context, courseID string, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int64, error) {
	return r.list(ctx, "e.course_id = $1", []any{courseID}, filter)
}

func (r *EnrollmentRepository) list(ctx context.Context, baseWhere string, baseArgs []any, filter domain.EnrollmentListFilter) ([]domain.Enrollment, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	where := []string{baseWhere}
	args := append([]any{}, baseArgs...)
	n := len(args) + 1
	if filter.Status != "" {
		where = append(where, fmt.Sprintf("e.status = $%d", n))
		args = append(args, string(filter.Status))
		n++
	}
	whereSQL := strings.Join(where, " AND ")

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM enrollments e WHERE %s`, whereSQL)
	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count enrollments list: %w", err)
	}

	listArgs := append(append([]any{}, args...), perPage, offset)
	listQ := fmt.Sprintf(`
		SELECT e.id::text, e.course_id::text, e.user_id::text, e.status, e.source,
		       e.invite_code_id::text, e.enrolled_at, e.waitlisted_at, e.cancelled_at,
		       e.created_at, e.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, '')
		FROM enrollments e
		LEFT JOIN users u ON u.id = e.user_id
		LEFT JOIN courses c ON c.id = e.course_id
		WHERE %s
		ORDER BY e.created_at DESC
		LIMIT $%d OFFSET $%d`, whereSQL, n, n+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list enrollments: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Enrollment, 0, perPage)
	for rows.Next() {
		en, err := scanEnrollment(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *en)
	}
	return out, total, rows.Err()
}

func (r *EnrollmentRepository) NextWaitlisted(ctx context.Context, courseID string) (*domain.Enrollment, error) {
	const q = `
		SELECT e.id::text, e.course_id::text, e.user_id::text, e.status, e.source,
		       e.invite_code_id::text, e.enrolled_at, e.waitlisted_at, e.cancelled_at,
		       e.created_at, e.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, '')
		FROM enrollments e
		LEFT JOIN users u ON u.id = e.user_id
		LEFT JOIN courses c ON c.id = e.course_id
		WHERE e.course_id = $1 AND e.status = 'waitlisted'
		ORDER BY e.waitlisted_at ASC NULLS LAST, e.created_at ASC
		LIMIT 1`

	en, err := scanEnrollment(r.pool.QueryRow(ctx, q, courseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("no waitlisted enrollment")
		}
		return nil, fmt.Errorf("next waitlisted: %w", err)
	}
	return en, nil
}

func (r *EnrollmentRepository) AppendEvent(ctx context.Context, ev *domain.EnrollmentEvent) error {
	const q = `
		INSERT INTO enrollment_events (enrollment_id, course_id, user_id, event_type)
		VALUES ($1,$2,$3,$4)
		RETURNING id::text, created_at`

	if err := r.pool.QueryRow(ctx, q, ev.EnrollmentID, ev.CourseID, ev.UserID, string(ev.EventType)).
		Scan(&ev.ID, &ev.CreatedAt); err != nil {
		return fmt.Errorf("append enrollment event: %w", err)
	}
	return nil
}

func scanEnrollment(row scannable) (*domain.Enrollment, error) {
	var e domain.Enrollment
	var status, source string
	var inviteID *string
	if err := row.Scan(
		&e.ID, &e.CourseID, &e.UserID, &status, &source,
		&inviteID, &e.EnrolledAt, &e.WaitlistedAt, &e.CancelledAt,
		&e.CreatedAt, &e.UpdatedAt,
		&e.UserEmail, &e.UserFullName, &e.CourseTitle, &e.CourseSlug,
	); err != nil {
		return nil, err
	}
	e.Status = domain.EnrollmentStatus(status)
	e.Source = domain.EnrollmentSource(source)
	e.InviteCodeID = inviteID
	return &e, nil
}

type EnrollmentInviteCodeRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentInviteCodeRepository(pool *pgxpool.Pool) *EnrollmentInviteCodeRepository {
	return &EnrollmentInviteCodeRepository{pool: pool}
}

func (r *EnrollmentInviteCodeRepository) Create(ctx context.Context, code *domain.EnrollmentInviteCode) error {
	const q = `
		INSERT INTO enrollment_invite_codes (course_id, code, max_uses, expires_at, created_by)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id::text, uses_count, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, code.CourseID, code.Code, code.MaxUses, code.ExpiresAt, code.CreatedBy).
		Scan(&code.ID, &code.UsesCount, &code.CreatedAt, &code.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("invite code already exists for this course")
		}
		return fmt.Errorf("create invite code: %w", err)
	}
	return nil
}

func (r *EnrollmentInviteCodeRepository) GetByID(ctx context.Context, id string) (*domain.EnrollmentInviteCode, error) {
	const q = `
		SELECT id::text, course_id::text, code, max_uses, uses_count, expires_at,
		       created_by::text, revoked_at, created_at, updated_at
		FROM enrollment_invite_codes WHERE id = $1`

	c, err := scanInviteCode(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("invite code not found")
		}
		return nil, fmt.Errorf("get invite code: %w", err)
	}
	return c, nil
}

func (r *EnrollmentInviteCodeRepository) GetByCourseCode(ctx context.Context, courseID, code string) (*domain.EnrollmentInviteCode, error) {
	const q = `
		SELECT id::text, course_id::text, code, max_uses, uses_count, expires_at,
		       created_by::text, revoked_at, created_at, updated_at
		FROM enrollment_invite_codes
		WHERE course_id = $1 AND lower(code) = lower($2)`

	c, err := scanInviteCode(r.pool.QueryRow(ctx, q, courseID, strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("invite code not found")
		}
		return nil, fmt.Errorf("get invite code by course: %w", err)
	}
	return c, nil
}

func (r *EnrollmentInviteCodeRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.EnrollmentInviteCode, error) {
	const q = `
		SELECT id::text, course_id::text, code, max_uses, uses_count, expires_at,
		       created_by::text, revoked_at, created_at, updated_at
		FROM enrollment_invite_codes
		WHERE course_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list invite codes: %w", err)
	}
	defer rows.Close()

	out := make([]domain.EnrollmentInviteCode, 0)
	for rows.Next() {
		c, err := scanInviteCode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *EnrollmentInviteCodeRepository) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE enrollment_invite_codes SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("revoke invite code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("invite code not found")
	}
	return nil
}

func (r *EnrollmentInviteCodeRepository) IncrementUses(ctx context.Context, id string) error {
	const q = `UPDATE enrollment_invite_codes SET uses_count = uses_count + 1 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("increment invite uses: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("invite code not found")
	}
	return nil
}

func scanInviteCode(row scannable) (*domain.EnrollmentInviteCode, error) {
	var c domain.EnrollmentInviteCode
	if err := row.Scan(
		&c.ID, &c.CourseID, &c.Code, &c.MaxUses, &c.UsesCount, &c.ExpiresAt,
		&c.CreatedBy, &c.RevokedAt, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}
