CREATE TABLE live_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    lesson_id           UUID REFERENCES lessons(id) ON DELETE SET NULL,
    title               TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'draft',
    starts_at           TIMESTAMPTZ NOT NULL,
    ends_at             TIMESTAMPTZ NOT NULL,
    timezone            TEXT NOT NULL DEFAULT 'UTC',
    provider            TEXT NOT NULL DEFAULT 'custom',
    join_url            TEXT NOT NULL DEFAULT '',
    host_url            TEXT NOT NULL DEFAULT '',
    external_id         TEXT NOT NULL DEFAULT '',
    provider_metadata   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by          UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT live_sessions_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT live_sessions_status_valid CHECK (status IN ('draft', 'scheduled', 'completed', 'cancelled')),
    CONSTRAINT live_sessions_provider_valid CHECK (provider IN ('custom', 'jitsi', 'zoom', 'google_meet')),
    CONSTRAINT live_sessions_time_order CHECK (ends_at > starts_at)
);

CREATE INDEX live_sessions_course_id_idx ON live_sessions (course_id);
CREATE INDEX live_sessions_starts_at_idx ON live_sessions (starts_at);
CREATE INDEX live_sessions_course_starts_idx ON live_sessions (course_id, starts_at);

CREATE TRIGGER live_sessions_set_updated_at
    BEFORE UPDATE ON live_sessions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE session_attendance (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES live_sessions(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'absent',
    joined_at       TIMESTAMPTZ,
    left_at         TIMESTAMPTZ,
    marked_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    note            TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT session_attendance_status_valid CHECK (status IN ('registered', 'present', 'absent', 'late', 'excused')),
    CONSTRAINT session_attendance_session_user_uidx UNIQUE (session_id, user_id)
);

CREATE INDEX session_attendance_session_id_idx ON session_attendance (session_id);
CREATE INDEX session_attendance_user_id_idx ON session_attendance (user_id);

CREATE TRIGGER session_attendance_set_updated_at
    BEFORE UPDATE ON session_attendance
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
