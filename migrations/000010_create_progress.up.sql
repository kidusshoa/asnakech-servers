CREATE TABLE lesson_progress (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id      UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    lesson_id      UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'in_progress',
    percent        INTEGER NOT NULL DEFAULT 0,
    last_position  TEXT NOT NULL DEFAULT '',
    completed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lesson_progress_status_valid CHECK (status IN ('in_progress', 'completed')),
    CONSTRAINT lesson_progress_percent_range CHECK (percent >= 0 AND percent <= 100)
);

CREATE UNIQUE INDEX lesson_progress_user_lesson_uidx ON lesson_progress (user_id, lesson_id);
CREATE INDEX lesson_progress_user_course_idx ON lesson_progress (user_id, course_id);
CREATE INDEX lesson_progress_course_id_idx ON lesson_progress (course_id);

CREATE TRIGGER lesson_progress_set_updated_at
    BEFORE UPDATE ON lesson_progress
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE course_progress (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id         UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    enrollment_id     UUID REFERENCES enrollments(id) ON DELETE SET NULL,
    percent           INTEGER NOT NULL DEFAULT 0,
    completed_lessons INTEGER NOT NULL DEFAULT 0,
    total_lessons     INTEGER NOT NULL DEFAULT 0,
    last_lesson_id    UUID REFERENCES lessons(id) ON DELETE SET NULL,
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT course_progress_percent_range CHECK (percent >= 0 AND percent <= 100),
    CONSTRAINT course_progress_counts_nonneg CHECK (completed_lessons >= 0 AND total_lessons >= 0)
);

CREATE UNIQUE INDEX course_progress_user_course_uidx ON course_progress (user_id, course_id);
CREATE INDEX course_progress_user_id_idx ON course_progress (user_id);
CREATE INDEX course_progress_course_id_idx ON course_progress (course_id);

CREATE TRIGGER course_progress_set_updated_at
    BEFORE UPDATE ON course_progress
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
