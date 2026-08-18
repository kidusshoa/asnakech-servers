package handlers

import (
	"io"
	"strconv"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	payments *service.PaymentService
}

func NewPaymentHandler(payments *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

type checkoutRequest struct {
	CouponCode string `json:"coupon_code"`
	Provider   string `json:"provider"`
}

type createCouponRequest struct {
	Code          string     `json:"code" binding:"required"`
	DiscountType  string     `json:"discount_type" binding:"required"`
	DiscountValue int        `json:"discount_value" binding:"required"`
	Currency      string     `json:"currency"`
	CourseID      *string    `json:"course_id"`
	MaxUses       *int       `json:"max_uses"`
	ValidFrom     *time.Time `json:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until"`
}

type OrderResponse struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	CourseID      string     `json:"course_id"`
	Status        string     `json:"status"`
	AmountCents   int        `json:"amount_cents"`
	DiscountCents int        `json:"discount_cents"`
	TotalCents    int        `json:"total_cents"`
	Currency      string     `json:"currency"`
	CouponID      *string    `json:"coupon_id,omitempty"`
	CouponCode    string     `json:"coupon_code,omitempty"`
	Provider      string     `json:"provider"`
	ProviderRef   string     `json:"provider_ref,omitempty"`
	EnrollmentID  *string    `json:"enrollment_id,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	RefundedAt    *time.Time `json:"refunded_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	UserEmail     string     `json:"user_email,omitempty"`
	UserFullName  string     `json:"user_full_name,omitempty"`
	CourseTitle   string     `json:"course_title,omitempty"`
	CourseSlug    string     `json:"course_slug,omitempty"`
}

type CheckoutResponse struct {
	Order        OrderResponse     `json:"order"`
	CheckoutURL  string            `json:"checkout_url,omitempty"`
	ProviderRef  string            `json:"provider_ref,omitempty"`
	ProviderMeta map[string]string `json:"provider_meta,omitempty"`
}

type CheckoutEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    CheckoutResponse `json:"data"`
}

type OrderEnvelope struct {
	Success bool          `json:"success" example:"true"`
	Data    OrderResponse `json:"data"`
}

type OrderListEnvelope struct {
	Success bool            `json:"success" example:"true"`
	Data    []OrderResponse `json:"data"`
	Meta    response.Meta   `json:"meta"`
}

type CouponResponse struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	DiscountType  string     `json:"discount_type"`
	DiscountValue int        `json:"discount_value"`
	Currency      string     `json:"currency,omitempty"`
	CourseID      *string    `json:"course_id,omitempty"`
	MaxUses       *int       `json:"max_uses,omitempty"`
	UsesCount     int        `json:"uses_count"`
	ValidFrom     *time.Time `json:"valid_from,omitempty"`
	ValidUntil    *time.Time `json:"valid_until,omitempty"`
	CreatedBy     string     `json:"created_by"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CouponEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    CouponResponse `json:"data"`
}

type CouponListEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    []CouponResponse `json:"data"`
}

// CreateCheckout godoc
// @Summary Start course checkout
// @Description Creates a pending order for a paid course. Free courses enroll directly.
// @Tags payments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Course ID"
// @Param Idempotency-Key header string false "Replay-safe checkout key"
// @Param body body checkoutRequest false "Checkout options"
// @Success 201 {object} CheckoutEnvelope
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/courses/{id}/checkout [post]
func (h *PaymentHandler) CreateCheckout(c *gin.Context) {
	var req checkoutRequest
	_ = c.ShouldBindJSON(&req)

	actorID := c.GetString(middleware.ContextUserID)
	idempotencyKey := c.GetHeader("Idempotency-Key")

	result, err := h.payments.CreateCheckout(
		c.Request.Context(),
		actorID,
		c.Param("id"),
		req.CouponCode,
		idempotencyKey,
		domain.PaymentProvider(req.Provider),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Created(c, CheckoutResponse{
		Order:        toOrderResponse(result.Order),
		CheckoutURL:  result.CheckoutURL,
		ProviderRef:  result.ProviderRef,
		ProviderMeta: result.ProviderMeta,
	})
}

// ConfirmOrder godoc
// @Summary Confirm manual payment (development)
// @Description Marks a pending manual-provider order as paid and enrolls the buyer.
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param orderId path string true "Order ID"
// @Success 200 {object} OrderEnvelope
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/orders/{orderId}/confirm [post]
func (h *PaymentHandler) ConfirmOrder(c *gin.Context) {
	order, err := h.payments.ConfirmManual(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("orderId"))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}

// GetOrder godoc
// @Summary Get order
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param orderId path string true "Order ID"
// @Success 200 {object} OrderEnvelope
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/orders/{orderId} [get]
func (h *PaymentHandler) GetOrder(c *gin.Context) {
	order, err := h.payments.GetOrder(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("orderId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}

// ListMyOrders godoc
// @Summary List my orders
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page"
// @Param per_page query int false "Per page"
// @Param status query string false "Order status filter"
// @Success 200 {object} OrderListEnvelope
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/me/orders [get]
func (h *PaymentHandler) ListMyOrders(c *gin.Context) {
	filter := domain.OrderListFilter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryInt(c, "per_page", 20),
		Status:  domain.OrderStatus(c.Query("status")),
	}
	orders, total, err := h.payments.ListMine(c.Request.Context(), c.GetString(middleware.ContextUserID), filter)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.JSON(c, 200, toOrderResponses(orders), response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      filter.Page,
		PerPage:   filter.PerPage,
		Total:     total,
	})
}

// ListCourseOrders godoc
// @Summary List course orders (teacher)
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param id path string true "Course ID"
// @Param page query int false "Page"
// @Param per_page query int false "Per page"
// @Param status query string false "Order status filter"
// @Success 200 {object} OrderListEnvelope
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/courses/{id}/orders [get]
func (h *PaymentHandler) ListCourseOrders(c *gin.Context) {
	filter := domain.OrderListFilter{
		Page:    queryInt(c, "page", 1),
		PerPage: queryInt(c, "per_page", 20),
		Status:  domain.OrderStatus(c.Query("status")),
	}
	orders, total, err := h.payments.ListForCourse(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		filter,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.JSON(c, 200, toOrderResponses(orders), response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      filter.Page,
		PerPage:   filter.PerPage,
		Total:     total,
	})
}

// RefundOrder godoc
// @Summary Refund a paid order
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param orderId path string true "Order ID"
// @Success 200 {object} OrderEnvelope
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/orders/{orderId}/refund [post]
func (h *PaymentHandler) RefundOrder(c *gin.Context) {
	order, err := h.payments.Refund(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("orderId"),
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}

// PaymentWebhook godoc
// @Summary Payment provider webhook
// @Description Idempotent callback from manual, Stripe, or Chapa. See docs/api/payments.md.
// @Tags payments
// @Accept json
// @Produce json
// @Param provider path string true "Provider (manual, stripe, chapa)"
// @Success 200 {object} OrderEnvelope
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/webhooks/payments/{provider} [post]
func (h *PaymentHandler) PaymentWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Fail(c, err)
		return
	}
	order, err := h.payments.HandleWebhook(
		c.Request.Context(),
		domain.PaymentProvider(c.Param("provider")),
		c.Request.Header,
		body,
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}

// CreateCoupon godoc
// @Summary Create coupon (admin)
// @Tags payments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body createCouponRequest true "Coupon"
// @Success 201 {object} CouponEnvelope
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/coupons [post]
func (h *PaymentHandler) CreateCoupon(c *gin.Context) {
	var req createCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, err)
		return
	}
	coupon, err := h.payments.CreateCoupon(c.Request.Context(), c.GetString(middleware.ContextUserID), domain.CreateCouponInput{
		Code:          req.Code,
		DiscountType:  domain.CouponDiscountType(req.DiscountType),
		DiscountValue: req.DiscountValue,
		Currency:      req.Currency,
		CourseID:      req.CourseID,
		MaxUses:       req.MaxUses,
		ValidFrom:     req.ValidFrom,
		ValidUntil:    req.ValidUntil,
	}, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toCouponResponse(coupon))
}

// ListCoupons godoc
// @Summary List coupons
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param course_id query string false "Filter by course"
// @Success 200 {object} CouponListEnvelope
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/coupons [get]
func (h *PaymentHandler) ListCoupons(c *gin.Context) {
	var courseID *string
	if raw := c.Query("course_id"); raw != "" {
		courseID = &raw
	}
	coupons, err := h.payments.ListCoupons(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		courseID,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toCouponResponses(coupons))
}

// RevokeCoupon godoc
// @Summary Revoke coupon (admin)
// @Tags payments
// @Produce json
// @Security BearerAuth
// @Param couponId path string true "Coupon ID"
// @Success 204 "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/coupons/{couponId}/revoke [post]
func (h *PaymentHandler) RevokeCoupon(c *gin.Context) {
	if err := h.payments.RevokeCoupon(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("couponId"),
		isPlatformAdmin(c),
	); err != nil {
		response.Fail(c, err)
		return
	}
	c.Status(204)
}

func toOrderResponse(o *domain.Order) OrderResponse {
	if o == nil {
		return OrderResponse{}
	}
	return OrderResponse{
		ID:            o.ID,
		UserID:        o.UserID,
		CourseID:      o.CourseID,
		Status:        string(o.Status),
		AmountCents:   o.AmountCents,
		DiscountCents: o.DiscountCents,
		TotalCents:    o.TotalCents,
		Currency:      o.Currency,
		CouponID:      o.CouponID,
		CouponCode:    o.CouponCode,
		Provider:      string(o.Provider),
		ProviderRef:   o.ProviderRef,
		EnrollmentID:  o.EnrollmentID,
		PaidAt:        o.PaidAt,
		FailedAt:      o.FailedAt,
		RefundedAt:    o.RefundedAt,
		CancelledAt:   o.CancelledAt,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
		UserEmail:     o.UserEmail,
		UserFullName:  o.UserFullName,
		CourseTitle:   o.CourseTitle,
		CourseSlug:    o.CourseSlug,
	}
}

func toOrderResponses(items []domain.Order) []OrderResponse {
	out := make([]OrderResponse, len(items))
	for i := range items {
		out[i] = toOrderResponse(&items[i])
	}
	return out
}

func toCouponResponse(c *domain.Coupon) CouponResponse {
	if c == nil {
		return CouponResponse{}
	}
	return CouponResponse{
		ID:            c.ID,
		Code:          c.Code,
		DiscountType:  string(c.DiscountType),
		DiscountValue: c.DiscountValue,
		Currency:      c.Currency,
		CourseID:      c.CourseID,
		MaxUses:       c.MaxUses,
		UsesCount:     c.UsesCount,
		ValidFrom:     c.ValidFrom,
		ValidUntil:    c.ValidUntil,
		CreatedBy:     c.CreatedBy,
		RevokedAt:     c.RevokedAt,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func toCouponResponses(items []domain.Coupon) []CouponResponse {
	out := make([]CouponResponse, len(items))
	for i := range items {
		out[i] = toCouponResponse(&items[i])
	}
	return out
}

func queryInt(c *gin.Context, key string, def int) int {
	raw := c.Query(key)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}
