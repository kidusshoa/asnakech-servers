package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LessonProgressRepository struct {
	pool *pgxpool.Pool
}

func NewLessonProgressRepository(pool *pgxpool.Pool) *LessonProgressRepository {
	return &LessonProgressRepository{pool: pool}
}

func (r *LessonProgressRepository) Upsert(ctx context.Context, p *domain.LessonProgress) error {
	const q = `
		INSERT INTO lesson_progress (
			user_id, course_id, lesson_id, status, percent, last_position, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (user_id, lesson_id) DO UPDATE SET
			status = EXCLUDED.status,
			percent = EXCLUDED.percent,
			last_position = EXCLUDED.last_position,
			completed_at = EXCLUDED.completed_at
		RETURNING id::text, created_at, updated_at`

	return r.pool.QueryRow(ctx, q,
		p.UserID, p.CourseID, p.LessonID, string(p.Status), p.Percent, p.LastPosition, p.CompletedAt,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *LessonProgressRepository) GetByUserLesson(ctx context.Context, userID, lessonID string) (*domain.LessonProgress, error) {
	const q = `
		SELECT lp.id::text, lp.user_id::text, lp.course_id::text, lp.lesson_id::text,
		       lp.status, lp.percent, lp.last_position, lp.completed_at, lp.created_at, lp.updated_at,
		       COALESCE(l.title, ''), COALESCE(l.slug, '')
		FROM lesson_progress lp
		LEFT JOIN lessons l ON l.id = lp.lesson_id
		WHERE lp.user_id = $1 AND lp.lesson_id = $2`

	p, err := scanLessonProgress(r.pool.QueryRow(ctx, q, userID, lessonID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("lesson progress not found")
		}
		return nil, fmt.Errorf("get lesson progress: %w", err)
	}
	return p, nil
}

func (r *LessonProgressRepository) ListByUserCourse(ctx context.Context, userID, courseID string) ([]domain.LessonProgress, error) {
	const q = `
		SELECT lp.id::text, lp.user_id::text, lp.course_id::text, lp.lesson_id::text,
		       lp.status, lp.percent, lp.last_position, lp.completed_at, lp.created_at, lp.updated_at,
		       COALESCE(l.title, ''), COALESCE(l.slug, '')
		FROM lesson_progress lp
		LEFT JOIN lessons l ON l.id = lp.lesson_id
		WHERE lp.user_id = $1 AND lp.course_id = $2
		ORDER BY lp.updated_at DESC`

	rows, err := r.pool.Query(ctx, q, userID, courseID)
	if err != nil {
		return nil, fmt.Errorf("list lesson progress: %w", err)
	}
	defer rows.Close()

	out := make([]domain.LessonProgress, 0)
	for rows.Next() {
		p, err := scanLessonProgress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *LessonProgressRepository) CountCompletedPublished(ctx context.Context, userID, courseID string) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM lesson_progress lp
		JOIN lessons l ON l.id = lp.lesson_id
		JOIN course_modules m ON m.id = l.module_id
		WHERE lp.user_id = $1
		  AND m.course_id = $2
		  AND lp.status = 'completed'
		  AND l.status = 'published'`
	var n int
	if err := r.pool.QueryRow(ctx, q, userID, courseID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count completed lessons: %w", err)
	}
	return n, nil
}

func scanLessonProgress(row scannable) (*domain.LessonProgress, error) {
	var p domain.LessonProgress
	var status string
	if err := row.Scan(
		&p.ID, &p.UserID, &p.CourseID, &p.LessonID,
		&status, &p.Percent, &p.LastPosition, &p.CompletedAt, &p.CreatedAt, &p.UpdatedAt,
		&p.LessonTitle, &p.LessonSlug,
	); err != nil {
		return nil, err
	}
	p.Status = domain.LessonProgressStatus(status)
	return &p, nil
}

type CourseProgressRepository struct {
	pool *pgxpool.Pool
}

func NewCourseProgressRepository(pool *pgxpool.Pool) *CourseProgressRepository {
	return &CourseProgressRepository{pool: pool}
}

func (r *CourseProgressRepository) Upsert(ctx context.Context, p *domain.CourseProgress) error {
	const q = `
		INSERT INTO course_progress (
			user_id, course_id, enrollment_id, percent, completed_lessons, total_lessons,
			last_lesson_id, completed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (user_id, course_id) DO UPDATE SET
			enrollment_id = EXCLUDED.enrollment_id,
			percent = EXCLUDED.percent,
			completed_lessons = EXCLUDED.completed_lessons,
			total_lessons = EXCLUDED.total_lessons,
			last_lesson_id = EXCLUDED.last_lesson_id,
			completed_at = EXCLUDED.completed_at
		RETURNING id::text, created_at, updated_at`

	return r.pool.QueryRow(ctx, q,
		p.UserID, p.CourseID, p.EnrollmentID, p.Percent, p.CompletedLessons, p.TotalLessons,
		p.LastLessonID, p.CompletedAt,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *CourseProgressRepository) GetByUserCourse(ctx context.Context, userID, courseID string) (*domain.CourseProgress, error) {
	const q = `
		SELECT cp.id::text, cp.user_id::text, cp.course_id::text, cp.enrollment_id::text,
		       cp.percent, cp.completed_lessons, cp.total_lessons, cp.last_lesson_id::text,
		       cp.completed_at, cp.created_at, cp.updated_at,
		       COALESCE(c.title, ''), COALESCE(c.slug, '')
		FROM course_progress cp
		LEFT JOIN courses c ON c.id = cp.course_id
		WHERE cp.user_id = $1 AND cp.course_id = $2`

	p, err := scanCourseProgress(r.pool.QueryRow(ctx, q, userID, courseID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("course progress not found")
		}
		return nil, fmt.Errorf("get course progress: %w", err)
	}
	return p, nil
}

func (r *CourseProgressRepository) ListByUser(ctx context.Context, userID string) ([]domain.CourseProgress, error) {
	const q = `
		SELECT cp.id::text, cp.user_id::text, cp.course_id::text, cp.enrollment_id::text,
		       cp.percent, cp.completed_lessons, cp.total_lessons, cp.last_lesson_id::text,
		       cp.completed_at, cp.created_at, cp.updated_at,
		       COALESCE(c.title, ''), COALESCE(c.slug, '')
		FROM course_progress cp
		LEFT JOIN courses c ON c.id = cp.course_id
		WHERE cp.user_id = $1
		ORDER BY cp.updated_at DESC`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list course progress: %w", err)
	}
	defer rows.Close()

	out := make([]domain.CourseProgress, 0)
	for rows.Next() {
		p, err := scanCourseProgress(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func scanCourseProgress(row scannable) (*domain.CourseProgress, error) {
	var p domain.CourseProgress
	var enrollmentID, lastLessonID *string
	if err := row.Scan(
		&p.ID, &p.UserID, &p.CourseID, &enrollmentID,
		&p.Percent, &p.CompletedLessons, &p.TotalLessons, &lastLessonID,
		&p.CompletedAt, &p.CreatedAt, &p.UpdatedAt,
		&p.CourseTitle, &p.CourseSlug,
	); err != nil {
		return nil, err
	}
	p.EnrollmentID = enrollmentID
	p.LastLessonID = lastLessonID
	return &p, nil
}
