CREATE TABLE certificates (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id           UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    verification_code   TEXT NOT NULL,
    learner_name        TEXT NOT NULL,
    course_title        TEXT NOT NULL,
    storage_key         TEXT NOT NULL DEFAULT '',
    public_url          TEXT NOT NULL DEFAULT '',
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT certificates_verification_code_nonempty CHECK (char_length(trim(verification_code)) > 0),
    CONSTRAINT certificates_course_user_uidx UNIQUE (course_id, user_id)
);

CREATE UNIQUE INDEX certificates_verification_code_uidx ON certificates (verification_code);
CREATE INDEX certificates_user_id_idx ON certificates (user_id, issued_at DESC);
CREATE INDEX certificates_course_id_idx ON certificates (course_id);

CREATE TRIGGER certificates_set_updated_at
    BEFORE UPDATE ON certificates
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
