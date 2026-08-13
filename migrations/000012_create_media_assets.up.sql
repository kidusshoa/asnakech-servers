CREATE TABLE media_assets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id         UUID REFERENCES courses(id) ON DELETE SET NULL,
    purpose           TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending',
    content_type      TEXT NOT NULL,
    original_filename TEXT NOT NULL DEFAULT '',
    size_bytes        BIGINT NOT NULL DEFAULT 0,
    storage_key       TEXT NOT NULL,
    public_url        TEXT NOT NULL DEFAULT '',
    duration_seconds  INTEGER,
    width             INTEGER,
    height            INTEGER,
    scan_status       TEXT NOT NULL DEFAULT 'pending',
    scan_note         TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT media_assets_purpose_valid CHECK (
        purpose IN ('avatar', 'course_cover', 'lesson_media', 'assignment_attachment', 'general')
    ),
    CONSTRAINT media_assets_status_valid CHECK (
        status IN ('pending', 'uploaded', 'ready', 'rejected', 'deleted')
    ),
    CONSTRAINT media_assets_scan_status_valid CHECK (
        scan_status IN ('pending', 'clean', 'infected', 'skipped')
    ),
    CONSTRAINT media_assets_size_nonneg CHECK (size_bytes >= 0),
    CONSTRAINT media_assets_content_type_nonempty CHECK (char_length(trim(content_type)) > 0),
    CONSTRAINT media_assets_storage_key_nonempty CHECK (char_length(trim(storage_key)) > 0)
);

CREATE UNIQUE INDEX media_assets_storage_key_uidx ON media_assets (storage_key);
CREATE INDEX media_assets_owner_id_idx ON media_assets (owner_id);
CREATE INDEX media_assets_course_id_idx ON media_assets (course_id);
CREATE INDEX media_assets_status_idx ON media_assets (status);

CREATE TRIGGER media_assets_set_updated_at
    BEFORE UPDATE ON media_assets
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
