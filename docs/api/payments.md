# Payments & monetization

Course checkout, orders, coupons, refunds, and provider webhooks. Paid courses (`price_cents > 0`) require checkout before enrollment.

## Flow

1. Student calls `POST /api/v1/courses/:id/checkout` (optional `coupon_code`, `Idempotency-Key` header).
2. Provider adapter returns checkout metadata (`manual`, `stripe`, or `chapa`).
3. On successful payment, webhook or manual confirm marks the order **paid** and creates an enrollment with `source=payment`.
4. Free courses (`price_cents = 0`) still enroll via `POST /courses/:id/enroll`.

## Providers

| Provider | Env | Checkout | Webhook |
|----------|-----|----------|---------|
| `manual` (default) | `PAYMENT_DEFAULT_PROVIDER=manual` | No external URL — use confirm endpoint | `POST /webhooks/payments/manual` |
| `stripe` | `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` | Stub checkout URL (v1) | `POST /webhooks/payments/stripe` |
| `chapa` | `CHAPA_SECRET_KEY`, `CHAPA_WEBHOOK_SECRET` | Stub checkout URL (v1) | `POST /webhooks/payments/chapa` |

### Manual webhook (development)

```json
POST /api/v1/webhooks/payments/manual
X-Payment-Signature: <hmac-sha256-hex of body using PAYMENT_WEBHOOK_SECRET>

{
  "event_id": "evt-001",
  "order_id": "<uuid>",
  "status": "paid"
}
```

When `PAYMENT_WEBHOOK_SECRET` is empty, signature is skipped (development only).

### Stripe webhook (v1 stub)

Expects `checkout.session.completed` with `data.object.client_reference_id` set to the order UUID. Requires `Stripe-Signature` header when `STRIPE_WEBHOOK_SECRET` is set.

### Chapa webhook (v1 stub)

Expects `tx_ref` = order UUID and `status` = `success`. Optional `X-Chapa-Signature` header.

## Idempotency

- **Checkout:** `Idempotency-Key` header — replays return the same pending order.
- **Webhooks:** `(provider, event_id)` stored in `payment_webhook_events`; duplicates are ignored.

## Coupons

Admin-only create/revoke. Teachers may list coupons scoped to their course.

| Type | `discount_value` |
|------|------------------|
| `percent` | 1–100 |
| `fixed` | Amount in cents (optional `currency` must match course) |

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST` | `/api/v1/courses/:id/checkout` | student |
| `GET` | `/api/v1/courses/:id/orders` | teacher |
| `GET` | `/api/v1/me/orders` | owner |
| `GET` | `/api/v1/orders/:id` | owner, teacher, admin |
| `POST` | `/api/v1/orders/:id/confirm` | owner (manual provider) |
| `POST` | `/api/v1/orders/:id/refund` | teacher / admin |
| `POST` | `/api/v1/webhooks/payments/:provider` | public (signed) |
| `POST` | `/api/v1/coupons` | admin |
| `GET` | `/api/v1/coupons` | admin / teacher (course filter) |
| `POST` | `/api/v1/coupons/:id/revoke` | admin |

## Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `PAYMENT_DEFAULT_PROVIDER` | `manual` | Default checkout provider |
| `PAYMENT_WEBHOOK_SECRET` | _(empty)_ | HMAC secret for manual webhooks |
| `STRIPE_SECRET_KEY` | _(empty)_ | Stripe API key |
| `STRIPE_WEBHOOK_SECRET` | _(empty)_ | Stripe webhook signing |
| `CHAPA_SECRET_KEY` | _(empty)_ | Chapa secret |
| `CHAPA_WEBHOOK_SECRET` | _(empty)_ | Chapa webhook header value |
