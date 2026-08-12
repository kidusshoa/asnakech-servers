# Progress & completion

Tracks per-lesson progress and aggregates course completion for active enrollees.

## Rules

| Rule | Behavior |
|------|----------|
| Access | Active enrollment required |
| Published only | Progress only on published lessons |
| Prerequisites | If a lesson has `prerequisite_lesson_id`, that lesson must be `completed` first |
| Course % | `completed_published_lessons / total_published_lessons * 100` |
| Course complete | All published lessons completed (`completed_at` set) |
| Idempotent writes | `PUT` upserts; `percent` only moves forward; repeating `completed: true` is safe |

## Lesson write body

```json
{
  "percent": 40,
  "last_position": "block:uuid-or-t=120",
  "completed": false
}
```

- `last_position` — opaque client bookmark (block id, timestamp, scroll offset)
- `completed: true` forces `percent=100` and status `completed`

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `PUT` | `/api/v1/lessons/:lessonId/progress` | active enrollee |
| `GET` | `/api/v1/lessons/:lessonId/progress` | active enrollee (zeros if none yet) |
| `GET` | `/api/v1/courses/:id/progress` | active enrollee (course + lesson rows) |
| `GET` | `/api/v1/me/progress` | dashboard of my course progress |

Certificates (Stage 17) will consume `course_progress.completed_at`.
