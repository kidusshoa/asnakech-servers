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

### Changed

### Fixed

### Security

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
