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

type AnnouncementRepository struct {
	pool *pgxpool.Pool
}

func NewAnnouncementRepository(pool *pgxpool.Pool) *AnnouncementRepository {
	return &AnnouncementRepository{pool: pool}
}

const announcementSelect = `
	a.id::text, a.course_id::text, a.author_id::text, a.title, a.body, a.status,
	a.pinned, a.published_at, a.created_at, a.updated_at, u.full_name`

func (r *AnnouncementRepository) Create(ctx context.Context, a *domain.Announcement) error {
	const sql = `
		INSERT INTO announcements (course_id, author_id, title, body, status, pinned)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id::text, created_at, updated_at`
	return r.pool.QueryRow(ctx, sql,
		a.CourseID, a.AuthorID, a.Title, a.Body, string(a.Status), a.Pinned,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AnnouncementRepository) GetByID(ctx context.Context, id string) (*domain.Announcement, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM announcements a
		JOIN users u ON u.id = a.author_id
		WHERE a.id = $1`, announcementSelect)
	a, err := scanAnnouncement(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("announcement not found")
		}
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return a, nil
}

func (r *AnnouncementRepository) ListByCourse(ctx context.Context, courseID string, includeDraft bool) ([]domain.Announcement, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM announcements a
		JOIN users u ON u.id = a.author_id
		WHERE a.course_id = $1`, announcementSelect)
	if !includeDraft {
		q += ` AND a.status = 'published'`
	}
	q += ` ORDER BY a.pinned DESC, COALESCE(a.published_at, a.created_at) DESC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list announcements: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Announcement, 0)
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *AnnouncementRepository) Update(ctx context.Context, a *domain.Announcement) (*domain.Announcement, error) {
	const sql = `
		UPDATE announcements SET title = $2, body = $3, pinned = $4
		WHERE id = $1 AND status = 'draft'`
	tag, err := r.pool.Exec(ctx, sql, a.ID, a.Title, a.Body, a.Pinned)
	if err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.Validation("only draft announcements can be edited")
	}
	return r.GetByID(ctx, a.ID)
}

func (r *AnnouncementRepository) SetStatus(ctx context.Context, id string, status domain.AnnouncementStatus, _ *domain.Announcement) error {
	const sql = `
		UPDATE announcements
		SET status = $2, published_at = CASE WHEN $2 = 'published' THEN NOW() ELSE published_at END
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sql, id, string(status))
	if err != nil {
		return fmt.Errorf("set announcement status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("announcement not found")
	}
	return nil
}

func (r *AnnouncementRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("announcement not found")
	}
	return nil
}

func scanAnnouncement(row pgx.Row) (*domain.Announcement, error) {
	var a domain.Announcement
	var status string
	err := row.Scan(
		&a.ID, &a.CourseID, &a.AuthorID, &a.Title, &a.Body, &status,
		&a.Pinned, &a.PublishedAt, &a.CreatedAt, &a.UpdatedAt, &a.AuthorName,
	)
	if err != nil {
		return nil, err
	}
	a.Status = domain.AnnouncementStatus(status)
	return &a, nil
}

// --- DiscussionThreadRepository ---

type DiscussionThreadRepository struct {
	pool *pgxpool.Pool
}

func NewDiscussionThreadRepository(pool *pgxpool.Pool) *DiscussionThreadRepository {
	return &DiscussionThreadRepository{pool: pool}
}

func (r *DiscussionThreadRepository) Create(ctx context.Context, t *domain.DiscussionThread) error {
	const sql = `
		INSERT INTO discussion_threads (course_id, author_id, title, status)
		VALUES ($1,$2,$3,$4)
		RETURNING id::text, created_at, updated_at`
	return r.pool.QueryRow(ctx, sql, t.CourseID, t.AuthorID, t.Title, string(t.Status)).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *DiscussionThreadRepository) GetByID(ctx context.Context, id string) (*domain.DiscussionThread, error) {
	const q = `
		SELECT t.id::text, t.course_id::text, t.author_id::text, t.title, t.status,
		       t.created_at, t.updated_at, u.full_name,
		       (SELECT COUNT(*)::int FROM discussion_posts p WHERE p.thread_id = t.id)
		FROM discussion_threads t
		JOIN users u ON u.id = t.author_id
		WHERE t.id = $1`
	t, err := scanThread(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("thread not found")
		}
		return nil, fmt.Errorf("get thread: %w", err)
	}
	return t, nil
}

func (r *DiscussionThreadRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.DiscussionThread, error) {
	const q = `
		SELECT t.id::text, t.course_id::text, t.author_id::text, t.title, t.status,
		       t.created_at, t.updated_at, u.full_name,
		       (SELECT COUNT(*)::int FROM discussion_posts p WHERE p.thread_id = t.id)
		FROM discussion_threads t
		JOIN users u ON u.id = t.author_id
		WHERE t.course_id = $1
		ORDER BY t.updated_at DESC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()

	out := make([]domain.DiscussionThread, 0)
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *DiscussionThreadRepository) Update(ctx context.Context, t *domain.DiscussionThread) (*domain.DiscussionThread, error) {
	const sql = `UPDATE discussion_threads SET title = $2 WHERE id = $1 AND author_id = $3`
	tag, err := r.pool.Exec(ctx, sql, t.ID, t.Title, t.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("update thread: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.Forbidden("only the author can edit the thread title")
	}
	return r.GetByID(ctx, t.ID)
}

func (r *DiscussionThreadRepository) SetStatus(ctx context.Context, id string, status domain.ThreadStatus) (*domain.DiscussionThread, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE discussion_threads SET status = $2 WHERE id = $1`, id, string(status))
	if err != nil {
		return nil, fmt.Errorf("set thread status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("thread not found")
	}
	return r.GetByID(ctx, id)
}

func scanThread(row pgx.Row) (*domain.DiscussionThread, error) {
	var t domain.DiscussionThread
	var status string
	err := row.Scan(
		&t.ID, &t.CourseID, &t.AuthorID, &t.Title, &status,
		&t.CreatedAt, &t.UpdatedAt, &t.AuthorName, &t.PostCount,
	)
	if err != nil {
		return nil, err
	}
	t.Status = domain.ThreadStatus(status)
	return &t, nil
}

// --- DiscussionPostRepository ---

type DiscussionPostRepository struct {
	pool *pgxpool.Pool
}

func NewDiscussionPostRepository(pool *pgxpool.Pool) *DiscussionPostRepository {
	return &DiscussionPostRepository{pool: pool}
}

func (r *DiscussionPostRepository) Create(ctx context.Context, p *domain.DiscussionPost) error {
	const sql = `
		INSERT INTO discussion_posts (thread_id, author_id, parent_id, body)
		VALUES ($1,$2,$3,$4)
		RETURNING id::text, created_at, updated_at`
	err := r.pool.QueryRow(ctx, sql, p.ThreadID, p.AuthorID, p.ParentID, p.Body).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	_, _ = r.pool.Exec(ctx, `UPDATE discussion_threads SET updated_at = NOW() WHERE id = $1`, p.ThreadID)
	return nil
}

func (r *DiscussionPostRepository) GetByID(ctx context.Context, id string) (*domain.DiscussionPost, error) {
	const q = `
		SELECT p.id::text, p.thread_id::text, p.author_id::text, p.parent_id::text,
		       p.body, p.created_at, p.updated_at, u.full_name
		FROM discussion_posts p
		JOIN users u ON u.id = p.author_id
		WHERE p.id = $1`
	post, err := scanPost(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("post not found")
		}
		return nil, fmt.Errorf("get post: %w", err)
	}
	return post, nil
}

func (r *DiscussionPostRepository) ListByThread(ctx context.Context, threadID string) ([]domain.DiscussionPost, error) {
	const q = `
		SELECT p.id::text, p.thread_id::text, p.author_id::text, p.parent_id::text,
		       p.body, p.created_at, p.updated_at, u.full_name
		FROM discussion_posts p
		JOIN users u ON u.id = p.author_id
		WHERE p.thread_id = $1
		ORDER BY p.created_at ASC`

	rows, err := r.pool.Query(ctx, q, threadID)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	out := make([]domain.DiscussionPost, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *DiscussionPostRepository) Update(ctx context.Context, p *domain.DiscussionPost) (*domain.DiscussionPost, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE discussion_posts SET body = $2 WHERE id = $1 AND author_id = $3`, p.ID, p.Body, p.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.Forbidden("only the author can edit this post")
	}
	return r.GetByID(ctx, p.ID)
}

func (r *DiscussionPostRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM discussion_posts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("post not found")
	}
	return nil
}

func (r *DiscussionPostRepository) ListParticipantIDs(ctx context.Context, threadID string) ([]string, error) {
	const q = `SELECT DISTINCT author_id::text FROM discussion_posts WHERE thread_id = $1`
	rows, err := r.pool.Query(ctx, q, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func scanPost(row pgx.Row) (*domain.DiscussionPost, error) {
	var p domain.DiscussionPost
	err := row.Scan(
		&p.ID, &p.ThreadID, &p.AuthorID, &p.ParentID,
		&p.Body, &p.CreatedAt, &p.UpdatedAt, &p.AuthorName,
	)
	return &p, err
}

// --- DMConversationRepository ---

type DMConversationRepository struct {
	pool *pgxpool.Pool
}

func NewDMConversationRepository(pool *pgxpool.Pool) *DMConversationRepository {
	return &DMConversationRepository{pool: pool}
}

func (r *DMConversationRepository) GetOrCreate(ctx context.Context, userAID, userBID string) (*domain.DMConversation, error) {
	a, b := orderPair(userAID, userBID)
	const sql = `
		INSERT INTO dm_conversations (user_a_id, user_b_id)
		VALUES ($1,$2)
		ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET updated_at = dm_conversations.updated_at
		RETURNING id::text, user_a_id::text, user_b_id::text, created_at, updated_at`
	var c domain.DMConversation
	err := r.pool.QueryRow(ctx, sql, a, b).Scan(&c.ID, &c.UserAID, &c.UserBID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get or create conversation: %w", err)
	}
	return &c, nil
}

func (r *DMConversationRepository) GetByID(ctx context.Context, id string) (*domain.DMConversation, error) {
	const q = `SELECT id::text, user_a_id::text, user_b_id::text, created_at, updated_at FROM dm_conversations WHERE id = $1`
	var c domain.DMConversation
	err := r.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.UserAID, &c.UserBID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("conversation not found")
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return &c, nil
}

func (r *DMConversationRepository) ListForUser(ctx context.Context, userID string) ([]domain.DMConversation, error) {
	const q = `
		SELECT c.id::text, c.user_a_id::text, c.user_b_id::text, c.created_at, c.updated_at,
		       CASE WHEN c.user_a_id = $1 THEN c.user_b_id ELSE c.user_a_id END::text AS other_id,
		       COALESCE(ou.full_name, '') AS other_name,
		       COALESCE((SELECT body FROM dm_messages m WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1), '') AS last_msg,
		       COALESCE((SELECT COUNT(*)::int FROM dm_messages m WHERE m.conversation_id = c.id AND m.sender_id <> $1 AND m.read_at IS NULL), 0) AS unread
		FROM dm_conversations c
		LEFT JOIN users ou ON ou.id = CASE WHEN c.user_a_id = $1 THEN c.user_b_id ELSE c.user_a_id END
		WHERE c.user_a_id = $1 OR c.user_b_id = $1
		ORDER BY c.updated_at DESC`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	out := make([]domain.DMConversation, 0)
	for rows.Next() {
		var c domain.DMConversation
		err := rows.Scan(
			&c.ID, &c.UserAID, &c.UserBID, &c.CreatedAt, &c.UpdatedAt,
			&c.OtherUserID, &c.OtherUserName, &c.LastMessage, &c.UnreadCount,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func orderPair(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// --- DMMessageRepository ---

type DMMessageRepository struct {
	pool *pgxpool.Pool
}

func NewDMMessageRepository(pool *pgxpool.Pool) *DMMessageRepository {
	return &DMMessageRepository{pool: pool}
}

func (r *DMMessageRepository) Create(ctx context.Context, m *domain.DMMessage) error {
	const sql = `
		INSERT INTO dm_messages (conversation_id, sender_id, body)
		VALUES ($1,$2,$3)
		RETURNING id::text, created_at`
	err := r.pool.QueryRow(ctx, sql, m.ConversationID, m.SenderID, m.Body).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	_, _ = r.pool.Exec(ctx, `UPDATE dm_conversations SET updated_at = NOW() WHERE id = $1`, m.ConversationID)
	return nil
}

func (r *DMMessageRepository) ListByConversation(ctx context.Context, conversationID string, limit, offset int) ([]domain.DMMessage, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM dm_messages WHERE conversation_id = $1`, conversationID).Scan(&total); err != nil {
		return nil, 0, err
	}
	const q = `
		SELECT id::text, conversation_id::text, sender_id::text, body, read_at, created_at
		FROM dm_messages WHERE conversation_id = $1
		ORDER BY created_at ASC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, conversationID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	out := make([]domain.DMMessage, 0)
	for rows.Next() {
		var m domain.DMMessage
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.ReadAt, &m.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

func (r *DMMessageRepository) MarkRead(ctx context.Context, conversationID, readerID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE dm_messages SET read_at = NOW()
		WHERE conversation_id = $1 AND sender_id <> $2 AND read_at IS NULL`, conversationID, readerID)
	return err
}

// --- NotificationRepository ---

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Enqueue(ctx context.Context, n *domain.Notification) error {
	meta, err := json.Marshal(n.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	const sql = `
		INSERT INTO notification_outbox (user_id, channel, event_type, subject, body, payload, status, sent_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7, CASE WHEN $7 = 'sent' THEN NOW() ELSE NULL END)
		RETURNING id::text, created_at`
	return r.pool.QueryRow(ctx, sql,
		n.UserID, string(n.Channel), n.EventType, n.Subject, n.Body, meta, string(n.Status),
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *NotificationRepository) ListForUser(ctx context.Context, userID string, filter domain.NotificationListFilter) ([]domain.Notification, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	where := `user_id = $1 AND channel = 'in_app'`
	args := []any{userID}
	if filter.UnreadOnly {
		where += ` AND read_at IS NULL`
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notification_outbox WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(args, filter.PerPage, offset)
	q := fmt.Sprintf(`
		SELECT id::text, user_id::text, channel, event_type, subject, body, payload,
		       status, read_at, created_at, sent_at
		FROM notification_outbox WHERE %s
		ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, len(listArgs)-1, len(listArgs))

	rows, err := r.pool.Query(ctx, q, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Notification, 0)
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *n)
	}
	return out, total, rows.Err()
}

func (r *NotificationRepository) GetByID(ctx context.Context, id, userID string) (*domain.Notification, error) {
	const q = `
		SELECT id::text, user_id::text, channel, event_type, subject, body, payload,
		       status, read_at, created_at, sent_at
		FROM notification_outbox WHERE id = $1 AND user_id = $2`
	n, err := scanNotification(r.pool.QueryRow(ctx, q, id, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("notification not found")
		}
		return nil, fmt.Errorf("get notification: %w", err)
	}
	return n, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_outbox SET read_at = NOW()
		WHERE id = $1 AND user_id = $2 AND channel = 'in_app'`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("notification not found")
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notification_outbox SET read_at = NOW()
		WHERE user_id = $1 AND channel = 'in_app' AND read_at IS NULL`, userID)
	return err
}

func scanNotification(row pgx.Row) (*domain.Notification, error) {
	var n domain.Notification
	var channel, status string
	var payloadJSON []byte
	err := row.Scan(
		&n.ID, &n.UserID, &channel, &n.EventType, &n.Subject, &n.Body, &payloadJSON,
		&status, &n.ReadAt, &n.CreatedAt, &n.SentAt,
	)
	if err != nil {
		return nil, err
	}
	n.Channel = domain.NotificationChannel(channel)
	n.Status = domain.NotificationStatus(status)
	n.Payload = map[string]string{}
	if len(payloadJSON) > 0 {
		_ = json.Unmarshal(payloadJSON, &n.Payload)
	}
	return &n, nil
}
