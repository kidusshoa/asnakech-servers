# API changelog

Client-facing HTTP API changes only. For the full project changelog see [../../CHANGELOG.md](../../CHANGELOG.md).

## Unreleased

### Added

- OpenAPI/Swagger UI at `/swagger/index.html`
- Documented envelope, versioning, and regeneration workflow
- Endpoints: `GET /health`, `GET /ready`, `GET /api/v1/`, `GET /api/v1/roles`
- Auth: register, login, refresh, logout, me, forgot/reset password, verify email
- Profiles + RBAC: `/users/me`, avatar hook, admin user CRUD, permission matrix
