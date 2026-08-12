# Assessments

Quizzes (auto-graded) and assignments (manual grading), plus a teacher gradebook.

## Quizzes

| Field | Notes |
|-------|-------|
| Question types | `mcq`, `short_answer` |
| MCQ options | `{ id, text, is_correct }` — secrets stripped for students until graded |
| Short answer | Case-insensitive exact match on `correct_answer` |
| Attempts | Optional `max_attempts`; in-progress attempt is resumed |
| Time limit | Optional `time_limit_seconds` from attempt start |
| Pass | `percent >= pass_percent` after submit |

Submit auto-grades and sets attempt status to `graded`.

### Curriculum link

Content blocks of type `quiz_ref` may set `quiz_ref_id` to a quiz UUID (FK enforced).

## Assignments

| Field | Notes |
|-------|-------|
| Rubric | `[{ id, criterion, max_points }]` (v1 simple) |
| Due | `due_at` + `allow_late` |
| Submission | One per student; draft → submitted → graded |

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST/GET` | `/api/v1/courses/:id/quizzes` | write / read (enrolled sees published) |
| `GET/PATCH` | `/api/v1/quizzes/:quizId` | read / write |
| `POST` | `/api/v1/quizzes/:quizId/publish` | teacher |
| `POST` | `/api/v1/quizzes/:quizId/questions` | teacher |
| `PUT` | `/api/v1/quizzes/:quizId/questions/reorder` | teacher |
| `PATCH/DELETE` | `/api/v1/questions/:questionId` | teacher |
| `POST/GET` | `/api/v1/quizzes/:quizId/attempts` | enrollee start / list mine |
| `GET` | `/api/v1/attempts/:attemptId` | owner or teacher |
| `PUT` | `/api/v1/attempts/:attemptId/answers` | owner |
| `POST` | `/api/v1/attempts/:attemptId/submit` | owner |
| `POST/GET` | `/api/v1/courses/:id/assignments` | write / read |
| `GET/PATCH` | `/api/v1/assignments/:assignmentId` | read / write |
| `POST` | `/api/v1/assignments/:assignmentId/publish` | teacher |
| `PUT/GET` | `/api/v1/assignments/:assignmentId/submission` | enrollee |
| `GET` | `/api/v1/assignments/:assignmentId/submissions` | teacher |
| `POST` | `/api/v1/submissions/:submissionId/grade` | teacher |
| `GET` | `/api/v1/courses/:id/gradebook` | teacher |
