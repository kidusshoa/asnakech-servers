# Asnakech School Servers

Backend API for the Asnakech online education platform. Built with Go and Gin as a **modular monolith** (domain → service → repository → handler).

> Status: foundation stages in progress (architecture, git docs, local stack). Education domains come next.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Docker + Docker Compose (optional, for local Postgres / Redis / MinIO)

### Installation

```bash
git clone https://github.com/kidusshoa/asnakech-servers.git
cd asnakech-servers
cp .env.example .env
make install
make tools   # optional: installs Air for hot-reload
```

### Running (API on host)

```bash
make up-infra          # Postgres, Redis, MinIO
make migrate-up        # apply SQL migrations (roles seed)
make tools && make dev # Air hot-reload, or go run fallback
```

### Running (full stack in Docker)

```bash
cp .env.example .env
make up
make logs
```

Server listens on `http://localhost:8080` by default. MinIO console defaults to `http://localhost:9011` (minioadmin / minioadmin).

If Compose fails with **port is already allocated**, another stack is using that host port (e.g. Redis on `6379`). Change `REDIS_HOST_PORT` / `MINIO_*_HOST_PORT` / `POSTGRES_HOST_PORT` in `.env` and keep `REDIS_URL` / `S3_ENDPOINT` / `DATABASE_URL` in sync.

## Project Structure

```
.
├── cmd/api/                 # Process entrypoint
├── internal/
│   ├── apperr/              # Typed application errors
│   ├── config/              # Env + optional .env loading
│   ├── database/            # Postgres pool helpers
│   ├── domain/              # Business entities (growing)
│   ├── handlers/            # HTTP handlers (thin)
│   ├── logging/             # Zerolog setup
│   ├── middleware/          # CORS, request ID, request logs
│   ├── platform/ready/      # Dependency readiness checks
│   ├── repository/          # Persistence interfaces
│   │   └── postgres/        # Postgres implementations
│   ├── response/            # Standard JSON envelope
│   ├── server/              # HTTP server wiring & routes
│   └── service/             # Use-cases / workflows
├── migrations/              # golang-migrate SQL (up/down)
├── docs/
│   ├── api/                 # Envelope, versioning, API changelog
│   ├── swagger/             # Generated OpenAPI (swag)
│   ├── git/                 # Contributing, branching, commits, releases
│   └── db/                  # Schema conventions
├── .github/                 # PR / issue templates, CODEOWNERS
├── Dockerfile
├── docker-compose.yml
├── .air.toml
├── .env.example
├── CHANGELOG.md
├── go.mod
├── Makefile
└── README.md
```

### Layering rules

| Layer | May depend on | Must not depend on |
|-------|---------------|--------------------|
| `handlers` | `service`, `response`, `apperr` | `repository` implementations, SQL |
| `service` | `domain`, `repository` interfaces, `apperr` | Gin / HTTP |
| `repository` | `domain` | Gin / handlers |
| `domain` | stdlib only | Gin, SQL, services |

## Available Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/` |
| `make run` | Build and run |
| `make test` | Run all tests |
| `make clean` | Remove build artifacts and `tmp/` |
| `make install` | Download modules |
| `make tools` | Install Air + swag CLI |
| `make dev` | Hot-reload with Air, or `go run` |
| `make up` | Docker Compose: API + infra |
| `make up-infra` | Postgres + Redis + MinIO only |
| `make down` | Stop Compose stack |
| `make logs` | Follow API container logs |
| `make migrate-up` | Apply all pending DB migrations |
| `make migrate-down` | Roll back one migration |
| `make migrate-version` | Show current migration version |
| `make migrate-create NAME=…` | Scaffold a new migration pair |
| `make docs` | Regenerate OpenAPI into `docs/swagger/` |
| `make docs-check` | Fail if swagger output is stale |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness / version |
| `GET` | `/ready` | Readiness (Postgres, Redis, S3 when configured) |
| `GET` | `/swagger/index.html` | Interactive OpenAPI (Swagger UI) |
| `GET` | `/api/v1/` | Welcome stub |
| `GET` | `/api/v1/roles` | List seeded platform roles (requires DB) |
| `POST` | `/api/v1/auth/register` | Create account (student) |
| `POST` | `/api/v1/auth/login` | Login |
| `POST` | `/api/v1/auth/refresh` | Rotate tokens |
| `POST` | `/api/v1/auth/logout` | Revoke refresh token |
| `GET` | `/api/v1/auth/me` | Current user (Bearer) |
| `POST` | `/api/v1/auth/forgot-password` | Request password reset |
| `POST` | `/api/v1/auth/reset-password` | Set new password |
| `POST` | `/api/v1/auth/verify-email` | Confirm email |
| `GET/PATCH` | `/api/v1/users/me` | Profile (Bearer) |
| `PUT` | `/api/v1/users/me/avatar` | Set avatar URL |
| `GET` | `/api/v1/admin/users` | List users (admin) |
| `POST/GET` | `/api/v1/organizations` | Create / list my schools |
| `POST` | `/api/v1/organizations/invites/accept` | Accept invite |
| `GET` | `/api/v1/categories` | Course categories |
| `GET/POST` | `/api/v1/courses` | Catalog list / create (teacher) |
| `GET` | `/api/v1/courses/:id/curriculum` | Nested modules → lessons → blocks |
| `POST/DELETE` | `/api/v1/courses/:id/enroll` | Enroll / unenroll |
| `GET` | `/api/v1/me/enrollments` | My enrollments |
| `PUT/GET` | `/api/v1/lessons/:id/progress` | Lesson progress |
| `GET` | `/api/v1/me/progress` | Progress dashboard |
| `POST/GET` | `/api/v1/courses/:id/quizzes` | Quizzes |
| `GET` | `/api/v1/courses/:id/gradebook` | Teacher gradebook |
| `POST` | `/api/v1/media/uploads` | Presigned upload intent |
| `GET` | `/api/v1/me/calendar` | Live session calendar feed |
| `GET` | `/api/v1/me/notifications` | In-app notification feed |

Human-readable API guides: **[docs/api/](docs/api/README.md)** (incl. [courses](docs/api/courses.md), [curriculum](docs/api/curriculum.md), [enrollment](docs/api/enrollment.md), [progress](docs/api/progress.md), [assessments](docs/api/assessments.md), [media](docs/api/media.md), [live](docs/api/live.md), [communication](docs/api/communication.md)).

Responses use a shared envelope:

```json
{
  "success": true,
  "data": { },
  "error": null,
  "meta": { }
}
```

Every response includes an `X-Request-ID` header (generated if the client did not send one). Logs are structured JSON in production and console-formatted in development.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `ENV` | `development` | `development` or `production` |
| `APP_VERSION` | `0.1.0` | Reported in health/welcome |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated origins |
| `CORS_ALLOW_CREDENTIALS` | `false` | Must stay false when origin is `*` |
| `DATABASE_URL` | _(empty)_ | Postgres URL; required in production; enables `/roles` |
| `REDIS_URL` | _(empty)_ | Redis URL; skipped in `/ready` if empty |
| `JWT_SECRET` | _(empty)_ | Required when `ENV=production` |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `S3_ENDPOINT` | _(empty)_ | MinIO/S3 endpoint; TCP-checked in `/ready` |
| `S3_REGION` | `us-east-1` | Bucket region |
| `S3_BUCKET` | `asnakech` | Default bucket |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | | Object storage credentials |
| `S3_USE_PATH_STYLE` | `true` | Path-style URLs (MinIO) |
| `S3_PUBLIC_BASE_URL` | _(empty)_ | Optional CDN base for `public_url` |
| `MEDIA_PRESIGN_TTL` | `15m` | Presigned upload URL lifetime |
| `LIVE_DEFAULT_PROVIDER` | `custom` | Default video provider (`jitsi`, `zoom`, `google_meet`) |
| `LIVE_JITSI_BASE_URL` | `https://meet.jit.si` | Jitsi room base URL |

Copy `.env.example` → `.env` for local defaults matching Compose.

## Roadmap

Expansion is tracked in 21 stages (architecture → production hardening), covering auth, courses, enrollment, assessments, media, live classes, payments, analytics, OpenAPI, and git/release docs.

## Contributing

See **[docs/git/](docs/git/README.md)** for the full workflow:

- [Contributing](docs/git/CONTRIBUTING.md)
- [Branching](docs/git/BRANCHING.md)
- [Commit convention](docs/git/COMMIT_CONVENTION.md)
- [Releases](docs/git/RELEASE.md)

Short version: branch from `master` → Conventional Commits → PR (squash-merge).

Notable changes are listed in [CHANGELOG.md](CHANGELOG.md).

## License

MIT — see [LICENSE](LICENSE).
