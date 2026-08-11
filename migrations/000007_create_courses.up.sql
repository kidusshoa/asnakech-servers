CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT categories_name_nonempty CHECK (char_length(trim(name)) > 0),
    CONSTRAINT categories_slug_nonempty CHECK (char_length(trim(slug)) > 0)
);

CREATE UNIQUE INDEX categories_slug_uidx ON categories (lower(slug));

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tags_name_nonempty CHECK (char_length(trim(name)) > 0),
    CONSTRAINT tags_slug_nonempty CHECK (char_length(trim(slug)) > 0)
);

CREATE UNIQUE INDEX tags_slug_uidx ON tags (lower(slug));

INSERT INTO categories (name, slug, description) VALUES
    ('Mathematics', 'mathematics', 'Numbers, algebra, geometry, and more'),
    ('Science', 'science', 'Physics, chemistry, biology'),
    ('Languages', 'languages', 'Reading, writing, and spoken languages'),
    ('Technology', 'technology', 'Computing and digital skills');

CREATE TABLE courses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    teacher_id      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    category_id     UUID REFERENCES categories(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    slug            TEXT NOT NULL,
    summary         TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'draft',
    cover_url       TEXT NOT NULL DEFAULT '',
    price_cents     INTEGER NOT NULL DEFAULT 0,
    currency        TEXT NOT NULL DEFAULT 'ETB',
    level           TEXT NOT NULL DEFAULT 'beginner',
    language        TEXT NOT NULL DEFAULT 'en',
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT courses_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT courses_slug_nonempty CHECK (char_length(trim(slug)) > 0),
    CONSTRAINT courses_status_valid CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT courses_level_valid CHECK (level IN ('beginner', 'intermediate', 'advanced')),
    CONSTRAINT courses_price_nonneg CHECK (price_cents >= 0),
    CONSTRAINT courses_currency_len CHECK (char_length(currency) = 3)
);

CREATE UNIQUE INDEX courses_slug_active_uidx
    ON courses (lower(slug))
    WHERE deleted_at IS NULL;

CREATE INDEX courses_teacher_id_idx ON courses (teacher_id);
CREATE INDEX courses_organization_id_idx ON courses (organization_id);
CREATE INDEX courses_category_id_idx ON courses (category_id);
CREATE INDEX courses_status_idx ON courses (status);

CREATE TRIGGER courses_set_updated_at
    BEFORE UPDATE ON courses
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE course_tags (
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    tag_id    UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, tag_id)
);

CREATE INDEX course_tags_tag_id_idx ON course_tags (tag_id);
