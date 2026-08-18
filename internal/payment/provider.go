package payment

import (
	"context"
	"net/http"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/domain"
)

// CheckoutSession is returned when a pending order is created.
type CheckoutSession struct {
	URL        string
	ExternalID string
	Metadata   map[string]string
}

// WebhookResult is a normalized provider callback.
type WebhookResult struct {
	EventID     string
	EventType   string
	OrderID     string
	Status      domain.OrderStatus
	ProviderRef string
}

// Provider creates checkout sessions and verifies webhooks.
type Provider interface {
	Type() domain.PaymentProvider
	CreateCheckout(ctx context.Context, order *domain.Order) (*CheckoutSession, error)
	ParseWebhook(ctx context.Context, headers http.Header, body []byte) (*WebhookResult, error)
}

// Registry selects the adapter for a provider name.
type Registry struct {
	providers map[domain.PaymentProvider]Provider
	defaultP  domain.PaymentProvider
}

func NewRegistry(cfg *config.Config) *Registry {
	defaultP := domain.PaymentProvider(cfg.PaymentDefaultProvider)
	if defaultP == "" {
		defaultP = domain.PaymentProviderManual
	}
	return &Registry{
		defaultP: defaultP,
		providers: map[domain.PaymentProvider]Provider{
			domain.PaymentProviderManual: ManualProvider{WebhookSecret: cfg.PaymentWebhookSecret},
			domain.PaymentProviderStripe: StripeProvider{
				SecretKey:     cfg.StripeSecretKey,
				WebhookSecret: cfg.StripeWebhookSecret,
			},
			domain.PaymentProviderChapa: ChapaProvider{
				SecretKey:     cfg.ChapaSecretKey,
				WebhookSecret: cfg.ChapaWebhookSecret,
			},
		},
	}
}

func (r *Registry) DefaultProvider() domain.PaymentProvider {
	return r.defaultP
}

func (r *Registry) For(p domain.PaymentProvider) Provider {
	if p == "" {
		p = r.defaultP
	}
	if prov, ok := r.providers[p]; ok {
		return prov
	}
	return r.providers[domain.PaymentProviderManual]
}
