package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- QuizRepository ---

type QuizRepository struct {
	pool *pgxpool.Pool
}

func NewQuizRepository(pool *pgxpool.Pool) *QuizRepository {
	return &QuizRepository{pool: pool}
}

const quizSelect = `
	id::text, course_id::text, title, description, status,
	time_limit_seconds, max_attempts, pass_percent, shuffle_questions,
	created_at, updated_at`

func (r *QuizRepository) Create(ctx context.Context, q *domain.Quiz) error {
	const sql = `
		INSERT INTO quizzes (
			course_id, title, description, status,
			time_limit_seconds, max_attempts, pass_percent, shuffle_questions
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, sql,
		q.CourseID, q.Title, q.Description, string(q.Status),
		q.TimeLimitSeconds, q.MaxAttempts, q.PassPercent, q.ShuffleQuestions,
	).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create quiz: %w", err)
	}
	return nil
}

func (r *QuizRepository) GetByID(ctx context.Context, id string) (*domain.Quiz, error) {
	q := fmt.Sprintf(`SELECT %s FROM quizzes WHERE id = $1`, quizSelect)
	quiz, err := scanQuiz(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("quiz not found")
		}
		return nil, fmt.Errorf("get quiz: %w", err)
	}
	return quiz, nil
}

func (r *QuizRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.Quiz, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM quizzes
		WHERE course_id = $1
		ORDER BY created_at ASC`, quizSelect)

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list quizzes: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Quiz, 0)
	for rows.Next() {
		quiz, err := scanQuiz(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *quiz)
	}
	return out, rows.Err()
}

func (r *QuizRepository) Update(ctx context.Context, q *domain.Quiz) (*domain.Quiz, error) {
	const sql = `
		UPDATE quizzes
		SET title = $2, description = $3, time_limit_seconds = $4,
		    max_attempts = $5, pass_percent = $6, shuffle_questions = $7
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql,
		q.ID, q.Title, q.Description, q.TimeLimitSeconds,
		q.MaxAttempts, q.PassPercent, q.ShuffleQuestions,
	)
	if err != nil {
		return nil, fmt.Errorf("update quiz: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("quiz not found")
	}
	return r.GetByID(ctx, q.ID)
}

func (r *QuizRepository) SetStatus(ctx context.Context, id string, status domain.QuizStatus) (*domain.Quiz, error) {
	const sql = `UPDATE quizzes SET status = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sql, id, string(status))
	if err != nil {
		return nil, fmt.Errorf("set quiz status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("quiz not found")
	}
	return r.GetByID(ctx, id)
}

func (r *QuizRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM quizzes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete quiz: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("quiz not found")
	}
	return nil
}

func scanQuiz(row scannable) (*domain.Quiz, error) {
	var q domain.Quiz
	var status string
	if err := row.Scan(
		&q.ID, &q.CourseID, &q.Title, &q.Description, &status,
		&q.TimeLimitSeconds, &q.MaxAttempts, &q.PassPercent, &q.ShuffleQuestions,
		&q.CreatedAt, &q.UpdatedAt,
	); err != nil {
		return nil, err
	}
	q.Status = domain.QuizStatus(status)
	return &q, nil
}

// --- QuizQuestionRepository ---

type QuizQuestionRepository struct {
	pool *pgxpool.Pool
}

func NewQuizQuestionRepository(pool *pgxpool.Pool) *QuizQuestionRepository {
	return &QuizQuestionRepository{pool: pool}
}

const quizQuestionSelect = `
	id::text, quiz_id::text, question_type, prompt, points, position,
	options, correct_answer, created_at, updated_at`

func (r *QuizQuestionRepository) Create(ctx context.Context, q *domain.QuizQuestion) error {
	optionsJSON, err := marshalQuizOptions(q.Options)
	if err != nil {
		return err
	}

	const sql = `
		INSERT INTO quiz_questions (
			quiz_id, question_type, prompt, points, position, options, correct_answer
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id::text, created_at, updated_at`

	err = r.pool.QueryRow(ctx, sql,
		q.QuizID, string(q.QuestionType), q.Prompt, q.Points, q.Position, optionsJSON, q.CorrectAnswer,
	).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("quiz question position conflict")
		}
		return fmt.Errorf("create quiz question: %w", err)
	}
	return nil
}

func (r *QuizQuestionRepository) GetByID(ctx context.Context, id string) (*domain.QuizQuestion, error) {
	q := fmt.Sprintf(`SELECT %s FROM quiz_questions WHERE id = $1`, quizQuestionSelect)
	qq, err := scanQuizQuestion(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("quiz question not found")
		}
		return nil, fmt.Errorf("get quiz question: %w", err)
	}
	return qq, nil
}

func (r *QuizQuestionRepository) ListByQuiz(ctx context.Context, quizID string) ([]domain.QuizQuestion, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM quiz_questions
		WHERE quiz_id = $1
		ORDER BY position ASC`, quizQuestionSelect)

	rows, err := r.pool.Query(ctx, q, quizID)
	if err != nil {
		return nil, fmt.Errorf("list quiz questions: %w", err)
	}
	defer rows.Close()

	out := make([]domain.QuizQuestion, 0)
	for rows.Next() {
		qq, err := scanQuizQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *qq)
	}
	return out, rows.Err()
}

func (r *QuizQuestionRepository) Update(ctx context.Context, q *domain.QuizQuestion) (*domain.QuizQuestion, error) {
	optionsJSON, err := marshalQuizOptions(q.Options)
	if err != nil {
		return nil, err
	}

	const sql = `
		UPDATE quiz_questions
		SET question_type = $2, prompt = $3, points = $4, options = $5, correct_answer = $6
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql,
		q.ID, string(q.QuestionType), q.Prompt, q.Points, optionsJSON, q.CorrectAnswer,
	)
	if err != nil {
		return nil, fmt.Errorf("update quiz question: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("quiz question not found")
	}
	return r.GetByID(ctx, q.ID)
}

func (r *QuizQuestionRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM quiz_questions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete quiz question: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("quiz question not found")
	}
	return nil
}

func (r *QuizQuestionRepository) Reorder(ctx context.Context, quizID string, orderedIDs []string) error {
	return reorderPositions(ctx, r.pool, "quiz_questions", "quiz_id", quizID, orderedIDs)
}

func (r *QuizQuestionRepository) NextPosition(ctx context.Context, quizID string) (int, error) {
	const q = `SELECT COALESCE(MAX(position), -1) + 1 FROM quiz_questions WHERE quiz_id = $1`
	var pos int
	if err := r.pool.QueryRow(ctx, q, quizID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("next quiz question position: %w", err)
	}
	return pos, nil
}

func scanQuizQuestion(row scannable) (*domain.QuizQuestion, error) {
	var q domain.QuizQuestion
	var typ string
	var optionsJSON []byte
	if err := row.Scan(
		&q.ID, &q.QuizID, &typ, &q.Prompt, &q.Points, &q.Position,
		&optionsJSON, &q.CorrectAnswer, &q.CreatedAt, &q.UpdatedAt,
	); err != nil {
		return nil, err
	}
	q.QuestionType = domain.QuestionType(typ)
	opts, err := unmarshalQuizOptions(optionsJSON)
	if err != nil {
		return nil, err
	}
	q.Options = opts
	return &q, nil
}

func marshalQuizOptions(opts []domain.QuizOption) ([]byte, error) {
	if opts == nil {
		opts = []domain.QuizOption{}
	}
	b, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("marshal quiz options: %w", err)
	}
	return b, nil
}

func unmarshalQuizOptions(b []byte) ([]domain.QuizOption, error) {
	if len(b) == 0 {
		return []domain.QuizOption{}, nil
	}
	var opts []domain.QuizOption
	if err := json.Unmarshal(b, &opts); err != nil {
		return nil, fmt.Errorf("unmarshal quiz options: %w", err)
	}
	if opts == nil {
		opts = []domain.QuizOption{}
	}
	return opts, nil
}

// --- QuizAttemptRepository ---

type QuizAttemptRepository struct {
	pool *pgxpool.Pool
}

func NewQuizAttemptRepository(pool *pgxpool.Pool) *QuizAttemptRepository {
	return &QuizAttemptRepository{pool: pool}
}

const quizAttemptSelect = `
	qa.id::text, qa.quiz_id::text, qa.user_id::text, qa.attempt_number, qa.status,
	qa.score_points, qa.max_points, qa.percent, qa.passed,
	qa.started_at, qa.submitted_at, qa.graded_at, qa.created_at, qa.updated_at,
	COALESCE(u.email, ''), COALESCE(u.full_name, '')`

func (r *QuizAttemptRepository) Create(ctx context.Context, a *domain.QuizAttempt) error {
	const sql = `
		INSERT INTO quiz_attempts (
			quiz_id, user_id, attempt_number, status,
			score_points, max_points, percent, passed,
			started_at, submitted_at, graded_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, sql,
		a.QuizID, a.UserID, a.AttemptNumber, string(a.Status),
		a.ScorePoints, a.MaxPoints, a.Percent, a.Passed,
		a.StartedAt, a.SubmittedAt, a.GradedAt,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("quiz attempt number conflict")
		}
		return fmt.Errorf("create quiz attempt: %w", err)
	}
	return nil
}

func (r *QuizAttemptRepository) GetByID(ctx context.Context, id string) (*domain.QuizAttempt, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM quiz_attempts qa
		LEFT JOIN users u ON u.id = qa.user_id
		WHERE qa.id = $1`, quizAttemptSelect)

	a, err := scanQuizAttempt(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("quiz attempt not found")
		}
		return nil, fmt.Errorf("get quiz attempt: %w", err)
	}
	return a, nil
}

func (r *QuizAttemptRepository) ListByQuizUser(ctx context.Context, quizID, userID string) ([]domain.QuizAttempt, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM quiz_attempts qa
		LEFT JOIN users u ON u.id = qa.user_id
		WHERE qa.quiz_id = $1 AND qa.user_id = $2
		ORDER BY qa.attempt_number ASC`, quizAttemptSelect)

	rows, err := r.pool.Query(ctx, q, quizID, userID)
	if err != nil {
		return nil, fmt.Errorf("list quiz attempts: %w", err)
	}
	defer rows.Close()

	out := make([]domain.QuizAttempt, 0)
	for rows.Next() {
		a, err := scanQuizAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *QuizAttemptRepository) CountByQuizUser(ctx context.Context, quizID, userID string) (int, error) {
	const q = `SELECT COUNT(*) FROM quiz_attempts WHERE quiz_id = $1 AND user_id = $2`
	var n int
	if err := r.pool.QueryRow(ctx, q, quizID, userID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count quiz attempts: %w", err)
	}
	return n, nil
}

func (r *QuizAttemptRepository) Update(ctx context.Context, a *domain.QuizAttempt) error {
	const sql = `
		UPDATE quiz_attempts
		SET status = $2, score_points = $3, max_points = $4, percent = $5, passed = $6,
		    submitted_at = $7, graded_at = $8
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql,
		a.ID, string(a.Status), a.ScorePoints, a.MaxPoints, a.Percent, a.Passed,
		a.SubmittedAt, a.GradedAt,
	)
	if err != nil {
		return fmt.Errorf("update quiz attempt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("quiz attempt not found")
	}
	return nil
}

func (r *QuizAttemptRepository) GetInProgress(ctx context.Context, quizID, userID string) (*domain.QuizAttempt, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM quiz_attempts qa
		LEFT JOIN users u ON u.id = qa.user_id
		WHERE qa.quiz_id = $1 AND qa.user_id = $2 AND qa.status = 'in_progress'
		LIMIT 1`, quizAttemptSelect)

	a, err := scanQuizAttempt(r.pool.QueryRow(ctx, q, quizID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("in-progress quiz attempt not found")
		}
		return nil, fmt.Errorf("get in-progress quiz attempt: %w", err)
	}
	return a, nil
}

func (r *QuizAttemptRepository) BestGradedByCourse(ctx context.Context, courseID string) ([]domain.QuizAttempt, error) {
	q := fmt.Sprintf(`
		SELECT DISTINCT ON (qa.quiz_id, qa.user_id) %s
		FROM quiz_attempts qa
		JOIN quizzes quiz ON quiz.id = qa.quiz_id
		LEFT JOIN users u ON u.id = qa.user_id
		WHERE quiz.course_id = $1 AND qa.status = 'graded'
		ORDER BY qa.quiz_id, qa.user_id, qa.percent DESC`, quizAttemptSelect)

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("best graded quiz attempts by course: %w", err)
	}
	defer rows.Close()

	out := make([]domain.QuizAttempt, 0)
	for rows.Next() {
		a, err := scanQuizAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func scanQuizAttempt(row scannable) (*domain.QuizAttempt, error) {
	var a domain.QuizAttempt
	var status string
	if err := row.Scan(
		&a.ID, &a.QuizID, &a.UserID, &a.AttemptNumber, &status,
		&a.ScorePoints, &a.MaxPoints, &a.Percent, &a.Passed,
		&a.StartedAt, &a.SubmittedAt, &a.GradedAt, &a.CreatedAt, &a.UpdatedAt,
		&a.UserEmail, &a.UserFullName,
	); err != nil {
		return nil, err
	}
	a.Status = domain.AttemptStatus(status)
	return &a, nil
}

// --- QuizAttemptAnswerRepository ---

type QuizAttemptAnswerRepository struct {
	pool *pgxpool.Pool
}

func NewQuizAttemptAnswerRepository(pool *pgxpool.Pool) *QuizAttemptAnswerRepository {
	return &QuizAttemptAnswerRepository{pool: pool}
}

const quizAttemptAnswerSelect = `
	id::text, attempt_id::text, question_id::text,
	selected_option_ids, text_answer, is_correct, points_awarded,
	created_at, updated_at`

func (r *QuizAttemptAnswerRepository) Upsert(ctx context.Context, a *domain.QuizAttemptAnswer) error {
	selectedJSON, err := marshalStringSlice(a.SelectedOptionIDs)
	if err != nil {
		return err
	}

	const sql = `
		INSERT INTO quiz_attempt_answers (
			attempt_id, question_id, selected_option_ids, text_answer, is_correct, points_awarded
		) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (attempt_id, question_id) DO UPDATE SET
			selected_option_ids = EXCLUDED.selected_option_ids,
			text_answer = EXCLUDED.text_answer,
			is_correct = EXCLUDED.is_correct,
			points_awarded = EXCLUDED.points_awarded
		RETURNING id::text, created_at, updated_at`

	err = r.pool.QueryRow(ctx, sql,
		a.AttemptID, a.QuestionID, selectedJSON, a.TextAnswer, a.IsCorrect, a.PointsAwarded,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert quiz attempt answer: %w", err)
	}
	return nil
}

func (r *QuizAttemptAnswerRepository) ListByAttempt(ctx context.Context, attemptID string) ([]domain.QuizAttemptAnswer, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM quiz_attempt_answers
		WHERE attempt_id = $1
		ORDER BY created_at ASC`, quizAttemptAnswerSelect)

	rows, err := r.pool.Query(ctx, q, attemptID)
	if err != nil {
		return nil, fmt.Errorf("list quiz attempt answers: %w", err)
	}
	defer rows.Close()

	out := make([]domain.QuizAttemptAnswer, 0)
	for rows.Next() {
		a, err := scanQuizAttemptAnswer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *QuizAttemptAnswerRepository) ReplaceAll(ctx context.Context, attemptID string, answers []domain.QuizAttemptAnswer) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace answers: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM quiz_attempt_answers WHERE attempt_id = $1`, attemptID); err != nil {
		return fmt.Errorf("delete quiz attempt answers: %w", err)
	}

	const sql = `
		INSERT INTO quiz_attempt_answers (
			attempt_id, question_id, selected_option_ids, text_answer, is_correct, points_awarded
		) VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id::text, created_at, updated_at`

	for i := range answers {
		a := &answers[i]
		a.AttemptID = attemptID
		selectedJSON, err := marshalStringSlice(a.SelectedOptionIDs)
		if err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, sql,
			a.AttemptID, a.QuestionID, selectedJSON, a.TextAnswer, a.IsCorrect, a.PointsAwarded,
		).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return apperr.Conflict("duplicate answer for question in attempt")
			}
			return fmt.Errorf("insert quiz attempt answer: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace answers: %w", err)
	}
	return nil
}

func scanQuizAttemptAnswer(row scannable) (*domain.QuizAttemptAnswer, error) {
	var a domain.QuizAttemptAnswer
	var selectedJSON []byte
	if err := row.Scan(
		&a.ID, &a.AttemptID, &a.QuestionID,
		&selectedJSON, &a.TextAnswer, &a.IsCorrect, &a.PointsAwarded,
		&a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	ids, err := unmarshalStringSlice(selectedJSON)
	if err != nil {
		return nil, err
	}
	a.SelectedOptionIDs = ids
	return &a, nil
}

func marshalStringSlice(ids []string) ([]byte, error) {
	if ids == nil {
		ids = []string{}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return nil, fmt.Errorf("marshal string slice: %w", err)
	}
	return b, nil
}

func unmarshalStringSlice(b []byte) ([]string, error) {
	if len(b) == 0 {
		return []string{}, nil
	}
	var ids []string
	if err := json.Unmarshal(b, &ids); err != nil {
		return nil, fmt.Errorf("unmarshal string slice: %w", err)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// --- AssignmentRepository ---

type AssignmentRepository struct {
	pool *pgxpool.Pool
}

func NewAssignmentRepository(pool *pgxpool.Pool) *AssignmentRepository {
	return &AssignmentRepository{pool: pool}
}

const assignmentSelect = `
	id::text, course_id::text, title, description, status,
	max_score, due_at, allow_late, rubric, created_at, updated_at`

func (r *AssignmentRepository) Create(ctx context.Context, a *domain.Assignment) error {
	rubricJSON, err := marshalRubric(a.Rubric)
	if err != nil {
		return err
	}

	const sql = `
		INSERT INTO assignments (
			course_id, title, description, status, max_score, due_at, allow_late, rubric
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, created_at, updated_at`

	err = r.pool.QueryRow(ctx, sql,
		a.CourseID, a.Title, a.Description, string(a.Status),
		a.MaxScore, a.DueAt, a.AllowLate, rubricJSON,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create assignment: %w", err)
	}
	return nil
}

func (r *AssignmentRepository) GetByID(ctx context.Context, id string) (*domain.Assignment, error) {
	q := fmt.Sprintf(`SELECT %s FROM assignments WHERE id = $1`, assignmentSelect)
	a, err := scanAssignment(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("assignment not found")
		}
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	return a, nil
}

func (r *AssignmentRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.Assignment, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM assignments
		WHERE course_id = $1
		ORDER BY created_at ASC`, assignmentSelect)

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Assignment, 0)
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *AssignmentRepository) Update(ctx context.Context, a *domain.Assignment) (*domain.Assignment, error) {
	rubricJSON, err := marshalRubric(a.Rubric)
	if err != nil {
		return nil, err
	}

	const sql = `
		UPDATE assignments
		SET title = $2, description = $3, max_score = $4, due_at = $5, allow_late = $6, rubric = $7
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, sql,
		a.ID, a.Title, a.Description, a.MaxScore, a.DueAt, a.AllowLate, rubricJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("update assignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("assignment not found")
	}
	return r.GetByID(ctx, a.ID)
}

func (r *AssignmentRepository) SetStatus(ctx context.Context, id string, status domain.AssignmentStatus) (*domain.Assignment, error) {
	const sql = `UPDATE assignments SET status = $2 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sql, id, string(status))
	if err != nil {
		return nil, fmt.Errorf("set assignment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("assignment not found")
	}
	return r.GetByID(ctx, id)
}

func (r *AssignmentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("assignment not found")
	}
	return nil
}

func scanAssignment(row scannable) (*domain.Assignment, error) {
	var a domain.Assignment
	var status string
	var rubricJSON []byte
	if err := row.Scan(
		&a.ID, &a.CourseID, &a.Title, &a.Description, &status,
		&a.MaxScore, &a.DueAt, &a.AllowLate, &rubricJSON, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	a.Status = domain.AssignmentStatus(status)
	rubric, err := unmarshalRubric(rubricJSON)
	if err != nil {
		return nil, err
	}
	a.Rubric = rubric
	return &a, nil
}

func marshalRubric(rubric []domain.RubricCriterion) ([]byte, error) {
	if rubric == nil {
		rubric = []domain.RubricCriterion{}
	}
	b, err := json.Marshal(rubric)
	if err != nil {
		return nil, fmt.Errorf("marshal rubric: %w", err)
	}
	return b, nil
}

func unmarshalRubric(b []byte) ([]domain.RubricCriterion, error) {
	if len(b) == 0 {
		return []domain.RubricCriterion{}, nil
	}
	var rubric []domain.RubricCriterion
	if err := json.Unmarshal(b, &rubric); err != nil {
		return nil, fmt.Errorf("unmarshal rubric: %w", err)
	}
	if rubric == nil {
		rubric = []domain.RubricCriterion{}
	}
	return rubric, nil
}

// --- AssignmentSubmissionRepository ---

type AssignmentSubmissionRepository struct {
	pool *pgxpool.Pool
}

func NewAssignmentSubmissionRepository(pool *pgxpool.Pool) *AssignmentSubmissionRepository {
	return &AssignmentSubmissionRepository{pool: pool}
}

const assignmentSubmissionSelect = `
	s.id::text, s.assignment_id::text, s.user_id::text, s.status,
	s.body, s.attachment_url, s.score, s.feedback, s.rubric_scores,
	s.submitted_at, s.graded_at, s.graded_by::text, s.created_at, s.updated_at,
	COALESCE(u.email, ''), COALESCE(u.full_name, '')`

func (r *AssignmentSubmissionRepository) Upsert(ctx context.Context, s *domain.AssignmentSubmission) error {
	scoresJSON, err := marshalRubricScores(s.RubricScores)
	if err != nil {
		return err
	}

	const sql = `
		INSERT INTO assignment_submissions (
			assignment_id, user_id, status, body, attachment_url,
			score, feedback, rubric_scores, submitted_at, graded_at, graded_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (assignment_id, user_id) DO UPDATE SET
			status = EXCLUDED.status,
			body = EXCLUDED.body,
			attachment_url = EXCLUDED.attachment_url,
			score = EXCLUDED.score,
			feedback = EXCLUDED.feedback,
			rubric_scores = EXCLUDED.rubric_scores,
			submitted_at = EXCLUDED.submitted_at,
			graded_at = EXCLUDED.graded_at,
			graded_by = EXCLUDED.graded_by
		RETURNING id::text, created_at, updated_at`

	err = r.pool.QueryRow(ctx, sql,
		s.AssignmentID, s.UserID, string(s.Status), s.Body, s.AttachmentURL,
		s.Score, s.Feedback, scoresJSON, s.SubmittedAt, s.GradedAt, s.GradedBy,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert assignment submission: %w", err)
	}
	return nil
}

func (r *AssignmentSubmissionRepository) GetByAssignmentUser(ctx context.Context, assignmentID, userID string) (*domain.AssignmentSubmission, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM assignment_submissions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.assignment_id = $1 AND s.user_id = $2`, assignmentSubmissionSelect)

	sub, err := scanAssignmentSubmission(r.pool.QueryRow(ctx, q, assignmentID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("assignment submission not found")
		}
		return nil, fmt.Errorf("get assignment submission: %w", err)
	}
	return sub, nil
}

func (r *AssignmentSubmissionRepository) GetByID(ctx context.Context, id string) (*domain.AssignmentSubmission, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM assignment_submissions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.id = $1`, assignmentSubmissionSelect)

	sub, err := scanAssignmentSubmission(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("assignment submission not found")
		}
		return nil, fmt.Errorf("get assignment submission: %w", err)
	}
	return sub, nil
}

func (r *AssignmentSubmissionRepository) ListByAssignment(ctx context.Context, assignmentID string) ([]domain.AssignmentSubmission, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM assignment_submissions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.assignment_id = $1
		ORDER BY s.updated_at DESC`, assignmentSubmissionSelect)

	rows, err := r.pool.Query(ctx, q, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("list assignment submissions: %w", err)
	}
	defer rows.Close()

	out := make([]domain.AssignmentSubmission, 0)
	for rows.Next() {
		sub, err := scanAssignmentSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, rows.Err()
}

func (r *AssignmentSubmissionRepository) ListGradedByCourse(ctx context.Context, courseID string) ([]domain.AssignmentSubmission, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM assignment_submissions s
		JOIN assignments a ON a.id = s.assignment_id
		LEFT JOIN users u ON u.id = s.user_id
		WHERE a.course_id = $1 AND s.status <> 'draft'
		ORDER BY s.updated_at DESC`, assignmentSubmissionSelect)

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list graded submissions by course: %w", err)
	}
	defer rows.Close()

	out := make([]domain.AssignmentSubmission, 0)
	for rows.Next() {
		sub, err := scanAssignmentSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, rows.Err()
}

func scanAssignmentSubmission(row scannable) (*domain.AssignmentSubmission, error) {
	var s domain.AssignmentSubmission
	var status string
	var scoresJSON []byte
	var gradedBy *string
	if err := row.Scan(
		&s.ID, &s.AssignmentID, &s.UserID, &status,
		&s.Body, &s.AttachmentURL, &s.Score, &s.Feedback, &scoresJSON,
		&s.SubmittedAt, &s.GradedAt, &gradedBy, &s.CreatedAt, &s.UpdatedAt,
		&s.UserEmail, &s.UserFullName,
	); err != nil {
		return nil, err
	}
	s.Status = domain.SubmissionStatus(status)
	s.GradedBy = gradedBy
	scores, err := unmarshalRubricScores(scoresJSON)
	if err != nil {
		return nil, err
	}
	s.RubricScores = scores
	return &s, nil
}

func marshalRubricScores(scores map[string]int) ([]byte, error) {
	if scores == nil {
		scores = map[string]int{}
	}
	b, err := json.Marshal(scores)
	if err != nil {
		return nil, fmt.Errorf("marshal rubric scores: %w", err)
	}
	return b, nil
}

func unmarshalRubricScores(b []byte) (map[string]int, error) {
	if len(b) == 0 {
		return map[string]int{}, nil
	}
	var scores map[string]int
	if err := json.Unmarshal(b, &scores); err != nil {
		return nil, fmt.Errorf("unmarshal rubric scores: %w", err)
	}
	if scores == nil {
		scores = map[string]int{}
	}
	return scores, nil
}
