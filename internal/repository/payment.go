package repository

import (
	"context"

	"github.com/asnakech/asnakech-servers/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, o *domain.Order) error
	ListByUser(ctx context.Context, userID string, filter domain.OrderListFilter) ([]domain.Order, int64, error)
	ListByCourse(ctx context.Context, courseID string, filter domain.OrderListFilter) ([]domain.Order, int64, error)
}

type CouponRepository interface {
	Create(ctx context.Context, c *domain.Coupon) error
	GetByID(ctx context.Context, id string) (*domain.Coupon, error)
	GetByCode(ctx context.Context, code string) (*domain.Coupon, error)
	List(ctx context.Context, courseID *string) ([]domain.Coupon, error)
	IncrementUses(ctx context.Context, id string) error
	Revoke(ctx context.Context, id string) error
}

type PaymentWebhookRepository interface {
	Record(ctx context.Context, e *domain.PaymentWebhookEvent) (bool, error)
}
