# Communication

Announcements, course discussions, direct messages, and an in-app notification outbox.

## Announcements

Teachers create draft announcements, publish to notify enrolled students (in-app outbox).

| Status | Students see |
|--------|----------------|
| `draft` | No (teacher only) |
| `published` | Yes |

## Discussions

Threads with nested replies (`parent_id`). Teachers can lock threads. New threads notify the course teacher; replies notify participants.

## Direct messages

1:1 conversations keyed by user pair. Messages enqueue an in-app notification for the recipient.

## Notification outbox

| Channel | v1 behavior |
|---------|-------------|
| `in_app` | Stored immediately (`status=sent`); list via `/me/notifications` |
| `email` | Queued as `pending` for a future worker |

Events: `announcement.published`, `discussion.new_thread`, `discussion.reply`, `dm.message`.

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST/GET` | `/api/v1/courses/:id/announcements` | teacher / enrolled |
| `GET/PATCH/DELETE` | `/api/v1/announcements/:id` | read / teacher |
| `POST` | `/api/v1/announcements/:id/publish` | teacher |
| `POST/GET` | `/api/v1/courses/:id/threads` | enrolled + teacher |
| `GET` | `/api/v1/threads/:threadId` | course access |
| `POST` | `/api/v1/threads/:threadId/lock` | teacher |
| `POST/GET` | `/api/v1/threads/:threadId/posts` | enrolled |
| `PATCH/DELETE` | `/api/v1/posts/:postId` | author or teacher |
| `POST` | `/api/v1/conversations` | start DM |
| `GET` | `/api/v1/me/conversations` | list mine |
| `GET/POST` | `/api/v1/conversations/:id/messages` | participant |
| `POST` | `/api/v1/conversations/:id/read` | participant |
| `GET` | `/api/v1/me/notifications` | in-app feed |
| `POST` | `/api/v1/notifications/:id/read` | mark read |
| `POST` | `/api/v1/me/notifications/read-all` | mark all read |
