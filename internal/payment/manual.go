package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
)

// ManualProvider supports local/dev checkout via confirm endpoint or signed webhook.
type ManualProvider struct {
	WebhookSecret string
}

func (ManualProvider) Type() domain.PaymentProvider { return domain.PaymentProviderManual }

func (ManualProvider) CreateCheckout(_ context.Context, order *domain.Order) (*CheckoutSession, error) {
	return &CheckoutSession{
		ExternalID: order.ID,
		Metadata: map[string]string{
			"mode":        "manual",
			"order_id":    order.ID,
			"instruction": "POST /api/v1/orders/{id}/confirm after payment, or send webhook to /api/v1/webhooks/payments/manual",
		},
	}, nil
}

type manualWebhookPayload struct {
	EventID     string `json:"event_id"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	ProviderRef string `json:"provider_ref"`
}

func (p ManualProvider) ParseWebhook(_ context.Context, headers http.Header, body []byte) (*WebhookResult, error) {
	if secret := strings.TrimSpace(p.WebhookSecret); secret != "" {
		sig := strings.TrimSpace(headers.Get("X-Payment-Signature"))
		if sig == "" {
			return nil, apperr.Forbidden("missing X-Payment-Signature")
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(expected)) {
			return nil, apperr.Forbidden("invalid webhook signature")
		}
	}

	var payload manualWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, apperr.Validation("invalid manual webhook JSON")
	}
	payload.EventID = strings.TrimSpace(payload.EventID)
	payload.OrderID = strings.TrimSpace(payload.OrderID)
	payload.Status = strings.TrimSpace(strings.ToLower(payload.Status))
	if payload.EventID == "" || payload.OrderID == "" {
		return nil, apperr.Validation("event_id and order_id are required")
	}
	status, err := parseOrderStatus(payload.Status)
	if err != nil {
		return nil, err
	}
	return &WebhookResult{
		EventID:     payload.EventID,
		EventType:   "manual.payment." + payload.Status,
		OrderID:     payload.OrderID,
		Status:      status,
		ProviderRef: strings.TrimSpace(payload.ProviderRef),
	}, nil
}

func parseOrderStatus(raw string) (domain.OrderStatus, error) {
	switch raw {
	case "paid", "success", "completed":
		return domain.OrderStatusPaid, nil
	case "failed":
		return domain.OrderStatusFailed, nil
	case "refunded":
		return domain.OrderStatusRefunded, nil
	case "cancelled", "canceled":
		return domain.OrderStatusCancelled, nil
	default:
		return "", apperr.Validation(fmt.Sprintf("unsupported payment status %q", raw))
	}
}
