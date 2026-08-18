# Discovery & platform polish

Search, recommendations, i18n, feature flags, and parent/guardian linking.

## Search

Unified full-text discovery:

`GET /api/v1/search?q=math&type=all`

| Param | Description |
|-------|-------------|
| `q` | Required query (Postgres `tsvector` on course title/summary/description) |
| `type` | `all` (default), `courses`, `categories`, `teachers` |
| `language` | Filter course hits by language code |
| `level` | Filter course hits by level |
| `page`, `per_page` | Pagination for course hits |

Course list (`GET /courses?q=`) also uses the same FTS index.

## Recommendations

`GET /api/v1/me/recommendations` — popular published courses not yet enrolled, preferring the user's locale (`Accept-Language` / profile locale).

## i18n

| Endpoint | Purpose |
|----------|---------|
| `GET /api/v1/locales` | Supported locale codes (`en`, `am`) |
| `GET /api/v1/i18n/messages?lang=am` | UI string bundle for clients |

Locale resolution order: `?lang=` → `Accept-Language` → `en`. Response includes `Content-Language` header.

## Feature flags

`GET /api/v1/features` — boolean map for client gating. Override via `FEATURE_FLAGS` env (comma-separated; prefix `!` to disable).

## Parent / guardian links

Parents (or admins) link student accounts by email:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/me/children/link` | `{ "student_email" }` |
| `GET` | `/api/v1/me/children` | List linked students |
| `DELETE` | `/api/v1/me/children/:studentId` | Revoke link |

Promote a user to `parent` role locally:

```bash
# SQL or assign via admin PATCH /admin/users/:id with role=parent
```

## Endpoints summary

| Method | Path | Access |
|--------|------|--------|
| `GET` | `/api/v1/search` | public |
| `GET` | `/api/v1/features` | public |
| `GET` | `/api/v1/locales` | public |
| `GET` | `/api/v1/i18n/messages` | public |
| `GET` | `/api/v1/me/recommendations` | authenticated |
| `POST` | `/api/v1/me/children/link` | parent / admin |
| `GET` | `/api/v1/me/children` | parent / admin |
| `DELETE` | `/api/v1/me/children/:studentId` | parent / admin |
