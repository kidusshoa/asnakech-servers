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

// ChapaProvider is a v1 stub for Ethiopian Chapa payments.
type ChapaProvider struct {
	SecretKey     string
	WebhookSecret string
}

func (ChapaProvider) Type() domain.PaymentProvider { return domain.PaymentProviderChapa }

func (p ChapaProvider) CreateCheckout(_ context.Context, order *domain.Order) (*CheckoutSession, error) {
	if strings.TrimSpace(p.SecretKey) == "" {
		return nil, apperr.Validation("CHAPA_SECRET_KEY is required for chapa provider — use manual in development")
	}
	return &CheckoutSession{
		URL:        fmt.Sprintf("https://checkout.chapa.co/checkout/payment/%s", order.ID),
		ExternalID: "chapa_" + order.ID,
		Metadata: map[string]string{
			"mode":     "chapa_stub",
			"order_id": order.ID,
		},
	}, nil
}

type chapaWebhookPayload struct {
	Event     string `json:"event"`
	TxRef     string `json:"tx_ref"`
	Reference string `json:"reference"`
	Status    string `json:"status"`
}

func (p ChapaProvider) ParseWebhook(_ context.Context, headers http.Header, body []byte) (*WebhookResult, error) {
	if secret := strings.TrimSpace(p.WebhookSecret); secret != "" {
		if strings.TrimSpace(headers.Get("X-Chapa-Signature")) != secret {
			return nil, apperr.Forbidden("invalid Chapa webhook signature")
		}
	}

	var payload chapaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, apperr.Validation("invalid chapa webhook JSON")
	}
	orderID := strings.TrimSpace(payload.TxRef)
	if orderID == "" {
		return nil, apperr.Validation("chapa tx_ref (order id) is required")
	}
	eventID := strings.TrimSpace(payload.Reference)
	if eventID == "" {
		eventID = "chapa-" + orderID + "-" + strings.TrimSpace(payload.Status)
	}

	status, err := parseOrderStatus(strings.ToLower(strings.TrimSpace(payload.Status)))
	if err != nil {
		return nil, err
	}
	eventType := strings.TrimSpace(payload.Event)
	if eventType == "" {
		eventType = "chapa.payment." + string(status)
	}

	return &WebhookResult{
		EventID:     eventID,
		EventType:   eventType,
		OrderID:     orderID,
		Status:      status,
		ProviderRef: strings.TrimSpace(payload.Reference),
	}, nil
}
