-- Allow enrollments unlocked by successful payment.
ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS enrollments_source_valid;
ALTER TABLE enrollments ADD CONSTRAINT enrollments_source_valid
    CHECK (source IN ('self', 'invite_code', 'teacher', 'payment'));

CREATE TABLE coupons (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT NOT NULL,
    discount_type   TEXT NOT NULL,
    discount_value  INTEGER NOT NULL,
    currency        TEXT,
    course_id       UUID REFERENCES courses(id) ON DELETE CASCADE,
    max_uses        INTEGER,
    uses_count      INTEGER NOT NULL DEFAULT 0,
    valid_from      TIMESTAMPTZ,
    valid_until     TIMESTAMPTZ,
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT coupons_code_nonempty CHECK (char_length(trim(code)) > 0),
    CONSTRAINT coupons_discount_type_valid CHECK (discount_type IN ('percent', 'fixed')),
    CONSTRAINT coupons_discount_value_pos CHECK (discount_value > 0),
    CONSTRAINT coupons_percent_max CHECK (
        discount_type != 'percent' OR discount_value <= 100
    ),
    CONSTRAINT coupons_max_uses_pos CHECK (max_uses IS NULL OR max_uses > 0),
    CONSTRAINT coupons_uses_nonneg CHECK (uses_count >= 0)
);

CREATE UNIQUE INDEX coupons_code_uidx ON coupons (lower(code));
CREATE INDEX coupons_course_id_idx ON coupons (course_id);

CREATE TRIGGER coupons_set_updated_at
    BEFORE UPDATE ON coupons
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id        UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'pending',
    amount_cents     INTEGER NOT NULL,
    discount_cents   INTEGER NOT NULL DEFAULT 0,
    total_cents      INTEGER NOT NULL,
    currency         TEXT NOT NULL DEFAULT 'ETB',
    coupon_id        UUID REFERENCES coupons(id) ON DELETE SET NULL,
    provider         TEXT NOT NULL,
    provider_ref     TEXT,
    idempotency_key  TEXT,
    enrollment_id    UUID REFERENCES enrollments(id) ON DELETE SET NULL,
    paid_at          TIMESTAMPTZ,
    failed_at        TIMESTAMPTZ,
    refunded_at      TIMESTAMPTZ,
    cancelled_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT orders_status_valid CHECK (
        status IN ('pending', 'paid', 'failed', 'refunded', 'cancelled')
    ),
    CONSTRAINT orders_provider_valid CHECK (provider IN ('manual', 'stripe', 'chapa')),
    CONSTRAINT orders_amount_nonneg CHECK (amount_cents >= 0),
    CONSTRAINT orders_discount_nonneg CHECK (discount_cents >= 0),
    CONSTRAINT orders_total_nonneg CHECK (total_cents >= 0),
    CONSTRAINT orders_total_lte_amount CHECK (total_cents <= amount_cents)
);

CREATE UNIQUE INDEX orders_idempotency_key_uidx
    ON orders (idempotency_key)
    WHERE idempotency_key IS NOT NULL;
CREATE INDEX orders_user_id_idx ON orders (user_id);
CREATE INDEX orders_course_id_idx ON orders (course_id);
CREATE INDEX orders_status_idx ON orders (status);
CREATE INDEX orders_provider_ref_idx ON orders (provider, provider_ref)
    WHERE provider_ref IS NOT NULL;

CREATE TRIGGER orders_set_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Idempotent webhook ingestion (provider event_id dedup).
CREATE TABLE payment_webhook_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider     TEXT NOT NULL,
    event_id     TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    order_id     UUID REFERENCES orders(id) ON DELETE SET NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payment_webhook_events_provider_valid
        CHECK (provider IN ('manual', 'stripe', 'chapa'))
);

CREATE UNIQUE INDEX payment_webhook_events_provider_event_uidx
    ON payment_webhook_events (provider, event_id);
CREATE INDEX payment_webhook_events_order_id_idx ON payment_webhook_events (order_id);
