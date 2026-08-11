CREATE TABLE course_modules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id  UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT course_modules_title_nonempty CHECK (char_length(trim(title)) > 0)
);

CREATE INDEX course_modules_course_id_idx ON course_modules (course_id);
CREATE UNIQUE INDEX course_modules_course_position_uidx ON course_modules (course_id, position);

CREATE TRIGGER course_modules_set_updated_at
    BEFORE UPDATE ON course_modules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE lessons (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_id              UUID NOT NULL REFERENCES course_modules(id) ON DELETE CASCADE,
    title                  TEXT NOT NULL,
    slug                   TEXT NOT NULL,
    summary                TEXT NOT NULL DEFAULT '',
    status                 TEXT NOT NULL DEFAULT 'draft',
    position               INTEGER NOT NULL DEFAULT 0,
    prerequisite_lesson_id UUID REFERENCES lessons(id) ON DELETE SET NULL,
    estimated_minutes      INTEGER NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lessons_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT lessons_slug_nonempty CHECK (char_length(trim(slug)) > 0),
    CONSTRAINT lessons_status_valid CHECK (status IN ('draft', 'published')),
    CONSTRAINT lessons_minutes_nonneg CHECK (estimated_minutes >= 0)
);

CREATE INDEX lessons_module_id_idx ON lessons (module_id);
CREATE UNIQUE INDEX lessons_module_position_uidx ON lessons (module_id, position);
CREATE UNIQUE INDEX lessons_module_slug_uidx ON lessons (module_id, lower(slug));

CREATE TRIGGER lessons_set_updated_at
    BEFORE UPDATE ON lessons
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE content_blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id   UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    block_type  TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    media_url   TEXT NOT NULL DEFAULT '',
    quiz_ref_id UUID,
    position    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT content_blocks_type_valid CHECK (block_type IN ('text', 'video', 'file', 'quiz_ref'))
);

CREATE INDEX content_blocks_lesson_id_idx ON content_blocks (lesson_id);
CREATE UNIQUE INDEX content_blocks_lesson_position_uidx ON content_blocks (lesson_id, position);

CREATE TRIGGER content_blocks_set_updated_at
    BEFORE UPDATE ON content_blocks
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
