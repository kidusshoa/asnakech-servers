package domain

import "time"

// PaymentProvider identifies the checkout backend.
type PaymentProvider string

const (
	PaymentProviderManual PaymentProvider = "manual"
	PaymentProviderStripe PaymentProvider = "stripe"
	PaymentProviderChapa  PaymentProvider = "chapa"
)

// OrderStatus tracks checkout lifecycle.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFailed    OrderStatus = "failed"
	OrderStatusRefunded  OrderStatus = "refunded"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// CouponDiscountType is how a coupon reduces price.
type CouponDiscountType string

const (
	CouponDiscountPercent CouponDiscountType = "percent"
	CouponDiscountFixed   CouponDiscountType = "fixed"
)

// Coupon is a redeemable discount code.
type Coupon struct {
	ID            string
	Code          string
	DiscountType  CouponDiscountType
	DiscountValue int
	Currency      string
	CourseID      *string
	MaxUses       *int
	UsesCount     int
	ValidFrom     *time.Time
	ValidUntil    *time.Time
	CreatedBy     string
	RevokedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Order is a course purchase attempt.
type Order struct {
	ID             string
	UserID         string
	CourseID       string
	Status         OrderStatus
	AmountCents    int
	DiscountCents  int
	TotalCents     int
	Currency       string
	CouponID       *string
	Provider       PaymentProvider
	ProviderRef    string
	IdempotencyKey string
	EnrollmentID   *string
	PaidAt         *time.Time
	FailedAt       *time.Time
	RefundedAt     *time.Time
	CancelledAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// Optional joined fields
	UserEmail    string
	UserFullName string
	CourseTitle  string
	CourseSlug   string
	CouponCode   string
}

// PaymentWebhookEvent records processed provider callbacks.
type PaymentWebhookEvent struct {
	ID          string
	Provider    PaymentProvider
	EventID     string
	EventType   string
	Payload     []byte
	OrderID     *string
	ProcessedAt time.Time
	CreatedAt   time.Time
}

// OrderListFilter paginates order listings.
type OrderListFilter struct {
	Page    int
	PerPage int
	Status  OrderStatus
}

// CreateCouponInput is service input for new coupons.
type CreateCouponInput struct {
	Code          string
	DiscountType  CouponDiscountType
	DiscountValue int
	Currency      string
	CourseID      *string
	MaxUses       *int
	ValidFrom     *time.Time
	ValidUntil    *time.Time
}
