package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	const q = `
		INSERT INTO orders (
			user_id, course_id, status, amount_cents, discount_cents, total_cents,
			currency, coupon_id, provider, provider_ref, idempotency_key
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id::text, created_at, updated_at`

	var idempotency *string
	if strings.TrimSpace(o.IdempotencyKey) != "" {
		idempotency = &o.IdempotencyKey
	}
	var providerRef *string
	if strings.TrimSpace(o.ProviderRef) != "" {
		providerRef = &o.ProviderRef
	}

	err := r.pool.QueryRow(ctx, q,
		o.UserID, o.CourseID, string(o.Status), o.AmountCents, o.DiscountCents, o.TotalCents,
		o.Currency, o.CouponID, string(o.Provider), providerRef, idempotency,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("duplicate idempotency key")
		}
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	const q = `
		SELECT o.id::text, o.user_id::text, o.course_id::text, o.status,
		       o.amount_cents, o.discount_cents, o.total_cents, o.currency,
		       o.coupon_id::text, o.provider, COALESCE(o.provider_ref, ''),
		       COALESCE(o.idempotency_key, ''), o.enrollment_id::text,
		       o.paid_at, o.failed_at, o.refunded_at, o.cancelled_at,
		       o.created_at, o.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, ''),
		       COALESCE(cp.code, '')
		FROM orders o
		LEFT JOIN users u ON u.id = o.user_id
		LEFT JOIN courses c ON c.id = o.course_id
		LEFT JOIN coupons cp ON cp.id = o.coupon_id
		WHERE o.id = $1`

	o, err := scanOrder(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("order not found")
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

func (r *OrderRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error) {
	const q = `
		SELECT o.id::text, o.user_id::text, o.course_id::text, o.status,
		       o.amount_cents, o.discount_cents, o.total_cents, o.currency,
		       o.coupon_id::text, o.provider, COALESCE(o.provider_ref, ''),
		       COALESCE(o.idempotency_key, ''), o.enrollment_id::text,
		       o.paid_at, o.failed_at, o.refunded_at, o.cancelled_at,
		       o.created_at, o.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, ''),
		       COALESCE(cp.code, '')
		FROM orders o
		LEFT JOIN users u ON u.id = o.user_id
		LEFT JOIN courses c ON c.id = o.course_id
		LEFT JOIN coupons cp ON cp.id = o.coupon_id
		WHERE o.idempotency_key = $1`

	o, err := scanOrder(r.pool.QueryRow(ctx, q, key))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("order not found")
		}
		return nil, fmt.Errorf("get order by idempotency key: %w", err)
	}
	return o, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, o *domain.Order) error {
	const q = `
		UPDATE orders
		SET status = $2, provider_ref = $3, enrollment_id = $4,
		    paid_at = $5, failed_at = $6, refunded_at = $7, cancelled_at = $8
		WHERE id = $1`

	var providerRef *string
	if strings.TrimSpace(o.ProviderRef) != "" {
		providerRef = &o.ProviderRef
	}

	tag, err := r.pool.Exec(ctx, q,
		o.ID, string(o.Status), providerRef, o.EnrollmentID,
		o.PaidAt, o.FailedAt, o.RefundedAt, o.CancelledAt,
	)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("order not found")
	}
	return nil
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID string, filter domain.OrderListFilter) ([]domain.Order, int64, error) {
	page, perPage := normalizeOrderPage(filter.Page, filter.PerPage)
	offset := (page - 1) * perPage

	where := "WHERE o.user_id = $1"
	args := []any{userID}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		where += fmt.Sprintf(" AND o.status = $%d", len(args))
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM orders o " + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user orders: %w", err)
	}

	listQ := `
		SELECT o.id::text, o.user_id::text, o.course_id::text, o.status,
		       o.amount_cents, o.discount_cents, o.total_cents, o.currency,
		       o.coupon_id::text, o.provider, COALESCE(o.provider_ref, ''),
		       COALESCE(o.idempotency_key, ''), o.enrollment_id::text,
		       o.paid_at, o.failed_at, o.refunded_at, o.cancelled_at,
		       o.created_at, o.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, ''),
		       COALESCE(cp.code, '')
		FROM orders o
		LEFT JOIN users u ON u.id = o.user_id
		LEFT JOIN courses c ON c.id = o.course_id
		LEFT JOIN coupons cp ON cp.id = o.coupon_id
		` + where + fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list user orders: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *o)
	}
	return out, total, rows.Err()
}

func (r *OrderRepository) ListByCourse(ctx context.Context, courseID string, filter domain.OrderListFilter) ([]domain.Order, int64, error) {
	page, perPage := normalizeOrderPage(filter.Page, filter.PerPage)
	offset := (page - 1) * perPage

	where := "WHERE o.course_id = $1"
	args := []any{courseID}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		where += fmt.Sprintf(" AND o.status = $%d", len(args))
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM orders o " + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count course orders: %w", err)
	}

	listQ := `
		SELECT o.id::text, o.user_id::text, o.course_id::text, o.status,
		       o.amount_cents, o.discount_cents, o.total_cents, o.currency,
		       o.coupon_id::text, o.provider, COALESCE(o.provider_ref, ''),
		       COALESCE(o.idempotency_key, ''), o.enrollment_id::text,
		       o.paid_at, o.failed_at, o.refunded_at, o.cancelled_at,
		       o.created_at, o.updated_at,
		       COALESCE(u.email, ''), COALESCE(u.full_name, ''),
		       COALESCE(c.title, ''), COALESCE(c.slug, ''),
		       COALESCE(cp.code, '')
		FROM orders o
		LEFT JOIN users u ON u.id = o.user_id
		LEFT JOIN courses c ON c.id = o.course_id
		LEFT JOIN coupons cp ON cp.id = o.coupon_id
		` + where + fmt.Sprintf(" ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list course orders: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *o)
	}
	return out, total, rows.Err()
}

type CouponRepository struct {
	pool *pgxpool.Pool
}

func NewCouponRepository(pool *pgxpool.Pool) *CouponRepository {
	return &CouponRepository{pool: pool}
}

func (r *CouponRepository) Create(ctx context.Context, c *domain.Coupon) error {
	const q = `
		INSERT INTO coupons (
			code, discount_type, discount_value, currency, course_id,
			max_uses, valid_from, valid_until, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id::text, uses_count, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		c.Code, string(c.DiscountType), c.DiscountValue, nullIfEmpty(c.Currency), c.CourseID,
		c.MaxUses, c.ValidFrom, c.ValidUntil, c.CreatedBy,
	).Scan(&c.ID, &c.UsesCount, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("coupon code already exists")
		}
		return fmt.Errorf("create coupon: %w", err)
	}
	return nil
}

func (r *CouponRepository) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	const q = `
		SELECT id::text, code, discount_type, discount_value, COALESCE(currency, ''),
		       course_id::text, max_uses, uses_count, valid_from, valid_until,
		       created_by::text, revoked_at, created_at, updated_at
		FROM coupons WHERE id = $1`

	c, err := scanCoupon(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("coupon not found")
		}
		return nil, fmt.Errorf("get coupon: %w", err)
	}
	return c, nil
}

func (r *CouponRepository) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	const q = `
		SELECT id::text, code, discount_type, discount_value, COALESCE(currency, ''),
		       course_id::text, max_uses, uses_count, valid_from, valid_until,
		       created_by::text, revoked_at, created_at, updated_at
		FROM coupons WHERE lower(code) = lower($1)`

	c, err := scanCoupon(r.pool.QueryRow(ctx, q, strings.TrimSpace(code)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("coupon not found")
		}
		return nil, fmt.Errorf("get coupon by code: %w", err)
	}
	return c, nil
}

func (r *CouponRepository) List(ctx context.Context, courseID *string) ([]domain.Coupon, error) {
	const q = `
		SELECT id::text, code, discount_type, discount_value, COALESCE(currency, ''),
		       course_id::text, max_uses, uses_count, valid_from, valid_until,
		       created_by::text, revoked_at, created_at, updated_at
		FROM coupons
		WHERE ($1::uuid IS NULL OR course_id = $1 OR course_id IS NULL)
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list coupons: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Coupon, 0)
	for rows.Next() {
		c, err := scanCoupon(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *CouponRepository) IncrementUses(ctx context.Context, id string) error {
	const q = `UPDATE coupons SET uses_count = uses_count + 1 WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("increment coupon uses: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("coupon not found")
	}
	return nil
}

func (r *CouponRepository) Revoke(ctx context.Context, id string) error {
	const q = `UPDATE coupons SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("revoke coupon: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("coupon not found or already revoked")
	}
	return nil
}

type PaymentWebhookRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentWebhookRepository(pool *pgxpool.Pool) *PaymentWebhookRepository {
	return &PaymentWebhookRepository{pool: pool}
}

// Record inserts the event; returns false if duplicate (already processed).
func (r *PaymentWebhookRepository) Record(ctx context.Context, e *domain.PaymentWebhookEvent) (bool, error) {
	const q = `
		INSERT INTO payment_webhook_events (provider, event_id, event_type, payload, order_id)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id::text, processed_at, created_at`

	err := r.pool.QueryRow(ctx, q,
		string(e.Provider), e.EventID, e.EventType, e.Payload, e.OrderID,
	).Scan(&e.ID, &e.ProcessedAt, &e.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, nil
		}
		return false, fmt.Errorf("record webhook event: %w", err)
	}
	return true, nil
}

func scanOrder(row pgx.Row) (*domain.Order, error) {
	var o domain.Order
	var status, provider string
	var couponID, enrollmentID *string
	err := row.Scan(
		&o.ID, &o.UserID, &o.CourseID, &status,
		&o.AmountCents, &o.DiscountCents, &o.TotalCents, &o.Currency,
		&couponID, &provider, &o.ProviderRef, &o.IdempotencyKey, &enrollmentID,
		&o.PaidAt, &o.FailedAt, &o.RefundedAt, &o.CancelledAt,
		&o.CreatedAt, &o.UpdatedAt,
		&o.UserEmail, &o.UserFullName, &o.CourseTitle, &o.CourseSlug, &o.CouponCode,
	)
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderStatus(status)
	o.Provider = domain.PaymentProvider(provider)
	o.CouponID = couponID
	o.EnrollmentID = enrollmentID
	return &o, nil
}

func scanCoupon(row pgx.Row) (*domain.Coupon, error) {
	var c domain.Coupon
	var discountType string
	var courseID *string
	err := row.Scan(
		&c.ID, &c.Code, &discountType, &c.DiscountValue, &c.Currency,
		&courseID, &c.MaxUses, &c.UsesCount, &c.ValidFrom, &c.ValidUntil,
		&c.CreatedBy, &c.RevokedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.DiscountType = domain.CouponDiscountType(discountType)
	c.CourseID = courseID
	return &c, nil
}

func normalizeOrderPage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func nullIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
