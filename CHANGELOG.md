# Changelog

All notable changes to this project are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Git collaboration docs under `docs/git/` (contributing, branching, commits, releases)
- GitHub PR template, issue templates, and `CODEOWNERS`
- `.env` loading via godotenv; config for DB, Redis, JWT, S3, and log level
- Structured logging with zerolog (console in dev, JSON in production)
- `GET /ready` readiness probe (Postgres, Redis, S3 endpoint when configured)
- Docker multi-stage `Dockerfile` and `docker-compose.yml` (API, Postgres, Redis, MinIO)
- Air hot-reload (`.air.toml`) and Makefile targets: `tools`, `up`, `up-infra`, `down`, `logs`
- PostgreSQL migrations (golang-migrate), `roles` table + seed, schema conventions in `docs/db/`
- Shared pgx pool, role repository/service, and `GET /api/v1/roles`
- OpenAPI via swag: `docs/swagger/`, Swagger UI at `/swagger/index.html`, guides in `docs/api/`
- Configurable Compose host ports (`REDIS_HOST_PORT`, `MINIO_*_HOST_PORT`, …) to avoid local conflicts
- Auth: users + token tables, JWT access/refresh, register/login/refresh/logout/me, password reset & email verify stubs, auth rate limiting
- Profiles + RBAC middleware, admin user management, `docs/api/rbac.md`
- Organizations (schools): memberships, invites, org-scoped manage checks
- Course catalog: categories/tags, draft→publish→archive, pricing metadata, filters
- Curriculum: modules, lessons, content blocks (`text`/`video`/`file`/`quiz_ref`), reorder
- Enrollments: active/waitlisted/cancelled, invite codes, capacity, waitlist promotion, enrollment events
- Progress: lesson progress + course aggregates, prerequisites, idempotent upserts, `/me/progress` dashboard
- Assessments: quizzes (MCQ/short answer, attempts, auto-grade), assignments (submissions, rubric), gradebook
- Media: `media_assets`, presigned PUT uploads, attachment limits, scan hook, avatar upload intent
- Live: `live_sessions`, `session_attendance`, Jitsi/custom/Zoom/Meet adapters, calendar feed
- Communication: announcements, threads/posts, DMs, `notification_outbox` (in-app + email queue)
- Certificates: `certificates` table, PDF download, public verify, transcript/grade summary
- Payments: orders, coupons, manual/Stripe/Chapa adapters, webhooks with idempotency, paid-course checkout
- Analytics: admin overview, enrollment/revenue/user reports, per-course teacher analytics
- Discovery: FTS search, recommendations, i18n (en/am), feature flags, parent-student links
- Production ops: security headers, global rate limits, Prometheus `/metrics`, GitHub Actions CI
- Ops docs: `docs/ops/RUNBOOK.md`, API deprecation policy

### Changed

- Dockerfile builder image updated to Go 1.25 (matches `go.mod`)
- Auth rate limits configurable via `RATE_LIMIT_AUTH_*` env vars

### Fixed

### Security

- Baseline security headers on all responses (`X-Content-Type-Options`, CSP, frame denial, optional HSTS)

## [0.1.0] - 2026-08-10

### Added

- Modular monolith layout: `domain`, `service`, `repository`, `handlers`, `server`
- Shared `apperr` error types and JSON `response` envelope
- CORS and request-ID middleware (generates ID when client omits it)
- Health and API welcome endpoints with version from config
- Config for `APP_VERSION`, `CORS_ALLOWED_ORIGINS`, `CORS_ALLOW_CREDENTIALS`
- Middleware and `apperr` unit tests

### Changed

- README aligned with real project structure (no longer claims missing middleware/hot-reload)

[Unreleased]: https://github.com/kidusshoa/asnakech-servers/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/kidusshoa/asnakech-servers/releases/tag/v0.1.0
