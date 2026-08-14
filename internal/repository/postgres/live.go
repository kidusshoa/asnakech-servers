package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LiveSessionRepository struct {
	pool *pgxpool.Pool
}

func NewLiveSessionRepository(pool *pgxpool.Pool) *LiveSessionRepository {
	return &LiveSessionRepository{pool: pool}
}

const liveSessionSelect = `
	ls.id::text, ls.course_id::text, ls.lesson_id::text, ls.title, ls.description, ls.status,
	ls.starts_at, ls.ends_at, ls.timezone, ls.provider,
	ls.join_url, ls.host_url, ls.external_id, ls.provider_metadata,
	ls.created_by::text, ls.created_at, ls.updated_at,
	c.title, c.slug`

func (r *LiveSessionRepository) Create(ctx context.Context, s *domain.LiveSession) error {
	meta, err := json.Marshal(s.ProviderMetadata)
	if err != nil {
		return fmt.Errorf("marshal provider metadata: %w", err)
	}
	const sql = `
		INSERT INTO live_sessions (
			course_id, lesson_id, title, description, status,
			starts_at, ends_at, timezone, provider,
			join_url, host_url, external_id, provider_metadata, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id::text, created_at, updated_at`

	err = r.pool.QueryRow(ctx, sql,
		s.CourseID, s.LessonID, s.Title, s.Description, string(s.Status),
		s.StartsAt, s.EndsAt, s.Timezone, string(s.Provider),
		s.JoinURL, s.HostURL, s.ExternalID, meta, s.CreatedBy,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create live session: %w", err)
	}
	return nil
}

func (r *LiveSessionRepository) GetByID(ctx context.Context, id string) (*domain.LiveSession, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM live_sessions ls
		JOIN courses c ON c.id = ls.course_id
		WHERE ls.id = $1`, liveSessionSelect)
	s, err := scanLiveSession(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("session not found")
		}
		return nil, fmt.Errorf("get live session: %w", err)
	}
	return s, nil
}

func (r *LiveSessionRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.LiveSession, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM live_sessions ls
		JOIN courses c ON c.id = ls.course_id
		WHERE ls.course_id = $1
		ORDER BY ls.starts_at ASC`, liveSessionSelect)

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list live sessions: %w", err)
	}
	defer rows.Close()

	out := make([]domain.LiveSession, 0)
	for rows.Next() {
		s, err := scanLiveSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *LiveSessionRepository) Update(ctx context.Context, s *domain.LiveSession) (*domain.LiveSession, error) {
	meta, err := json.Marshal(s.ProviderMetadata)
	if err != nil {
		return nil, fmt.Errorf("marshal provider metadata: %w", err)
	}
	const sql = `
		UPDATE live_sessions
		SET lesson_id = $2, title = $3, description = $4,
		    starts_at = $5, ends_at = $6, timezone = $7, provider = $8,
		    join_url = $9, host_url = $10, external_id = $11, provider_metadata = $12
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql,
		s.ID, s.LessonID, s.Title, s.Description,
		s.StartsAt, s.EndsAt, s.Timezone, string(s.Provider),
		s.JoinURL, s.HostURL, s.ExternalID, meta,
	)
	if err != nil {
		return nil, fmt.Errorf("update live session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("session not found")
	}
	return r.GetByID(ctx, s.ID)
}

func (r *LiveSessionRepository) SetStatus(ctx context.Context, id string, status domain.LiveSessionStatus) (*domain.LiveSession, error) {
	const sql = `UPDATE live_sessions SET status = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sql, id, string(status))
	if err != nil {
		return nil, fmt.Errorf("set session status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("session not found")
	}
	return r.GetByID(ctx, id)
}

func (r *LiveSessionRepository) ListCalendar(ctx context.Context, filter domain.CalendarFilter) ([]domain.LiveSession, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 50
	}
	if filter.PerPage > 200 {
		filter.PerPage = 200
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	args := []any{filter.From, filter.To, filter.UserID}
	where := `
		ls.status IN ('scheduled', 'completed')
		AND ls.starts_at >= $1 AND ls.starts_at < $2
		AND (
			c.teacher_id = $3
			OR EXISTS (
				SELECT 1 FROM enrollments e
				WHERE e.course_id = c.id AND e.user_id = $3 AND e.status = 'active'
			)
		)
		AND (c.status = 'published' OR c.teacher_id = $3)`

	if filter.Admin {
		where = `
		ls.status IN ('scheduled', 'completed')
		AND ls.starts_at >= $1 AND ls.starts_at < $2`
		args = []any{filter.From, filter.To}
	}

	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM live_sessions ls
		JOIN courses c ON c.id = ls.course_id
		WHERE %s`, where)

	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count calendar: %w", err)
	}

	listArgs := append(args, filter.PerPage, offset)
	q := fmt.Sprintf(`
		SELECT %s
		FROM live_sessions ls
		JOIN courses c ON c.id = ls.course_id
		WHERE %s
		ORDER BY ls.starts_at ASC
		LIMIT $%d OFFSET $%d`, liveSessionSelect, where, len(listArgs)-1, len(listArgs))

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list calendar: %w", err)
	}
	defer rows.Close()

	out := make([]domain.LiveSession, 0)
	for rows.Next() {
		s, err := scanLiveSession(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *s)
	}
	return out, total, rows.Err()
}

type sessionAttendanceScanner interface {
	Scan(dest ...any) error
}

func scanLiveSession(row sessionAttendanceScanner) (*domain.LiveSession, error) {
	var s domain.LiveSession
	var lessonID *string
	var status, provider string
	var metaJSON []byte
	err := row.Scan(
		&s.ID, &s.CourseID, &lessonID, &s.Title, &s.Description, &status,
		&s.StartsAt, &s.EndsAt, &s.Timezone, &provider,
		&s.JoinURL, &s.HostURL, &s.ExternalID, &metaJSON,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		&s.CourseTitle, &s.CourseSlug,
	)
	if err != nil {
		return nil, err
	}
	s.LessonID = lessonID
	s.Status = domain.LiveSessionStatus(status)
	s.Provider = domain.LiveProvider(provider)
	s.ProviderMetadata = map[string]string{}
	if len(metaJSON) > 0 && string(metaJSON) != "{}" {
		_ = json.Unmarshal(metaJSON, &s.ProviderMetadata)
	}
	return &s, nil
}

// --- SessionAttendanceRepository ---

type SessionAttendanceRepository struct {
	pool *pgxpool.Pool
}

func NewSessionAttendanceRepository(pool *pgxpool.Pool) *SessionAttendanceRepository {
	return &SessionAttendanceRepository{pool: pool}
}

const attendanceSelect = `
	a.id::text, a.session_id::text, a.user_id::text, a.status,
	a.joined_at, a.left_at, a.marked_by::text, a.note,
	a.created_at, a.updated_at,
	u.email, u.full_name`

func (r *SessionAttendanceRepository) Upsert(ctx context.Context, a *domain.SessionAttendance) error {
	const sql = `
		INSERT INTO session_attendance (session_id, user_id, status, joined_at, left_at, marked_by, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (session_id, user_id) DO UPDATE SET
			status = EXCLUDED.status,
			joined_at = COALESCE(EXCLUDED.joined_at, session_attendance.joined_at),
			left_at = EXCLUDED.left_at,
			marked_by = EXCLUDED.marked_by,
			note = EXCLUDED.note
		RETURNING id::text, created_at, updated_at`

	return r.pool.QueryRow(ctx, sql,
		a.SessionID, a.UserID, string(a.Status), a.JoinedAt, a.LeftAt, a.MarkedBy, a.Note,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *SessionAttendanceRepository) GetBySessionUser(ctx context.Context, sessionID, userID string) (*domain.SessionAttendance, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM session_attendance a
		JOIN users u ON u.id = a.user_id
		WHERE a.session_id = $1 AND a.user_id = $2`, attendanceSelect)

	a, err := scanAttendance(r.pool.QueryRow(ctx, q, sessionID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("attendance not found")
		}
		return nil, fmt.Errorf("get attendance: %w", err)
	}
	return a, nil
}

func (r *SessionAttendanceRepository) ListBySession(ctx context.Context, sessionID string) ([]domain.SessionAttendance, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM session_attendance a
		JOIN users u ON u.id = a.user_id
		WHERE a.session_id = $1
		ORDER BY u.full_name ASC, u.email ASC`, attendanceSelect)

	rows, err := r.pool.Query(ctx, q, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list attendance: %w", err)
	}
	defer rows.Close()

	out := make([]domain.SessionAttendance, 0)
	for rows.Next() {
		a, err := scanAttendance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func scanAttendance(row sessionAttendanceScanner) (*domain.SessionAttendance, error) {
	var a domain.SessionAttendance
	var status string
	var markedBy *string
	err := row.Scan(
		&a.ID, &a.SessionID, &a.UserID, &status,
		&a.JoinedAt, &a.LeftAt, &markedBy, &a.Note,
		&a.CreatedAt, &a.UpdatedAt,
		&a.UserEmail, &a.UserFullName,
	)
	if err != nil {
		return nil, err
	}
	a.Status = domain.AttendanceStatus(status)
	a.MarkedBy = markedBy
	return &a, nil
}
