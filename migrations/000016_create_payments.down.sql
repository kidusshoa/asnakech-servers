DROP TABLE IF EXISTS payment_webhook_events;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS coupons;

ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS enrollments_source_valid;
ALTER TABLE enrollments ADD CONSTRAINT enrollments_source_valid
    CHECK (source IN ('self', 'invite_code', 'teacher'));
