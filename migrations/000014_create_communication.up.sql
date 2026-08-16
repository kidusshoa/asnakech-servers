CREATE TABLE announcements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title           TEXT NOT NULL,
    body            TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'draft',
    pinned          BOOLEAN NOT NULL DEFAULT FALSE,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT announcements_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT announcements_status_valid CHECK (status IN ('draft', 'published'))
);

CREATE INDEX announcements_course_id_idx ON announcements (course_id, created_at DESC);

CREATE TRIGGER announcements_set_updated_at
    BEFORE UPDATE ON announcements
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE discussion_threads (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title           TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'open',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT discussion_threads_title_nonempty CHECK (char_length(trim(title)) > 0),
    CONSTRAINT discussion_threads_status_valid CHECK (status IN ('open', 'locked'))
);

CREATE INDEX discussion_threads_course_id_idx ON discussion_threads (course_id, created_at DESC);

CREATE TRIGGER discussion_threads_set_updated_at
    BEFORE UPDATE ON discussion_threads
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE discussion_posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id       UUID NOT NULL REFERENCES discussion_threads(id) ON DELETE CASCADE,
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_id       UUID REFERENCES discussion_posts(id) ON DELETE CASCADE,
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT discussion_posts_body_nonempty CHECK (char_length(trim(body)) > 0)
);

CREATE INDEX discussion_posts_thread_id_idx ON discussion_posts (thread_id, created_at ASC);

CREATE TRIGGER discussion_posts_set_updated_at
    BEFORE UPDATE ON discussion_posts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE dm_conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_a_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT dm_conversations_distinct_users CHECK (user_a_id <> user_b_id),
    CONSTRAINT dm_conversations_pair_uidx UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX dm_conversations_user_a_idx ON dm_conversations (user_a_id);
CREATE INDEX dm_conversations_user_b_idx ON dm_conversations (user_b_id);

CREATE TRIGGER dm_conversations_set_updated_at
    BEFORE UPDATE ON dm_conversations
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE dm_messages (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id     UUID NOT NULL REFERENCES dm_conversations(id) ON DELETE CASCADE,
    sender_id           UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body                TEXT NOT NULL,
    read_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT dm_messages_body_nonempty CHECK (char_length(trim(body)) > 0)
);

CREATE INDEX dm_messages_conversation_id_idx ON dm_messages (conversation_id, created_at ASC);

CREATE TABLE notification_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel         TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    subject         TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL DEFAULT '',
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'pending',
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    CONSTRAINT notification_outbox_channel_valid CHECK (channel IN ('in_app', 'email')),
    CONSTRAINT notification_outbox_status_valid CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX notification_outbox_user_id_idx ON notification_outbox (user_id, created_at DESC);
CREATE INDEX notification_outbox_pending_idx ON notification_outbox (status) WHERE status = 'pending';
