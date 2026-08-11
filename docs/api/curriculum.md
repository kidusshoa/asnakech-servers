# Curriculum

Course structure: **modules → lessons → content blocks**.

## Visibility

| Viewer | Sees |
|--------|------|
| Public / student (published course) | Published lessons only (+ their blocks) |
| Course teacher or platform admin | All modules/lessons/blocks (draft + published) |
| Draft/archived course (non-owner) | `404` |

Lesson status is independent of course publish: `draft` → `published`.

## Content block types

| Type | Fields used |
|------|-------------|
| `text` | `body` |
| `video` | `media_url`, optional `title`/`body` |
| `file` | `media_url`, optional `title` |
| `quiz_ref` | `quiz_ref_id` (stub until assessments stage) |

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `GET` | `/api/v1/courses/:id/curriculum` | public published tree / owner drafts |
| `POST` | `/api/v1/courses/:id/modules` | teacher owner or admin (`courses:write`) |
| `PUT` | `/api/v1/courses/:id/modules/reorder` | teacher owner or admin |
| `PATCH` | `/api/v1/modules/:moduleId` | teacher owner or admin |
| `DELETE` | `/api/v1/modules/:moduleId` | teacher owner or admin |
| `POST` | `/api/v1/modules/:moduleId/lessons` | teacher owner or admin |
| `PUT` | `/api/v1/modules/:moduleId/lessons/reorder` | teacher owner or admin |
| `PATCH` | `/api/v1/lessons/:lessonId` | teacher owner or admin |
| `POST` | `/api/v1/lessons/:lessonId/publish` | teacher owner or admin |
| `POST` | `/api/v1/lessons/:lessonId/unpublish` | teacher owner or admin |
| `DELETE` | `/api/v1/lessons/:lessonId` | teacher owner or admin |
| `POST` | `/api/v1/lessons/:lessonId/blocks` | teacher owner or admin |
| `PUT` | `/api/v1/lessons/:lessonId/blocks/reorder` | teacher owner or admin |
| `PATCH` | `/api/v1/blocks/:blockId` | teacher owner or admin |
| `DELETE` | `/api/v1/blocks/:blockId` | teacher owner or admin |

### Reorder body

```json
{ "ids": ["uuid-1", "uuid-2", "uuid-3"] }
```

IDs must be the full set for that parent, in the desired order.

## Tree shape

`GET .../curriculum` returns nested modules → lessons → blocks (ordered by `position`).
