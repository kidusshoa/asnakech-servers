-- Course enrollment controls (NULL capacity = unlimited).
ALTER TABLE courses
    ADD COLUMN enrollment_capacity INTEGER,
    ADD COLUMN enrollment_open BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN waitlist_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE courses
    ADD CONSTRAINT courses_enrollment_capacity_pos
    CHECK (enrollment_capacity IS NULL OR enrollment_capacity > 0);

CREATE TABLE enrollment_invite_codes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id    UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    code         TEXT NOT NULL,
    max_uses     INTEGER,
    uses_count   INTEGER NOT NULL DEFAULT 0,
    expires_at   TIMESTAMPTZ,
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollment_invite_codes_code_nonempty CHECK (char_length(trim(code)) > 0),
    CONSTRAINT enrollment_invite_codes_max_uses_pos CHECK (max_uses IS NULL OR max_uses > 0),
    CONSTRAINT enrollment_invite_codes_uses_nonneg CHECK (uses_count >= 0)
);

CREATE UNIQUE INDEX enrollment_invite_codes_course_code_uidx
    ON enrollment_invite_codes (course_id, lower(code));
CREATE INDEX enrollment_invite_codes_course_id_idx ON enrollment_invite_codes (course_id);

CREATE TRIGGER enrollment_invite_codes_set_updated_at
    BEFORE UPDATE ON enrollment_invite_codes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE enrollments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id        UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'active',
    source           TEXT NOT NULL DEFAULT 'self',
    invite_code_id   UUID REFERENCES enrollment_invite_codes(id) ON DELETE SET NULL,
    enrolled_at      TIMESTAMPTZ,
    waitlisted_at    TIMESTAMPTZ,
    cancelled_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollments_status_valid CHECK (status IN ('active', 'waitlisted', 'cancelled')),
    CONSTRAINT enrollments_source_valid CHECK (source IN ('self', 'invite_code', 'teacher'))
);

CREATE UNIQUE INDEX enrollments_course_user_uidx ON enrollments (course_id, user_id);
CREATE INDEX enrollments_user_id_idx ON enrollments (user_id);
CREATE INDEX enrollments_course_status_idx ON enrollments (course_id, status);
CREATE INDEX enrollments_waitlist_idx ON enrollments (course_id, waitlisted_at)
    WHERE status = 'waitlisted';

CREATE TRIGGER enrollments_set_updated_at
    BEFORE UPDATE ON enrollments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Append-only lifecycle log for later notification workers.
CREATE TABLE enrollment_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    course_id     UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type    TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollment_events_type_valid CHECK (
        event_type IN ('enrolled', 'waitlisted', 'activated', 'cancelled')
    )
);

CREATE INDEX enrollment_events_enrollment_id_idx ON enrollment_events (enrollment_id);
CREATE INDEX enrollment_events_course_id_idx ON enrollment_events (course_id);
CREATE INDEX enrollment_events_created_at_idx ON enrollment_events (created_at);
