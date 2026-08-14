# Live learning & calendar

Scheduled live class sessions with join links, attendance, and a personal calendar feed.

## Session lifecycle

| Status | Visible to students |
|--------|---------------------|
| `draft` | No |
| `scheduled` | Yes |
| `completed` | Yes (history) |
| `cancelled` | No |

Flow: create (draft) → publish → students join/check-in → complete (or cancel).

## Providers

| Provider | v1 behavior |
|----------|-------------|
| `custom` | Paste `join_url` / optional `host_url` |
| `jitsi` | Auto-generates room at `LIVE_JITSI_BASE_URL` |
| `zoom` | Manual URL (API adapter stub) |
| `google_meet` | Manual URL (API adapter stub) |

`POST /sessions/:id/generate-link` runs the provider adapter. Publish auto-generates for `jitsi` when `join_url` is empty.

Students receive join URLs via `GET /sessions/:id/join`, not in list/get responses.

## Attendance

| Status | Meaning |
|--------|---------|
| `registered` | Reserved seat |
| `present` | On time check-in or teacher mark |
| `late` | Self check-in after grace period |
| `absent` | Default / teacher mark |
| `excused` | Teacher mark |

Self check-in window: 15 minutes before start through 30 minutes after end.

## Calendar

`GET /api/v1/me/calendar?from=2026-08-01T00:00:00Z&to=2026-09-01T00:00:00Z`

Returns scheduled/completed sessions for courses you teach or are actively enrolled in.

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST/GET` | `/api/v1/courses/:id/sessions` | teacher / enrolled list |
| `GET/PATCH` | `/api/v1/sessions/:sessionId` | read / teacher |
| `POST` | `/api/v1/sessions/:sessionId/publish` | teacher |
| `POST` | `/api/v1/sessions/:sessionId/complete` | teacher |
| `POST` | `/api/v1/sessions/:sessionId/cancel` | teacher |
| `POST` | `/api/v1/sessions/:sessionId/generate-link` | teacher |
| `GET` | `/api/v1/sessions/:sessionId/join` | teacher or enrolled |
| `GET` | `/api/v1/sessions/:sessionId/attendance` | teacher |
| `PUT` | `/api/v1/sessions/:sessionId/attendance/:userId` | teacher |
| `POST` | `/api/v1/sessions/:sessionId/attendance/check-in` | enrolled |
| `GET` | `/api/v1/me/calendar` | authenticated |

Env: `LIVE_DEFAULT_PROVIDER`, `LIVE_JITSI_BASE_URL`.
