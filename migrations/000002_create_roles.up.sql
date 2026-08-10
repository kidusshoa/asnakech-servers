CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_code_nonempty CHECK (char_length(trim(code)) > 0)
);

CREATE TRIGGER roles_set_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

INSERT INTO roles (code, name, description) VALUES
    ('student', 'Student', 'Learner enrolled in courses'),
    ('teacher', 'Teacher', 'Creates and delivers course content'),
    ('admin', 'Admin', 'Platform or school administrator'),
    ('parent', 'Parent', 'Guardian linked to one or more students');
