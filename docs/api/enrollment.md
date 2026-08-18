# Enrollments

Learners join published courses via enrollment. Content blocks in the curriculum tree are only returned to **active** enrollees, the course teacher, or a platform admin.

## Status lifecycle

`active` ↔ `waitlisted` → `cancelled` (re-enroll from cancelled reactivates)

| Status | Meaning |
|--------|---------|
| `active` | Has a seat; can access lesson content |
| `waitlisted` | Waiting for capacity; no content access |
| `cancelled` | Left the course |

## Capacity & waitlist

Set on the course via `PATCH /courses/:id/enrollment-settings`:

| Field | Notes |
|-------|-------|
| `enrollment_capacity` | Max active seats; omit to leave unchanged; `0` clears (unlimited) |
| `enrollment_open` | When `false`, self-enroll requires an invite code |
| `waitlist_enabled` | If full and true → waitlist; if full and false → `409` |

Unenrolling an active seat auto-promotes the oldest waitlisted learner.

## Invite codes

Teachers create shareable codes. Codes may have `max_uses` and `expires_at`. Closed courses still accept valid invite codes.

## Events

Each status change appends to `enrollment_events` (`enrolled`, `waitlisted`, `activated`, `cancelled`) for later notifications (Stage 16).

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST` | `/api/v1/courses/:id/enroll` | authenticated (`courses:read`); body optional `{ "invite_code" }` |
| `DELETE` | `/api/v1/courses/:id/enroll` | self unenroll |
| `GET` | `/api/v1/me/enrollments` | list mine (`status`, `page`, `per_page`) |
| `GET` | `/api/v1/courses/:id/enrollments` | teacher owner or admin |
| `GET` | `/api/v1/courses/:id/access` | `{ can_access_content, enrollment, … }` |
| `PATCH` | `/api/v1/courses/:id/enrollment-settings` | teacher owner or admin |
| `POST` | `/api/v1/courses/:id/invite-codes` | teacher owner or admin |
| `GET` | `/api/v1/courses/:id/invite-codes` | teacher owner or admin |
| `DELETE` | `/api/v1/courses/:id/invite-codes/:codeId` | revoke |

## Paid courses

When `price_cents > 0`, direct enroll returns validation error — use [checkout](./payments.md) (`POST /courses/:id/checkout`). Successful payment creates enrollment with `source=payment`.
