package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/payment"
	"github.com/asnakech/asnakech-servers/internal/repository"
)

type PaymentService struct {
	courses     repository.CourseRepository
	orders      repository.OrderRepository
	coupons     repository.CouponRepository
	webhooks    repository.PaymentWebhookRepository
	enrollments *EnrollmentService
	providers   *payment.Registry
}

func NewPaymentService(
	courses repository.CourseRepository,
	orders repository.OrderRepository,
	coupons repository.CouponRepository,
	webhooks repository.PaymentWebhookRepository,
	enrollments *EnrollmentService,
	providers *payment.Registry,
) *PaymentService {
	return &PaymentService{
		courses:     courses,
		orders:      orders,
		coupons:     coupons,
		webhooks:    webhooks,
		enrollments: enrollments,
		providers:   providers,
	}
}

type CheckoutResult struct {
	Order          *domain.Order
	CheckoutURL    string
	ProviderRef    string
	ProviderMeta   map[string]string
}

func (s *PaymentService) CreateCheckout(
	ctx context.Context,
	actorID, courseID, couponCode, idempotencyKey string,
	provider domain.PaymentProvider,
) (*CheckoutResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey != "" {
		if existing, err := s.orders.GetByIdempotencyKey(ctx, idempotencyKey); err == nil {
			return s.checkoutFromExisting(ctx, existing)
		} else if !apperr.IsNotFound(err) {
			return nil, err
		}
	}

	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if course.Status != domain.CourseStatusPublished {
		return nil, apperr.Validation("only published courses can be purchased")
	}
	if course.TeacherID == actorID {
		return nil, apperr.Validation("teachers cannot purchase their own course")
	}
	if course.PriceCents <= 0 {
		return nil, apperr.Validation("course is free — enroll directly")
	}

	if active, err := s.enrollments.HasActiveEnrollment(ctx, courseID, actorID); err != nil {
		return nil, err
	} else if active {
		return nil, apperr.Conflict("already enrolled in this course")
	}

	amount := course.PriceCents
	currency := course.Currency
	if currency == "" {
		currency = "ETB"
	}
	discount := 0
	var couponID *string

	couponCode = strings.TrimSpace(couponCode)
	if couponCode != "" {
		coupon, err := s.coupons.GetByCode(ctx, couponCode)
		if err != nil {
			return nil, err
		}
		if err := validateCoupon(coupon, courseID, currency); err != nil {
			return nil, err
		}
		discount = computeDiscount(coupon, amount)
		couponID = &coupon.ID
	}

	total := amount - discount
	if total < 0 {
		total = 0
	}

	if provider == "" {
		provider = s.providers.DefaultProvider()
	}
	prov := s.providers.For(provider)

	order := &domain.Order{
		UserID:         actorID,
		CourseID:       courseID,
		Status:         domain.OrderStatusPending,
		AmountCents:    amount,
		DiscountCents:  discount,
		TotalCents:     total,
		Currency:       currency,
		CouponID:       couponID,
		Provider:       prov.Type(),
		IdempotencyKey: idempotencyKey,
	}
	if err := s.orders.Create(ctx, order); err != nil {
		return nil, err
	}

	session, err := prov.CreateCheckout(ctx, order)
	if err != nil {
		return nil, err
	}
	if session.ExternalID != "" {
		order.ProviderRef = session.ExternalID
		if err := s.orders.UpdateStatus(ctx, order); err != nil {
			return nil, err
		}
	}

	full, err := s.orders.GetByID(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{
		Order:        full,
		CheckoutURL:  session.URL,
		ProviderRef:  session.ExternalID,
		ProviderMeta: session.Metadata,
	}, nil
}

func (s *PaymentService) checkoutFromExisting(ctx context.Context, order *domain.Order) (*CheckoutResult, error) {
	if order.Status != domain.OrderStatusPending {
		return nil, apperr.Conflict("idempotency key already used for a non-pending order")
	}
	prov := s.providers.For(order.Provider)
	session, err := prov.CreateCheckout(ctx, order)
	if err != nil {
		return nil, err
	}
	return &CheckoutResult{
		Order:        order,
		CheckoutURL:  session.URL,
		ProviderRef:  session.ExternalID,
		ProviderMeta: session.Metadata,
	}, nil
}

func (s *PaymentService) ConfirmManual(ctx context.Context, actorID, orderID string) (*domain.Order, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != actorID {
		return nil, apperr.Forbidden("cannot confirm another user's order")
	}
	if order.Provider != domain.PaymentProviderManual {
		return nil, apperr.Validation("manual confirm only applies to manual provider orders")
	}
	if order.Status != domain.OrderStatusPending {
		return nil, apperr.Conflict("order is not pending")
	}
	return s.applyPaymentSuccess(ctx, order, "manual-confirm-"+order.ID, "manual.confirm")
}

func (s *PaymentService) HandleWebhook(
	ctx context.Context,
	provider domain.PaymentProvider,
	headers http.Header,
	body []byte,
) (*domain.Order, error) {
	prov := s.providers.For(provider)
	result, err := prov.ParseWebhook(ctx, headers, body)
	if err != nil {
		return nil, err
	}

	event := &domain.PaymentWebhookEvent{
		Provider:  prov.Type(),
		EventID:   result.EventID,
		EventType: result.EventType,
		Payload:   body,
		OrderID:   &result.OrderID,
	}
	inserted, err := s.webhooks.Record(ctx, event)
	if err != nil {
		return nil, err
	}
	if !inserted {
		order, err := s.orders.GetByID(ctx, result.OrderID)
		if err != nil {
			return nil, err
		}
		return order, nil
	}

	order, err := s.orders.GetByID(ctx, result.OrderID)
	if err != nil {
		return nil, err
	}
	if result.ProviderRef != "" {
		order.ProviderRef = result.ProviderRef
	}

	switch result.Status {
	case domain.OrderStatusPaid:
		return s.applyPaymentSuccess(ctx, order, result.EventID, result.EventType)
	case domain.OrderStatusRefunded:
		return s.applyRefund(ctx, order)
	case domain.OrderStatusFailed:
		return s.markFailed(ctx, order)
	case domain.OrderStatusCancelled:
		return s.markCancelled(ctx, order)
	default:
		return order, nil
	}
}

func (s *PaymentService) GetOrder(ctx context.Context, actorID, orderID string, platformAdmin bool) (*domain.Order, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != actorID && !platformAdmin {
		course, err := s.courses.GetByID(ctx, order.CourseID)
		if err != nil {
			return nil, err
		}
		if course.TeacherID != actorID {
			return nil, apperr.Forbidden("cannot view this order")
		}
	}
	return order, nil
}

func (s *PaymentService) ListMine(ctx context.Context, actorID string, filter domain.OrderListFilter) ([]domain.Order, int64, error) {
	return s.orders.ListByUser(ctx, actorID, filter)
}

func (s *PaymentService) ListForCourse(ctx context.Context, actorID, courseID string, filter domain.OrderListFilter, platformAdmin bool) ([]domain.Order, int64, error) {
	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, 0, err
	}
	if !platformAdmin && course.TeacherID != actorID {
		return nil, 0, apperr.Forbidden("only the course teacher or an admin can list orders")
	}
	return s.orders.ListByCourse(ctx, courseID, filter)
}

func (s *PaymentService) Refund(ctx context.Context, actorID, orderID string, platformAdmin bool) (*domain.Order, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	course, err := s.courses.GetByID(ctx, order.CourseID)
	if err != nil {
		return nil, err
	}
	if !platformAdmin && course.TeacherID != actorID {
		return nil, apperr.Forbidden("only the course teacher or an admin can refund orders")
	}
	if order.Status != domain.OrderStatusPaid {
		return nil, apperr.Validation("only paid orders can be refunded")
	}
	return s.applyRefund(ctx, order)
}

func (s *PaymentService) CreateCoupon(ctx context.Context, actorID string, in domain.CreateCouponInput, platformAdmin bool) (*domain.Coupon, error) {
	if !platformAdmin {
		return nil, apperr.Forbidden("only platform admins can create coupons")
	}
	code := strings.TrimSpace(in.Code)
	if code == "" {
		return nil, apperr.Validation("code is required")
	}
	if in.DiscountType != domain.CouponDiscountPercent && in.DiscountType != domain.CouponDiscountFixed {
		return nil, apperr.Validation("discount_type must be percent or fixed")
	}
	if in.DiscountValue <= 0 {
		return nil, apperr.Validation("discount_value must be > 0")
	}
	if in.DiscountType == domain.CouponDiscountPercent && in.DiscountValue > 100 {
		return nil, apperr.Validation("percent discount cannot exceed 100")
	}
	if in.CourseID != nil {
		if _, err := s.courses.GetByID(ctx, *in.CourseID); err != nil {
			return nil, err
		}
	}
	if in.MaxUses != nil && *in.MaxUses <= 0 {
		return nil, apperr.Validation("max_uses must be > 0 when set")
	}

	coupon := &domain.Coupon{
		Code:          strings.ToUpper(code),
		DiscountType:  in.DiscountType,
		DiscountValue: in.DiscountValue,
		Currency:      strings.TrimSpace(in.Currency),
		CourseID:      in.CourseID,
		MaxUses:       in.MaxUses,
		ValidFrom:     in.ValidFrom,
		ValidUntil:    in.ValidUntil,
		CreatedBy:     actorID,
	}
	if err := s.coupons.Create(ctx, coupon); err != nil {
		return nil, err
	}
	return s.coupons.GetByID(ctx, coupon.ID)
}

func (s *PaymentService) ListCoupons(ctx context.Context, actorID string, courseID *string, platformAdmin bool) ([]domain.Coupon, error) {
	if !platformAdmin {
		if courseID == nil {
			return nil, apperr.Forbidden("only platform admins can list all coupons")
		}
		course, err := s.courses.GetByID(ctx, *courseID)
		if err != nil {
			return nil, err
		}
		if course.TeacherID != actorID {
			return nil, apperr.Forbidden("cannot list coupons for this course")
		}
	}
	return s.coupons.List(ctx, courseID)
}

func (s *PaymentService) RevokeCoupon(ctx context.Context, actorID, couponID string, platformAdmin bool) error {
	if !platformAdmin {
		return apperr.Forbidden("only platform admins can revoke coupons")
	}
	return s.coupons.Revoke(ctx, couponID)
}

func (s *PaymentService) applyPaymentSuccess(ctx context.Context, order *domain.Order, eventID, eventType string) (*domain.Order, error) {
	if order.Status == domain.OrderStatusPaid {
		return order, nil
	}
	if order.Status != domain.OrderStatusPending {
		return nil, apperr.Conflict("order is not pending")
	}

	now := time.Now().UTC()
	order.Status = domain.OrderStatusPaid
	order.PaidAt = &now

	if order.CouponID != nil {
		if err := s.coupons.IncrementUses(ctx, *order.CouponID); err != nil {
			return nil, err
		}
	}

	en, err := s.enrollments.EnrollFromPayment(ctx, order.UserID, order.CourseID)
	if err != nil {
		return nil, err
	}
	order.EnrollmentID = &en.ID

	if err := s.orders.UpdateStatus(ctx, order); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, order.ID)
}

func (s *PaymentService) applyRefund(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	if order.Status == domain.OrderStatusRefunded {
		return order, nil
	}
	if order.Status != domain.OrderStatusPaid {
		return nil, apperr.Validation("only paid orders can be refunded")
	}
	now := time.Now().UTC()
	order.Status = domain.OrderStatusRefunded
	order.RefundedAt = &now
	if err := s.orders.UpdateStatus(ctx, order); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, order.ID)
}

func (s *PaymentService) markFailed(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	if order.Status != domain.OrderStatusPending {
		return order, nil
	}
	now := time.Now().UTC()
	order.Status = domain.OrderStatusFailed
	order.FailedAt = &now
	if err := s.orders.UpdateStatus(ctx, order); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, order.ID)
}

func (s *PaymentService) markCancelled(ctx context.Context, order *domain.Order) (*domain.Order, error) {
	if order.Status != domain.OrderStatusPending {
		return order, nil
	}
	now := time.Now().UTC()
	order.Status = domain.OrderStatusCancelled
	order.CancelledAt = &now
	if err := s.orders.UpdateStatus(ctx, order); err != nil {
		return nil, err
	}
	return s.orders.GetByID(ctx, order.ID)
}

func validateCoupon(c *domain.Coupon, courseID, currency string) error {
	if c.RevokedAt != nil {
		return apperr.Forbidden("coupon has been revoked")
	}
	now := time.Now().UTC()
	if c.ValidFrom != nil && now.Before(*c.ValidFrom) {
		return apperr.Forbidden("coupon is not yet valid")
	}
	if c.ValidUntil != nil && now.After(*c.ValidUntil) {
		return apperr.Forbidden("coupon has expired")
	}
	if c.MaxUses != nil && c.UsesCount >= *c.MaxUses {
		return apperr.Forbidden("coupon has reached its use limit")
	}
	if c.CourseID != nil && *c.CourseID != courseID {
		return apperr.Forbidden("coupon does not apply to this course")
	}
	if c.DiscountType == domain.CouponDiscountFixed && c.Currency != "" && !strings.EqualFold(c.Currency, currency) {
		return apperr.Forbidden("coupon currency does not match course currency")
	}
	return nil
}

func computeDiscount(c *domain.Coupon, amountCents int) int {
	switch c.DiscountType {
	case domain.CouponDiscountPercent:
		d := amountCents * c.DiscountValue / 100
		if d > amountCents {
			return amountCents
		}
		return d
	case domain.CouponDiscountFixed:
		if c.DiscountValue > amountCents {
			return amountCents
		}
		return c.DiscountValue
	default:
		return 0
	}
}
