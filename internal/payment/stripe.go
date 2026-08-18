package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
)

// StripeProvider is a v1 stub — parses checkout.session.completed webhooks.
// Full Stripe Checkout Session creation requires stripe-go in a later iteration.
type StripeProvider struct {
	SecretKey     string
	WebhookSecret string
}

func (StripeProvider) Type() domain.PaymentProvider { return domain.PaymentProviderStripe }

func (p StripeProvider) CreateCheckout(_ context.Context, order *domain.Order) (*CheckoutSession, error) {
	if strings.TrimSpace(p.SecretKey) == "" {
		return nil, apperr.Validation("STRIPE_SECRET_KEY is required for stripe provider — use manual in development")
	}
	// Stub URL — real integration would call Stripe Checkout Sessions API.
	return &CheckoutSession{
		URL:        fmt.Sprintf("https://checkout.stripe.com/c/pay/%s", order.ID),
		ExternalID: "cs_stub_" + order.ID,
		Metadata: map[string]string{
			"mode":     "stripe_stub",
			"order_id": order.ID,
		},
	}, nil
}

type stripeWebhookEnvelope struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID                string `json:"id"`
			ClientReferenceID string `json:"client_reference_id"`
			PaymentStatus     string `json:"payment_status"`
		} `json:"object"`
	} `json:"data"`
}

func (p StripeProvider) ParseWebhook(_ context.Context, headers http.Header, body []byte) (*WebhookResult, error) {
	if secret := strings.TrimSpace(p.WebhookSecret); secret != "" {
		sig := strings.TrimSpace(headers.Get("Stripe-Signature"))
		if sig == "" {
			return nil, apperr.Forbidden("missing Stripe-Signature")
		}
		// v1: presence check only; production should verify timestamp + HMAC.
		if !strings.Contains(sig, "v1=") {
			return nil, apperr.Forbidden("invalid Stripe-Signature")
		}
	}

	var env stripeWebhookEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, apperr.Validation("invalid stripe webhook JSON")
	}
	if env.ID == "" {
		return nil, apperr.Validation("stripe event id is required")
	}

	orderID := strings.TrimSpace(env.Data.Object.ClientReferenceID)
	if orderID == "" {
		return nil, apperr.Validation("stripe client_reference_id (order id) is required")
	}

	status := domain.OrderStatusFailed
	switch env.Type {
	case "checkout.session.completed", "payment_intent.succeeded":
		if env.Data.Object.PaymentStatus == "" || env.Data.Object.PaymentStatus == "paid" {
			status = domain.OrderStatusPaid
		}
	case "charge.refunded":
		status = domain.OrderStatusRefunded
	default:
		return nil, apperr.Validation("unsupported stripe event type")
	}

	return &WebhookResult{
		EventID:     env.ID,
		EventType:   env.Type,
		OrderID:     orderID,
		Status:      status,
		ProviderRef: strings.TrimSpace(env.Data.Object.ID),
	}, nil
}
