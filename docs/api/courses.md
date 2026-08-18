# Course catalog

Courses are catalog entries authored by teachers. See [curriculum.md](./curriculum.md) for modules, lessons, and content blocks.

## Status lifecycle

`draft` → `published` → `archived` (also soft-delete sets archived)

- **Published** courses are visible to everyone
- **Draft/archived** visible to the teacher owner or platform admin

## Pricing metadata

`price_cents` + `currency` (ISO-like 3-letter, default `ETB`). `0` means free (enroll directly). Paid courses use [checkout](./payments.md).

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `GET` | `/api/v1/categories` | public |
| `POST` | `/api/v1/categories` | `courses:manage` (admin) |
| `GET` | `/api/v1/courses` | public (+ optional Bearer for drafts) |
| `POST` | `/api/v1/courses` | `courses:write` |
| `GET` | `/api/v1/courses/:id` | public published / owner drafts |
| `PATCH` | `/api/v1/courses/:id` | teacher owner or admin |
| `PUT` | `/api/v1/courses/:id/tags` | teacher owner or admin |
| `POST` | `/api/v1/courses/:id/publish` | teacher owner or admin |
| `POST` | `/api/v1/courses/:id/archive` | teacher owner or admin |
| `DELETE` | `/api/v1/courses/:id` | teacher owner or admin |

### List filters

`page`, `per_page`, `q`, `category` (slug), `tag` (slug), `organization_id`, `teacher_id`, `level`, `status`

## Seeded categories

Mathematics, Science, Languages, Technology.

Promote a user to `teacher` (or `admin`) before creating courses:

```sql
UPDATE users
SET role_id = (SELECT id FROM roles WHERE code = 'teacher')
WHERE email = 'you@example.com';
```
