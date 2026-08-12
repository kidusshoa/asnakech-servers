CREATE TABLE quizzes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title               TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    status              TEXT NOT NULL DEFAULT 'draft',
    time_limit_seconds  INTEGER,
    max_attempts        INTEGER,
    pass_percent        INTEGER NOT NULL DEFAULT 60,
    shuffle_questions   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quizzes_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT quizzes_status_valid CHECK (status IN ('draft', 'published')),
    CONSTRAINT quizzes_pass_percent_range CHECK (pass_percent >= 0 AND pass_percent <= 100),
    CONSTRAINT quizzes_time_limit_pos CHECK (time_limit_seconds IS NULL OR time_limit_seconds > 0),
    CONSTRAINT quizzes_max_attempts_pos CHECK (max_attempts IS NULL OR max_attempts > 0)
);

CREATE INDEX quizzes_course_id_idx ON quizzes (course_id);

CREATE TRIGGER quizzes_set_updated_at
    BEFORE UPDATE ON quizzes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE quiz_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id         UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    question_type   TEXT NOT NULL,
    prompt          TEXT NOT NULL,
    points          INTEGER NOT NULL DEFAULT 1,
    position        INTEGER NOT NULL DEFAULT 0,
    options         JSONB NOT NULL DEFAULT '[]'::jsonb,
    correct_answer  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quiz_questions_type_valid CHECK (question_type IN ('mcq', 'short_answer')),
    CONSTRAINT quiz_questions_prompt_nonempty CHECK (char_length(trim(prompt)) > 0),
    CONSTRAINT quiz_questions_points_pos CHECK (points > 0)
);

CREATE INDEX quiz_questions_quiz_id_idx ON quiz_questions (quiz_id);
CREATE UNIQUE INDEX quiz_questions_quiz_position_uidx ON quiz_questions (quiz_id, position);

CREATE TRIGGER quiz_questions_set_updated_at
    BEFORE UPDATE ON quiz_questions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE quiz_attempts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id         UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attempt_number  INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'in_progress',
    score_points    INTEGER NOT NULL DEFAULT 0,
    max_points      INTEGER NOT NULL DEFAULT 0,
    percent         INTEGER NOT NULL DEFAULT 0,
    passed          BOOLEAN NOT NULL DEFAULT FALSE,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at    TIMESTAMPTZ,
    graded_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quiz_attempts_status_valid CHECK (status IN ('in_progress', 'submitted', 'graded')),
    CONSTRAINT quiz_attempts_number_pos CHECK (attempt_number > 0),
    CONSTRAINT quiz_attempts_percent_range CHECK (percent >= 0 AND percent <= 100)
);

CREATE UNIQUE INDEX quiz_attempts_quiz_user_number_uidx ON quiz_attempts (quiz_id, user_id, attempt_number);
CREATE INDEX quiz_attempts_user_id_idx ON quiz_attempts (user_id);
CREATE INDEX quiz_attempts_quiz_id_idx ON quiz_attempts (quiz_id);

CREATE TRIGGER quiz_attempts_set_updated_at
    BEFORE UPDATE ON quiz_attempts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE quiz_attempt_answers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id           UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id          UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    selected_option_ids  JSONB NOT NULL DEFAULT '[]'::jsonb,
    text_answer          TEXT NOT NULL DEFAULT '',
    is_correct           BOOLEAN,
    points_awarded       INTEGER NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT quiz_attempt_answers_points_nonneg CHECK (points_awarded >= 0)
);

CREATE UNIQUE INDEX quiz_attempt_answers_attempt_question_uidx
    ON quiz_attempt_answers (attempt_id, question_id);

CREATE TRIGGER quiz_attempt_answers_set_updated_at
    BEFORE UPDATE ON quiz_attempt_answers
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Optional link from curriculum quiz_ref blocks to quizzes.
ALTER TABLE content_blocks
    ADD CONSTRAINT content_blocks_quiz_ref_fk
    FOREIGN KEY (quiz_ref_id) REFERENCES quizzes(id) ON DELETE SET NULL;

CREATE TABLE assignments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id   UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    max_score   INTEGER NOT NULL DEFAULT 100,
    due_at      TIMESTAMPTZ,
    allow_late  BOOLEAN NOT NULL DEFAULT FALSE,
    rubric      JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assignments_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT assignments_status_valid CHECK (status IN ('draft', 'published')),
    CONSTRAINT assignments_max_score_pos CHECK (max_score > 0)
);

CREATE INDEX assignments_course_id_idx ON assignments (course_id);

CREATE TRIGGER assignments_set_updated_at
    BEFORE UPDATE ON assignments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE assignment_submissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id   UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'draft',
    body            TEXT NOT NULL DEFAULT '',
    attachment_url  TEXT NOT NULL DEFAULT '',
    score           INTEGER,
    feedback        TEXT NOT NULL DEFAULT '',
    rubric_scores   JSONB NOT NULL DEFAULT '{}'::jsonb,
    submitted_at    TIMESTAMPTZ,
    graded_at       TIMESTAMPTZ,
    graded_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT assignment_submissions_status_valid CHECK (
        status IN ('draft', 'submitted', 'graded', 'returned')
    ),
    CONSTRAINT assignment_submissions_score_nonneg CHECK (score IS NULL OR score >= 0)
);

CREATE UNIQUE INDEX assignment_submissions_assignment_user_uidx
    ON assignment_submissions (assignment_id, user_id);
CREATE INDEX assignment_submissions_user_id_idx ON assignment_submissions (user_id);

CREATE TRIGGER assignment_submissions_set_updated_at
    BEFORE UPDATE ON assignment_submissions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
