package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnalyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{pool: pool}
}

func (r *AnalyticsRepository) PlatformOverview(ctx context.Context) (*domain.PlatformOverview, error) {
	out := &domain.PlatformOverview{
		GeneratedAt:         time.Now().UTC(),
		UsersByRole:         map[string]int64{},
		CoursesByStatus:     map[string]int64{},
		EnrollmentsByStatus: map[string]int64{},
		RevenueCurrency:     "ETB",
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND is_active = TRUE
	`).Scan(&out.UsersTotal); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT r.code, COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL AND u.is_active = TRUE
		GROUP BY r.code
	`)
	if err != nil {
		return nil, fmt.Errorf("users by role: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		var n int64
		if err := rows.Scan(&code, &n); err != nil {
			return nil, err
		}
		out.UsersByRole[code] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`).Scan(&out.CoursesTotal); err != nil {
		return nil, fmt.Errorf("count courses: %w", err)
	}

	rows, err = r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM courses WHERE deleted_at IS NULL GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("courses by status: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out.CoursesByStatus[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM enrollments`).Scan(&out.EnrollmentsTotal); err != nil {
		return nil, fmt.Errorf("count enrollments: %w", err)
	}

	rows, err = r.pool.Query(ctx, `SELECT status, COUNT(*) FROM enrollments GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("enrollments by status: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		out.EnrollmentsByStatus[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL`).Scan(&out.OrganizationsTotal); err != nil {
		return nil, fmt.Errorf("count organizations: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM certificates WHERE revoked_at IS NULL`).Scan(&out.CertificatesTotal); err != nil {
		return nil, fmt.Errorf("count certificates: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_cents), 0), COALESCE(MAX(currency), 'ETB')
		FROM orders WHERE status = 'paid'
	`).Scan(&out.OrdersPaid, &out.RevenueCents, &out.RevenueCurrency); err != nil {
		return nil, fmt.Errorf("revenue totals: %w", err)
	}

	since := time.Now().UTC().AddDate(0, 0, -7)
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM enrollments WHERE created_at >= $1
	`, since).Scan(&out.EnrollmentsLast7Days); err != nil {
		return nil, fmt.Errorf("enrollments last 7d: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_cents), 0) FROM orders WHERE status = 'paid' AND paid_at >= $1
	`, since).Scan(&out.RevenueLast7DaysCents); err != nil {
		return nil, fmt.Errorf("revenue last 7d: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at >= $1
	`, since).Scan(&out.NewUsersLast7Days); err != nil {
		return nil, fmt.Errorf("users last 7d: %w", err)
	}

	return out, nil
}

func (r *AnalyticsRepository) CourseAnalytics(ctx context.Context, courseID string) (*domain.CourseAnalytics, error) {
	out := &domain.CourseAnalytics{
		CourseID:    courseID,
		GeneratedAt: time.Now().UTC(),
		Currency:    "ETB",
	}

	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(title, ''), COALESCE(currency, 'ETB')
		FROM courses WHERE id = $1 AND deleted_at IS NULL
	`, courseID).Scan(&out.CourseTitle, &out.Currency)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperr.NotFound("course not found")
		}
		return nil, fmt.Errorf("course title: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*) FROM enrollments WHERE course_id = $1 GROUP BY status
	`, courseID)
	if err != nil {
		return nil, fmt.Errorf("enrollment counts: %w", err)
	}
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return nil, err
		}
		switch domain.EnrollmentStatus(status) {
		case domain.EnrollmentStatusActive:
			out.EnrollmentsActive = n
		case domain.EnrollmentStatusWaitlisted:
			out.EnrollmentsWaitlisted = n
		case domain.EnrollmentStatusCancelled:
			out.EnrollmentsCancelled = n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM course_progress
		WHERE course_id = $1 AND completed_at IS NOT NULL
	`, courseID).Scan(&out.CompletionCount); err != nil {
		return nil, fmt.Errorf("completion count: %w", err)
	}

	var avgProgress *float64
	if err := r.pool.QueryRow(ctx, `
		SELECT AVG(percent)::float8 FROM course_progress WHERE course_id = $1
	`, courseID).Scan(&avgProgress); err != nil {
		return nil, fmt.Errorf("avg progress: %w", err)
	}
	if avgProgress != nil {
		out.AverageProgressPercent = int(*avgProgress + 0.5)
	}

	if out.EnrollmentsActive > 0 {
		out.CompletionRatePercent = int(float64(out.CompletionCount)/float64(out.EnrollmentsActive)*100 + 0.5)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_cents), 0)
		FROM orders WHERE course_id = $1 AND status = 'paid'
	`, courseID).Scan(&out.OrdersPaid, &out.RevenueCents); err != nil {
		return nil, fmt.Errorf("course revenue: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM certificates WHERE course_id = $1 AND revoked_at IS NULL
	`, courseID).Scan(&out.CertificatesIssued); err != nil {
		return nil, fmt.Errorf("certificates: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizzes WHERE course_id = $1 AND status = 'published'
	`, courseID).Scan(&out.QuizzesPublished); err != nil {
		return nil, fmt.Errorf("quizzes: %w", err)
	}

	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assignments WHERE course_id = $1 AND status = 'published'
	`, courseID).Scan(&out.AssignmentsPublished); err != nil {
		return nil, fmt.Errorf("assignments: %w", err)
	}

	return out, nil
}

func (r *AnalyticsRepository) EnrollmentReport(ctx context.Context, filter domain.ReportRange) ([]domain.EnrollmentReportRow, error) {
	from, to := reportBounds(filter)
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', e.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS period,
		       e.course_id::text,
		       COALESCE(c.title, ''),
		       e.status,
		       COUNT(*)
		FROM enrollments e
		LEFT JOIN courses c ON c.id = e.course_id
		WHERE e.created_at >= $1 AND e.created_at < $2
		GROUP BY 1, e.course_id, c.title, e.status
		ORDER BY 1 DESC, c.title, e.status
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("enrollment report: %w", err)
	}
	defer rows.Close()

	out := make([]domain.EnrollmentReportRow, 0)
	for rows.Next() {
		var row domain.EnrollmentReportRow
		if err := rows.Scan(&row.Period, &row.CourseID, &row.CourseTitle, &row.Status, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) RevenueReport(ctx context.Context, filter domain.ReportRange) ([]domain.RevenueReportRow, error) {
	from, to := reportBounds(filter)
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', o.paid_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS period,
		       o.course_id::text,
		       COALESCE(c.title, ''),
		       o.currency,
		       COUNT(*),
		       COALESCE(SUM(o.total_cents), 0)
		FROM orders o
		LEFT JOIN courses c ON c.id = o.course_id
		WHERE o.status = 'paid'
		  AND o.paid_at IS NOT NULL
		  AND o.paid_at >= $1 AND o.paid_at < $2
		GROUP BY 1, o.course_id, c.title, o.currency
		ORDER BY 1 DESC, c.title
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("revenue report: %w", err)
	}
	defer rows.Close()

	out := make([]domain.RevenueReportRow, 0)
	for rows.Next() {
		var row domain.RevenueReportRow
		if err := rows.Scan(&row.Period, &row.CourseID, &row.CourseTitle, &row.Currency, &row.OrderCount, &row.RevenueCents); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *AnalyticsRepository) UserGrowthReport(ctx context.Context, filter domain.ReportRange) ([]domain.UserGrowthPoint, error) {
	from, to := reportBounds(filter)
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(date_trunc('day', u.created_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS period,
		       r.code,
		       COUNT(*)
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL
		  AND u.created_at >= $1 AND u.created_at < $2
		GROUP BY 1, r.code
		ORDER BY 1 DESC, r.code
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("user growth report: %w", err)
	}
	defer rows.Close()

	out := make([]domain.UserGrowthPoint, 0)
	for rows.Next() {
		var row domain.UserGrowthPoint
		if err := rows.Scan(&row.Period, &row.Role, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func reportBounds(filter domain.ReportRange) (time.Time, time.Time) {
	now := time.Now().UTC()
	to := now
	if filter.To != nil {
		to = filter.To.UTC()
	}
	from := to.AddDate(0, 0, -30)
	if filter.From != nil {
		from = filter.From.UTC()
	}
	if !from.Before(to) {
		from = to.AddDate(0, 0, -30)
	}
	return from, to
}
