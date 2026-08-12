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

type CourseRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) *CourseRepository {
	return &CourseRepository{pool: pool}
}

const courseSelect = `
	c.id::text, c.organization_id::text, c.teacher_id::text, c.category_id::text,
	c.title, c.slug, c.summary, c.description, c.status, c.cover_url,
	c.price_cents, c.currency, c.level, c.language, c.published_at,
	c.created_at, c.updated_at,
	c.enrollment_capacity, c.enrollment_open, c.waitlist_enabled,
	COALESCE(cat.name, ''), COALESCE(u.full_name, '')`

func (r *CourseRepository) Create(ctx context.Context, course *domain.Course) error {
	const q = `
		INSERT INTO courses (
			organization_id, teacher_id, category_id, title, slug, summary, description,
			status, cover_url, price_cents, currency, level, language
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		course.OrganizationID,
		course.TeacherID,
		course.CategoryID,
		course.Title,
		course.Slug,
		course.Summary,
		course.Description,
		string(course.Status),
		course.CoverURL,
		course.PriceCents,
		course.Currency,
		string(course.Level),
		course.Language,
	).Scan(&course.ID, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("course slug already exists")
		}
		return fmt.Errorf("create course: %w", err)
	}
	return nil
}

func (r *CourseRepository) GetByID(ctx context.Context, id string) (*domain.Course, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM courses c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.teacher_id
		WHERE c.id = $1 AND c.deleted_at IS NULL`, courseSelect)

	course, err := scanCourse(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("course not found")
		}
		return nil, fmt.Errorf("get course: %w", err)
	}
	return course, nil
}

func (r *CourseRepository) GetBySlug(ctx context.Context, slug string) (*domain.Course, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM courses c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.teacher_id
		WHERE lower(c.slug) = lower($1) AND c.deleted_at IS NULL`, courseSelect)

	course, err := scanCourse(r.pool.QueryRow(ctx, q, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("course not found")
		}
		return nil, fmt.Errorf("get course by slug: %w", err)
	}
	return course, nil
}

func (r *CourseRepository) Update(ctx context.Context, id string, patch domain.CourseUpdate) (*domain.Course, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	title := current.Title
	summary := current.Summary
	description := current.Description
	cover := current.CoverURL
	price := current.PriceCents
	currency := current.Currency
	level := current.Level
	language := current.Language
	categoryID := current.CategoryID

	if patch.Title != nil {
		title = strings.TrimSpace(*patch.Title)
	}
	if patch.Summary != nil {
		summary = strings.TrimSpace(*patch.Summary)
	}
	if patch.Description != nil {
		description = strings.TrimSpace(*patch.Description)
	}
	if patch.CoverURL != nil {
		cover = strings.TrimSpace(*patch.CoverURL)
	}
	if patch.PriceCents != nil {
		price = *patch.PriceCents
	}
	if patch.Currency != nil {
		currency = strings.ToUpper(strings.TrimSpace(*patch.Currency))
	}
	if patch.Level != nil {
		level = *patch.Level
	}
	if patch.Language != nil {
		language = strings.TrimSpace(*patch.Language)
	}
	if patch.CategoryID != nil {
		if *patch.CategoryID == "" {
			categoryID = nil
		} else {
			categoryID = patch.CategoryID
		}
	}

	const q = `
		UPDATE courses SET
			category_id = $2, title = $3, summary = $4, description = $5,
			cover_url = $6, price_cents = $7, currency = $8, level = $9, language = $10
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, categoryID, title, summary, description, cover, price, currency, string(level), language)
	if err != nil {
		return nil, fmt.Errorf("update course: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("course not found")
	}
	return r.GetByID(ctx, id)
}

func (r *CourseRepository) SetStatus(ctx context.Context, id string, status domain.CourseStatus, publishedAt *time.Time) (*domain.Course, error) {
	const q = `
		UPDATE courses
		SET status = $2, published_at = $3
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, string(status), publishedAt)
	if err != nil {
		return nil, fmt.Errorf("set course status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("course not found")
	}
	return r.GetByID(ctx, id)
}

func (r *CourseRepository) SoftDelete(ctx context.Context, id string, at time.Time) error {
	const q = `
		UPDATE courses SET deleted_at = $2, status = 'archived'
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, at)
	if err != nil {
		return fmt.Errorf("delete course: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("course not found")
	}
	return nil
}

func (r *CourseRepository) List(ctx context.Context, filter domain.CourseListFilter) ([]domain.Course, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	where := []string{"c.deleted_at IS NULL"}
	args := make([]any, 0, 8)
	n := 1

	if filter.Admin {
		// all statuses unless filtered
	} else if filter.OwnerID != "" {
		where = append(where, fmt.Sprintf("(c.status = 'published' OR c.teacher_id = $%d)", n))
		args = append(args, filter.OwnerID)
		n++
	} else {
		where = append(where, "c.status = 'published'")
	}

	if filter.Status != "" {
		where = append(where, fmt.Sprintf("c.status = $%d", n))
		args = append(args, string(filter.Status))
		n++
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		where = append(where, fmt.Sprintf("(c.title ILIKE $%d OR c.summary ILIKE $%d)", n, n))
		args = append(args, "%"+q+"%")
		n++
	}
	if filter.CategorySlug != "" {
		where = append(where, fmt.Sprintf("cat.slug = $%d", n))
		args = append(args, filter.CategorySlug)
		n++
	}
	if filter.TagSlug != "" {
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM course_tags ct
			JOIN tags t ON t.id = ct.tag_id
			WHERE ct.course_id = c.id AND lower(t.slug) = lower($%d)
		)`, n))
		args = append(args, filter.TagSlug)
		n++
	}
	if filter.OrganizationID != "" {
		where = append(where, fmt.Sprintf("c.organization_id = $%d", n))
		args = append(args, filter.OrganizationID)
		n++
	}
	if filter.TeacherID != "" {
		where = append(where, fmt.Sprintf("c.teacher_id = $%d", n))
		args = append(args, filter.TeacherID)
		n++
	}
	if filter.Level != "" {
		where = append(where, fmt.Sprintf("c.level = $%d", n))
		args = append(args, string(filter.Level))
		n++
	}

	whereSQL := strings.Join(where, " AND ")

	countQ := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM courses c
		LEFT JOIN categories cat ON cat.id = c.category_id
		WHERE %s`, whereSQL)

	var total int64
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count courses: %w", err)
	}

	listArgs := append(append([]any{}, args...), perPage, offset)
	listQ := fmt.Sprintf(`
		SELECT %s
		FROM courses c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.teacher_id
		WHERE %s
		ORDER BY c.created_at DESC
		LIMIT $%d OFFSET $%d`, courseSelect, whereSQL, n, n+1)

	rows, err := r.pool.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list courses: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Course, 0, perPage)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *course)
	}
	return out, total, rows.Err()
}

func scanCourse(row scannable) (*domain.Course, error) {
	var c domain.Course
	var orgID, catID *string
	var status, level string
	if err := row.Scan(
		&c.ID,
		&orgID,
		&c.TeacherID,
		&catID,
		&c.Title,
		&c.Slug,
		&c.Summary,
		&c.Description,
		&status,
		&c.CoverURL,
		&c.PriceCents,
		&c.Currency,
		&level,
		&c.Language,
		&c.PublishedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.EnrollmentCapacity,
		&c.EnrollmentOpen,
		&c.WaitlistEnabled,
		&c.CategoryName,
		&c.TeacherName,
	); err != nil {
		return nil, err
	}
	c.OrganizationID = orgID
	c.CategoryID = catID
	c.Status = domain.CourseStatus(status)
	c.Level = domain.CourseLevel(level)
	return &c, nil
}

func (r *CourseRepository) UpdateEnrollmentSettings(ctx context.Context, id string, settings domain.CourseEnrollmentSettings) (*domain.Course, error) {
	const q = `
		UPDATE courses
		SET enrollment_capacity = $2, enrollment_open = $3, waitlist_enabled = $4
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, settings.Capacity, settings.EnrollmentOpen, settings.WaitlistEnabled)
	if err != nil {
		return nil, fmt.Errorf("update enrollment settings: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("course not found")
	}
	return r.GetByID(ctx, id)
}
