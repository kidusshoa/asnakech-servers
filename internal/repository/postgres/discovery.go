package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchRepository struct {
	pool *pgxpool.Pool
}

func NewSearchRepository(pool *pgxpool.Pool) *SearchRepository {
	return &SearchRepository{pool: pool}
}

func (r *SearchRepository) Search(ctx context.Context, filter domain.SearchFilter) (*domain.SearchResults, error) {
	q := strings.TrimSpace(filter.Query)
	if q == "" {
		return nil, apperr.Validation("q is required")
	}
	page, perPage := normalizeSearchPage(filter.Page, filter.PerPage)
	out := &domain.SearchResults{Query: q}

	searchType := filter.Type
	if searchType == "" {
		searchType = domain.SearchTypeAll
	}

	if searchType == domain.SearchTypeAll || searchType == domain.SearchTypeCourses {
		courses, err := r.searchCourses(ctx, q, filter, page, perPage)
		if err != nil {
			return nil, err
		}
		out.Courses = courses
	}
	if searchType == domain.SearchTypeAll || searchType == domain.SearchTypeCategories {
		cats, err := r.searchCategories(ctx, q, 10)
		if err != nil {
			return nil, err
		}
		out.Categories = cats
	}
	if searchType == domain.SearchTypeAll || searchType == domain.SearchTypeTeachers {
		teachers, err := r.searchTeachers(ctx, q, 10)
		if err != nil {
			return nil, err
		}
		out.Teachers = teachers
	}
	return out, nil
}

func (r *SearchRepository) searchCourses(ctx context.Context, q string, filter domain.SearchFilter, page, perPage int) ([]domain.SearchCourseHit, error) {
	offset := (page - 1) * perPage
	where := []string{"c.deleted_at IS NULL", "c.status = 'published'", "c.search_vector @@ plainto_tsquery('simple', $1)"}
	args := []any{q}
	n := 2

	if lang := strings.TrimSpace(filter.Language); lang != "" {
		where = append(where, fmt.Sprintf("c.language = $%d", n))
		args = append(args, lang)
		n++
	}
	if filter.Level != "" {
		where = append(where, fmt.Sprintf("c.level = $%d", n))
		args = append(args, string(filter.Level))
		n++
	}

	whereSQL := strings.Join(where, " AND ")
	listQ := fmt.Sprintf(`
		SELECT c.id::text, c.organization_id::text, c.teacher_id::text, c.category_id::text,
		       c.title, c.slug, c.summary, c.description, c.status, c.cover_url,
		       c.price_cents, c.currency, c.level, c.language, c.published_at,
		       c.created_at, c.updated_at,
		       c.enrollment_capacity, c.enrollment_open, c.waitlist_enabled,
		       COALESCE(cat.name, ''), COALESCE(u.full_name, ''),
		       ts_rank(c.search_vector, plainto_tsquery('simple', $1))::float4
		FROM courses c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.teacher_id
		WHERE %s
		ORDER BY ts_rank(c.search_vector, plainto_tsquery('simple', $1)) DESC, c.published_at DESC NULLS LAST
		LIMIT $%d OFFSET $%d`, whereSQL, n, n+1)

	args = append(args, perPage, offset)
	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, fmt.Errorf("search courses: %w", err)
	}
	defer rows.Close()

	out := make([]domain.SearchCourseHit, 0)
	for rows.Next() {
		var hit domain.SearchCourseHit
		var orgID, catID *string
		var status, level, currency string
		var publishedAt *time.Time
		if err := rows.Scan(
			&hit.ID, &orgID, &hit.TeacherID, &catID,
			&hit.Title, &hit.Slug, &hit.Summary, &hit.Description, &status, &hit.CoverURL,
			&hit.PriceCents, &currency, &level, &hit.Language, &publishedAt,
			&hit.CreatedAt, &hit.UpdatedAt,
			&hit.EnrollmentCapacity, &hit.EnrollmentOpen, &hit.WaitlistEnabled,
			&hit.CategoryName, &hit.TeacherName,
			&hit.Rank,
		); err != nil {
			return nil, err
		}
		hit.OrganizationID = orgID
		hit.CategoryID = catID
		hit.Status = domain.CourseStatus(status)
		hit.Currency = currency
		hit.Level = domain.CourseLevel(level)
		hit.PublishedAt = publishedAt
		out = append(out, hit)
	}
	return out, rows.Err()
}

func (r *SearchRepository) searchCategories(ctx context.Context, q string, limit int) ([]domain.Category, error) {
	const query = `
		SELECT id::text, name, slug, description, created_at, updated_at
		FROM categories
		WHERE name ILIKE $1 OR description ILIKE $1 OR slug ILIKE $1
		ORDER BY name
		LIMIT $2`
	pattern := "%" + q + "%"
	rows, err := r.pool.Query(ctx, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search categories: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Category, 0)
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *SearchRepository) searchTeachers(ctx context.Context, q string, limit int) ([]domain.SearchTeacherHit, error) {
	const query = `
		SELECT u.id::text, u.full_name, u.email
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL AND u.is_active = TRUE
		  AND r.code = 'teacher'
		  AND (u.full_name ILIKE $1 OR u.email ILIKE $1)
		ORDER BY u.full_name
		LIMIT $2`
	pattern := "%" + q + "%"
	rows, err := r.pool.Query(ctx, query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search teachers: %w", err)
	}
	defer rows.Close()

	out := make([]domain.SearchTeacherHit, 0)
	for rows.Next() {
		var t domain.SearchTeacherHit
		if err := rows.Scan(&t.ID, &t.FullName, &t.Email); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *SearchRepository) Recommendations(ctx context.Context, userID, locale string, limit int) ([]domain.CourseRecommendation, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}

	const q = `
		SELECT c.id::text, c.organization_id::text, c.teacher_id::text, c.category_id::text,
		       c.title, c.slug, c.summary, c.description, c.status, c.cover_url,
		       c.price_cents, c.currency, c.level, c.language, c.published_at,
		       c.created_at, c.updated_at,
		       c.enrollment_capacity, c.enrollment_open, c.waitlist_enabled,
		       COALESCE(cat.name, ''), COALESCE(u.full_name, ''),
		       COUNT(e.id)::int
		FROM courses c
		LEFT JOIN categories cat ON cat.id = c.category_id
		LEFT JOIN users u ON u.id = c.teacher_id
		LEFT JOIN enrollments e ON e.course_id = c.id AND e.status = 'active'
		WHERE c.deleted_at IS NULL
		  AND c.status = 'published'
		  AND NOT EXISTS (
		    SELECT 1 FROM enrollments mine
		    WHERE mine.course_id = c.id AND mine.user_id = $1
		      AND mine.status IN ('active', 'waitlisted')
		  )
		GROUP BY c.id, cat.name, u.full_name
		ORDER BY
		  CASE WHEN c.language = $2 THEN 0 ELSE 1 END,
		  COUNT(e.id) DESC,
		  c.published_at DESC NULLS LAST
		LIMIT $3`

	rows, err := r.pool.Query(ctx, q, userID, locale, limit)
	if err != nil {
		return nil, fmt.Errorf("recommendations: %w", err)
	}
	defer rows.Close()

	out := make([]domain.CourseRecommendation, 0)
	for rows.Next() {
		var rec domain.CourseRecommendation
		var orgID, catID *string
		var status, level, currency string
		var publishedAt *time.Time
		var enrollCount int
		if err := rows.Scan(
			&rec.ID, &orgID, &rec.TeacherID, &catID,
			&rec.Title, &rec.Slug, &rec.Summary, &rec.Description, &status, &rec.CoverURL,
			&rec.PriceCents, &currency, &level, &rec.Language, &publishedAt,
			&rec.CreatedAt, &rec.UpdatedAt,
			&rec.EnrollmentCapacity, &rec.EnrollmentOpen, &rec.WaitlistEnabled,
			&rec.CategoryName, &rec.TeacherName,
			&enrollCount,
		); err != nil {
			return nil, err
		}
		rec.OrganizationID = orgID
		rec.CategoryID = catID
		rec.Status = domain.CourseStatus(status)
		rec.Currency = currency
		rec.Level = domain.CourseLevel(level)
		rec.PublishedAt = publishedAt
		rec.Score = enrollCount
		if rec.Language == locale {
			rec.Reason = "popular_in_your_language"
		} else {
			rec.Reason = "popular"
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func normalizeSearchPage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 50 {
		perPage = 50
	}
	return page, perPage
}

type ParentLinkRepository struct {
	pool *pgxpool.Pool
}

func NewParentLinkRepository(pool *pgxpool.Pool) *ParentLinkRepository {
	return &ParentLinkRepository{pool: pool}
}

func (r *ParentLinkRepository) Create(ctx context.Context, link *domain.ParentStudentLink) error {
	const q = `
		INSERT INTO parent_student_links (parent_user_id, student_user_id, status)
		VALUES ($1, $2, 'active')
		ON CONFLICT (parent_user_id, student_user_id)
		DO UPDATE SET status = 'active', updated_at = NOW()
		RETURNING id::text, created_at, updated_at`
	err := r.pool.QueryRow(ctx, q, link.ParentUserID, link.StudentUserID).
		Scan(&link.ID, &link.CreatedAt, &link.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create parent link: %w", err)
	}
	link.Status = "active"
	return nil
}

func (r *ParentLinkRepository) ListByParent(ctx context.Context, parentID string) ([]domain.ParentStudentLink, error) {
	const q = `
		SELECT l.id::text, l.parent_user_id::text, l.student_user_id::text, l.status,
		       l.created_at, l.updated_at,
		       u.email, u.full_name
		FROM parent_student_links l
		JOIN users u ON u.id = l.student_user_id
		WHERE l.parent_user_id = $1 AND l.status = 'active'
		ORDER BY u.full_name`
	rows, err := r.pool.Query(ctx, q, parentID)
	if err != nil {
		return nil, fmt.Errorf("list parent links: %w", err)
	}
	defer rows.Close()

	out := make([]domain.ParentStudentLink, 0)
	for rows.Next() {
		var l domain.ParentStudentLink
		if err := rows.Scan(
			&l.ID, &l.ParentUserID, &l.StudentUserID, &l.Status,
			&l.CreatedAt, &l.UpdatedAt,
			&l.StudentEmail, &l.StudentFullName,
		); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *ParentLinkRepository) Revoke(ctx context.Context, parentID, studentID string) error {
	const q = `
		UPDATE parent_student_links SET status = 'revoked'
		WHERE parent_user_id = $1 AND student_user_id = $2 AND status = 'active'`
	tag, err := r.pool.Exec(ctx, q, parentID, studentID)
	if err != nil {
		return fmt.Errorf("revoke parent link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("link not found")
	}
	return nil
}

func (r *ParentLinkRepository) GetActive(ctx context.Context, parentID, studentID string) (*domain.ParentStudentLink, error) {
	const q = `
		SELECT l.id::text, l.parent_user_id::text, l.student_user_id::text, l.status,
		       l.created_at, l.updated_at,
		       u.email, u.full_name
		FROM parent_student_links l
		JOIN users u ON u.id = l.student_user_id
		WHERE l.parent_user_id = $1 AND l.student_user_id = $2 AND l.status = 'active'`
	var l domain.ParentStudentLink
	err := r.pool.QueryRow(ctx, q, parentID, studentID).Scan(
		&l.ID, &l.ParentUserID, &l.StudentUserID, &l.Status,
		&l.CreatedAt, &l.UpdatedAt,
		&l.StudentEmail, &l.StudentFullName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("link not found")
		}
		return nil, fmt.Errorf("get parent link: %w", err)
	}
	return &l, nil
}
