-- Full-text search vector for course discovery.
ALTER TABLE courses ADD COLUMN IF NOT EXISTS search_vector tsvector;

UPDATE courses SET search_vector =
    setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(summary, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(description, '')), 'C')
WHERE search_vector IS NULL;

CREATE INDEX IF NOT EXISTS courses_search_vector_idx ON courses USING GIN (search_vector);

CREATE OR REPLACE FUNCTION courses_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', coalesce(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(NEW.summary, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(NEW.description, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS courses_search_vector_trigger ON courses;
CREATE TRIGGER courses_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, summary, description ON courses
    FOR EACH ROW
    EXECUTE FUNCTION courses_search_vector_update();

-- Parent/guardian ↔ student links.
CREATE TABLE parent_student_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT parent_student_links_status_valid CHECK (status IN ('active', 'revoked')),
    CONSTRAINT parent_student_links_not_self CHECK (parent_user_id != student_user_id)
);

CREATE UNIQUE INDEX parent_student_links_pair_uidx
    ON parent_student_links (parent_user_id, student_user_id);
CREATE INDEX parent_student_links_parent_idx ON parent_student_links (parent_user_id);
CREATE INDEX parent_student_links_student_idx ON parent_student_links (student_user_id);

CREATE TRIGGER parent_student_links_set_updated_at
    BEFORE UPDATE ON parent_student_links
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
