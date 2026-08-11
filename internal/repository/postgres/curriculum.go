package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModuleRepository struct {
	pool *pgxpool.Pool
}

func NewModuleRepository(pool *pgxpool.Pool) *ModuleRepository {
	return &ModuleRepository{pool: pool}
}

func (r *ModuleRepository) Create(ctx context.Context, m *domain.CourseModule) error {
	const q = `
		INSERT INTO course_modules (course_id, title, position)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, m.CourseID, m.Title, m.Position).
		Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("module position conflict")
		}
		return fmt.Errorf("create module: %w", err)
	}
	return nil
}

func (r *ModuleRepository) GetByID(ctx context.Context, id string) (*domain.CourseModule, error) {
	const q = `
		SELECT id::text, course_id::text, title, position, created_at, updated_at
		FROM course_modules WHERE id = $1`

	var m domain.CourseModule
	err := r.pool.QueryRow(ctx, q, id).Scan(&m.ID, &m.CourseID, &m.Title, &m.Position, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("module not found")
		}
		return nil, fmt.Errorf("get module: %w", err)
	}
	return &m, nil
}

func (r *ModuleRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.CourseModule, error) {
	const q = `
		SELECT id::text, course_id::text, title, position, created_at, updated_at
		FROM course_modules
		WHERE course_id = $1
		ORDER BY position ASC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	defer rows.Close()

	out := make([]domain.CourseModule, 0)
	for rows.Next() {
		var m domain.CourseModule
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.Position, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ModuleRepository) Update(ctx context.Context, id, title string) (*domain.CourseModule, error) {
	const q = `UPDATE course_modules SET title = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, title)
	if err != nil {
		return nil, fmt.Errorf("update module: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("module not found")
	}
	return r.GetByID(ctx, id)
}

func (r *ModuleRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM course_modules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete module: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("module not found")
	}
	return nil
}

func (r *ModuleRepository) NextPosition(ctx context.Context, courseID string) (int, error) {
	const q = `SELECT COALESCE(MAX(position), -1) + 1 FROM course_modules WHERE course_id = $1`
	var pos int
	if err := r.pool.QueryRow(ctx, q, courseID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("next module position: %w", err)
	}
	return pos, nil
}

func (r *ModuleRepository) Reorder(ctx context.Context, courseID string, orderedIDs []string) error {
	return reorderPositions(ctx, r.pool, "course_modules", "course_id", courseID, orderedIDs)
}

type LessonRepository struct {
	pool *pgxpool.Pool
}

func NewLessonRepository(pool *pgxpool.Pool) *LessonRepository {
	return &LessonRepository{pool: pool}
}

func (r *LessonRepository) Create(ctx context.Context, l *domain.Lesson) error {
	const q = `
		INSERT INTO lessons (module_id, title, slug, summary, status, position, prerequisite_lesson_id, estimated_minutes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		l.ModuleID, l.Title, l.Slug, l.Summary, string(l.Status), l.Position, l.PrerequisiteLessonID, l.EstimatedMinutes,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("lesson slug or position conflict")
		}
		return fmt.Errorf("create lesson: %w", err)
	}
	return nil
}

func (r *LessonRepository) GetByID(ctx context.Context, id string) (*domain.Lesson, error) {
	const q = `
		SELECT id::text, module_id::text, title, slug, summary, status, position,
		       prerequisite_lesson_id::text, estimated_minutes, created_at, updated_at
		FROM lessons WHERE id = $1`

	l, err := scanLesson(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("lesson not found")
		}
		return nil, fmt.Errorf("get lesson: %w", err)
	}
	return l, nil
}

func (r *LessonRepository) ListByModule(ctx context.Context, moduleID string) ([]domain.Lesson, error) {
	const q = `
		SELECT id::text, module_id::text, title, slug, summary, status, position,
		       prerequisite_lesson_id::text, estimated_minutes, created_at, updated_at
		FROM lessons
		WHERE module_id = $1
		ORDER BY position ASC`

	rows, err := r.pool.Query(ctx, q, moduleID)
	if err != nil {
		return nil, fmt.Errorf("list lessons: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Lesson, 0)
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func (r *LessonRepository) Update(ctx context.Context, id string, title, summary string, minutes int, prereq *string) (*domain.Lesson, error) {
	const q = `
		UPDATE lessons
		SET title = $2, summary = $3, estimated_minutes = $4, prerequisite_lesson_id = $5
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, id, title, summary, minutes, prereq)
	if err != nil {
		return nil, fmt.Errorf("update lesson: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("lesson not found")
	}
	return r.GetByID(ctx, id)
}

func (r *LessonRepository) SetStatus(ctx context.Context, id string, status domain.LessonStatus) (*domain.Lesson, error) {
	const q = `UPDATE lessons SET status = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, string(status))
	if err != nil {
		return nil, fmt.Errorf("set lesson status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("lesson not found")
	}
	return r.GetByID(ctx, id)
}

func (r *LessonRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM lessons WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete lesson: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("lesson not found")
	}
	return nil
}

func (r *LessonRepository) NextPosition(ctx context.Context, moduleID string) (int, error) {
	const q = `SELECT COALESCE(MAX(position), -1) + 1 FROM lessons WHERE module_id = $1`
	var pos int
	if err := r.pool.QueryRow(ctx, q, moduleID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("next lesson position: %w", err)
	}
	return pos, nil
}

func (r *LessonRepository) Reorder(ctx context.Context, moduleID string, orderedIDs []string) error {
	return reorderPositions(ctx, r.pool, "lessons", "module_id", moduleID, orderedIDs)
}

func scanLesson(row scannable) (*domain.Lesson, error) {
	var l domain.Lesson
	var status string
	var prereq *string
	if err := row.Scan(
		&l.ID, &l.ModuleID, &l.Title, &l.Slug, &l.Summary, &status, &l.Position,
		&prereq, &l.EstimatedMinutes, &l.CreatedAt, &l.UpdatedAt,
	); err != nil {
		return nil, err
	}
	l.Status = domain.LessonStatus(status)
	l.PrerequisiteLessonID = prereq
	return &l, nil
}

type ContentBlockRepository struct {
	pool *pgxpool.Pool
}

func NewContentBlockRepository(pool *pgxpool.Pool) *ContentBlockRepository {
	return &ContentBlockRepository{pool: pool}
}

func (r *ContentBlockRepository) Create(ctx context.Context, b *domain.ContentBlock) error {
	const q = `
		INSERT INTO content_blocks (lesson_id, block_type, title, body, media_url, quiz_ref_id, position)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		b.LessonID, string(b.BlockType), b.Title, b.Body, b.MediaURL, b.QuizRefID, b.Position,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("content block position conflict")
		}
		return fmt.Errorf("create content block: %w", err)
	}
	return nil
}

func (r *ContentBlockRepository) GetByID(ctx context.Context, id string) (*domain.ContentBlock, error) {
	const q = `
		SELECT id::text, lesson_id::text, block_type, title, body, media_url, quiz_ref_id::text,
		       position, created_at, updated_at
		FROM content_blocks WHERE id = $1`

	b, err := scanBlock(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("content block not found")
		}
		return nil, fmt.Errorf("get content block: %w", err)
	}
	return b, nil
}

func (r *ContentBlockRepository) ListByLesson(ctx context.Context, lessonID string) ([]domain.ContentBlock, error) {
	const q = `
		SELECT id::text, lesson_id::text, block_type, title, body, media_url, quiz_ref_id::text,
		       position, created_at, updated_at
		FROM content_blocks
		WHERE lesson_id = $1
		ORDER BY position ASC`

	rows, err := r.pool.Query(ctx, q, lessonID)
	if err != nil {
		return nil, fmt.Errorf("list content blocks: %w", err)
	}
	defer rows.Close()

	out := make([]domain.ContentBlock, 0)
	for rows.Next() {
		b, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (r *ContentBlockRepository) Update(ctx context.Context, b *domain.ContentBlock) (*domain.ContentBlock, error) {
	const q = `
		UPDATE content_blocks
		SET block_type = $2, title = $3, body = $4, media_url = $5, quiz_ref_id = $6
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, q, b.ID, string(b.BlockType), b.Title, b.Body, b.MediaURL, b.QuizRefID)
	if err != nil {
		return nil, fmt.Errorf("update content block: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("content block not found")
	}
	return r.GetByID(ctx, b.ID)
}

func (r *ContentBlockRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM content_blocks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete content block: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("content block not found")
	}
	return nil
}

func (r *ContentBlockRepository) NextPosition(ctx context.Context, lessonID string) (int, error) {
	const q = `SELECT COALESCE(MAX(position), -1) + 1 FROM content_blocks WHERE lesson_id = $1`
	var pos int
	if err := r.pool.QueryRow(ctx, q, lessonID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("next block position: %w", err)
	}
	return pos, nil
}

func (r *ContentBlockRepository) Reorder(ctx context.Context, lessonID string, orderedIDs []string) error {
	return reorderPositions(ctx, r.pool, "content_blocks", "lesson_id", lessonID, orderedIDs)
}

func scanBlock(row scannable) (*domain.ContentBlock, error) {
	var b domain.ContentBlock
	var typ string
	var quizRef *string
	if err := row.Scan(
		&b.ID, &b.LessonID, &typ, &b.Title, &b.Body, &b.MediaURL, &quizRef,
		&b.Position, &b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return nil, err
	}
	b.BlockType = domain.ContentBlockType(typ)
	b.QuizRefID = quizRef
	return &b, nil
}

func reorderPositions(ctx context.Context, pool *pgxpool.Pool, table, parentCol, parentID string, orderedIDs []string) error {
	if len(orderedIDs) == 0 {
		return apperr.Validation("ids are required")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reorder: %w", err)
	}
	defer tx.Rollback(ctx)

	// Move to temporary negative positions to avoid unique conflicts.
	for i, id := range orderedIDs {
		q := fmt.Sprintf(`UPDATE %s SET position = $1 WHERE id = $2 AND %s = $3`, table, parentCol)
		tag, err := tx.Exec(ctx, q, -(i + 1), id, parentID)
		if err != nil {
			return fmt.Errorf("temp reorder: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return apperr.Validation("invalid id in reorder list")
		}
	}
	for i, id := range orderedIDs {
		q := fmt.Sprintf(`UPDATE %s SET position = $1 WHERE id = $2 AND %s = $3`, table, parentCol)
		if _, err := tx.Exec(ctx, q, i, id, parentID); err != nil {
			return fmt.Errorf("final reorder: %w", err)
		}
	}
	return tx.Commit(ctx)
}
