# API changelog

Client-facing HTTP API changes only. For the full project changelog see [../../CHANGELOG.md](../../CHANGELOG.md).

## Unreleased

### Added

- OpenAPI/Swagger UI at `/swagger/index.html`
- Documented envelope, versioning, and regeneration workflow
- Endpoints: `GET /health`, `GET /ready`, `GET /api/v1/`, `GET /api/v1/roles`
- Auth: register, login, refresh, logout, me, forgot/reset password, verify email
- Profiles + RBAC: `/users/me`, avatar hook, admin user CRUD, permission matrix
- Organizations: create/list, members, invites/accept
- Course catalog: categories, CRUD, tags, publish/archive, list filters
- Curriculum: modules → lessons → content blocks, reorder, lesson publish
- Enrollments: self enroll/unenroll, capacity/waitlist, invite codes, content access gate
- Progress: lesson upsert (idempotent), course %, completion, student dashboard
- Assessments: MCQ/short-answer quizzes, assignments + rubric, auto/manual grade, gradebook
- Media: presigned S3/MinIO uploads, purpose limits, video metadata, virus-scan hook, CDN URLs
