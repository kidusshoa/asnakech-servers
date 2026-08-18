# Asnakech School Servers

Backend API for the **Asnakech** online education platform — schools, teachers, students, and parents on a single Go service.

Built as a **modular monolith**: `handlers` → `service` → `repository` → Postgres, with a shared JSON envelope, JWT auth, and RBAC.

| | |
|---|---|
| **Stack** | Go 1.25 · Gin · pgx · golang-migrate · zerolog · S3/MinIO |
| **API** | REST `/api/v1` · OpenAPI at `/swagger/index.html` |
| **Status** | 21-stage roadmap **complete** (foundation → production hardening) |
| **Migrations** | 17 SQL pairs under `migrations/` |

---

## Quick start

### Prerequisites

- Go **1.25+**
- Git
- Docker + Docker Compose (recommended for Postgres, Redis, MinIO)

### Install & run (API on host)

```bash
git clone https://github.com/kidusshoa/asnakech-servers.git
cd asnakech-servers
cp .env.example .env
make install
make up-infra          # Postgres, Redis, MinIO
make migrate-up        # apply all migrations
make tools && make dev # Air hot-reload (or go run fallback)
```

API: **http://localhost:8080** · MinIO console: **http://localhost:9011** (`minioadmin` / `minioadmin`)

### Full stack in Docker

```bash
cp .env.example .env
make up
make logs
```

**Port conflicts:** if Compose reports *port already allocated*, change `POSTGRES_HOST_PORT`, `REDIS_HOST_PORT`, or `MINIO_*_HOST_PORT` in `.env` and keep `DATABASE_URL`, `REDIS_URL`, and `S3_ENDPOINT` in sync.

### Smoke checks

```bash
curl -s localhost:8080/health | jq
curl -s localhost:8080/ready  | jq
curl -s localhost:8080/metrics | head
open http://localhost:8080/swagger/index.html
```

---

## Platform capabilities

| Domain | Highlights |
|--------|------------|
| **Auth & profiles** | Register/login, JWT access + refresh, password reset & email verify stubs, `/users/me`, avatar uploads |
| **RBAC & admin** | Role permissions, admin user CRUD, platform KPIs & reports |
| **Organizations** | Schools, memberships, invites |
| **Courses & catalog** | Categories/tags, draft → publish → archive, pricing, FTS list filters |
| **Curriculum** | Modules → lessons → content blocks (`text` / `video` / `file` / `quiz_ref`), reorder |
| **Enrollment** | Self-enroll, capacity & waitlist, invite codes, paid-course gate |
| **Progress** | Lesson upserts, course completion %, student dashboard |
| **Assessments** | Quizzes (MCQ, short answer), assignments + rubric, gradebook |
| **Media** | Presigned S3/MinIO uploads, scan hook, CDN-friendly URLs |
| **Live learning** | Sessions, attendance, Jitsi/custom/Zoom/Meet join links, calendar feed |
| **Communication** | Announcements, discussion threads, DMs, in-app notifications |
| **Certificates** | PDF completion certs, public verify, transcripts |
| **Payments** | Checkout, orders, coupons, manual/Stripe/Chapa adapters, webhooks |
| **Analytics** | Admin overview, enrollment/revenue/user reports, per-course teacher analytics |
| **Discovery** | Unified search, recommendations, i18n (en/am), feature flags, parent–student links |
| **Ops** | Liveness/readiness probes, Prometheus metrics, security headers, rate limits, GitHub Actions CI |

Every JSON response uses the shared envelope (`success`, `data`, `error`, `meta`). See [docs/api/envelope.md](docs/api/envelope.md).

```json
{
  "success": true,
  "data": { },
  "error": null,
  "meta": { "request_id": "…" }
}
```

All responses include **`X-Request-ID`** (generated when the client omits it). Production logs are structured JSON; development uses console formatting.

### API reference

| Resource | Where to look |
|----------|---------------|
| **Interactive docs** | [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) |
| **OpenAPI files** | `docs/swagger/swagger.json` |
| **Human guides** | [docs/api/README.md](docs/api/README.md) — auth, courses, curriculum, enrollment, progress, assessments, media, live, communication, certificates, payments, analytics, discovery |
| **Versioning & deprecation** | [versioning.md](docs/api/versioning.md) · [deprecation.md](docs/api/deprecation.md) |
| **API changelog** | [docs/api/CHANGELOG.md](docs/api/CHANGELOG.md) |

**Ops endpoints (not under `/api/v1`):**

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` | Liveness + `APP_VERSION` |
| `GET` | `/ready` | Readiness (Postgres; Redis/S3 when configured) |
| `GET` | `/metrics` | Prometheus text scrape target |
| `GET` | `/swagger/index.html` | Swagger UI |

---

## Project structure

```
.
├── cmd/api/                    # Process entrypoint
├── internal/
│   ├── apperr/                 # Typed application errors
│   ├── auth/                   # JWT token manager
│   ├── config/                 # Env + optional .env loading
│   ├── database/               # Postgres pool
│   ├── domain/                 # Business entities
│   ├── handlers/               # HTTP handlers (thin)
│   ├── i18n/                   # Locale messages (en, am)
│   ├── live/                   # Video provider adapters
│   ├── logging/                # Zerolog setup
│   ├── middleware/             # CORS, auth, RBAC, security, rate limits, metrics
│   ├── notify/                 # Notification outbox
│   ├── payment/                # Payment provider adapters
│   ├── platform/
│   │   ├── metrics/            # Prometheus registry
│   │   └── ready/              # Dependency checks
│   ├── rbac/                   # Permissions
│   ├── repository/postgres/    # SQL implementations
│   ├── response/               # JSON envelope helpers
│   ├── server/                 # Route wiring
│   ├── service/                # Use-cases
│   └── storage/                # S3/MinIO client
├── migrations/                 # golang-migrate SQL (000001–000017)
├── scripts/
│   ├── promote_teacher.go      # TUID=<uuid> go run ./scripts/promote_teacher.go
│   └── promote_admin.go        # TUID=<uuid> go run ./scripts/promote_admin.go
├── docs/
│   ├── api/                    # Guides + API changelog
│   ├── swagger/                # Generated OpenAPI
│   ├── git/                    # Contributing, branching, releases
│   ├── ops/                    # Runbook
│   └── db/                     # Schema conventions
├── .github/workflows/ci.yml    # CI pipeline
├── Dockerfile                  # Multi-stage, Go 1.25
├── docker-compose.yml
├── .air.toml
├── .env.example
├── CHANGELOG.md
└── Makefile
```

### Layering rules

| Layer | May depend on | Must not depend on |
|-------|---------------|--------------------|
| `handlers` | `service`, `response`, `apperr` | `repository` implementations, SQL |
| `service` | `domain`, `repository` interfaces, `apperr` | Gin / HTTP |
| `repository` | `domain` | Gin / handlers |
| `domain` | stdlib only | Gin, SQL, services |

---

## Makefile commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `bin/` |
| `make run` | Build and run |
| `make test` | Run all tests |
| `make dev` | Hot-reload with Air, or `go run` |
| `make ci` | fmt-check + vet + test + build + docs-check |
| `make vet` | `go vet ./...` |
| `make fmt-check` | Fail if sources need `gofmt` |
| `make docs` | Regenerate OpenAPI into `docs/swagger/` |
| `make docs-check` | Fail if swagger output is stale |
| `make migrate-up` | Apply pending migrations |
| `make migrate-down` | Roll back one migration |
| `make migrate-version` | Show current migration version |
| `make migrate-create NAME=…` | Scaffold a new migration pair |
| `make up` | Docker Compose: API + infra |
| `make up-infra` | Postgres + Redis + MinIO only |
| `make down` | Stop Compose stack |
| `make logs` | Follow API container logs |
| `make tools` | Install Air + swag CLI |
| `make install` | Download Go modules |
| `make clean` | Remove build artifacts |

---

## Environment variables

Copy `.env.example` → `.env`. Key settings:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `ENV` | `development` | `development` or `production` |
| `APP_VERSION` | `0.1.0` | Reported in `/health` and welcome |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DATABASE_URL` | _(empty)_ | Postgres URL — **required in production** |
| `REDIS_URL` | _(empty)_ | Redis URL; skipped in `/ready` if empty |
| `JWT_SECRET` | _(empty)_ | **Required when `ENV=production`** |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated origins |
| `CORS_ALLOW_CREDENTIALS` | `false` | Must stay `false` when origin is `*` |
| `S3_*` | see `.env.example` | MinIO/S3 for media uploads |
| `LIVE_DEFAULT_PROVIDER` | `custom` | `jitsi`, `zoom`, `google_meet`, … |
| `PAYMENT_DEFAULT_PROVIDER` | `manual` | `stripe`, `chapa` |
| `FEATURE_FLAGS` | _(empty)_ | Comma-separated toggles; prefix `!` to disable |
| `RATE_LIMIT_GLOBAL_RPS` | `100` | Global limit (`0` = off); skips ops paths |
| `RATE_LIMIT_AUTH_RPS` | `2` | Auth endpoint limit |
| `METRICS_ENABLED` | `true` | Expose `GET /metrics` |
| `SECURITY_HSTS` | `false` | HSTS header (enable with HTTPS) |
| `TRUSTED_PROXIES` | _(empty)_ | LB CIDRs for correct client IP |

Full list with payment and compose port vars: **[.env.example](.env.example)**.

---

## Production & CI

**CI** (`.github/workflows/ci.yml`) on every push/PR: `gofmt`, `go vet`, `go test -race`, build, OpenAPI freshness, migration up/down smoke on Postgres 16.

**Deploy & day-2 ops:** [docs/ops/RUNBOOK.md](docs/ops/RUNBOOK.md) — probes, migrations, backups, secrets rotation, incident steps.

**Releases:** [docs/git/RELEASE.md](docs/git/RELEASE.md) — SemVer, changelog, tagging.

Local CI gate before opening a PR:

```bash
make ci
```

---

## Development notes

**Promote roles locally** (re-login after so JWT carries the new role):

```bash
TUID=<user-uuid> go run ./scripts/promote_teacher.go
TUID=<user-uuid> go run ./scripts/promote_admin.go
```

**Regenerate OpenAPI** after changing handler swag comments:

```bash
make docs
```

**New migration:**

```bash
make migrate-create NAME=add_foo
make migrate-up
```

---

## Contributing

See **[docs/git/](docs/git/README.md)**:

- [Contributing](docs/git/CONTRIBUTING.md)
- [Branching](docs/git/BRANCHING.md)
- [Commit convention](docs/git/COMMIT_CONVENTION.md)
- [Releases](docs/git/RELEASE.md)

Branch from `master` → Conventional Commits → PR (squash-merge). Project history: [CHANGELOG.md](CHANGELOG.md).

---

## License

MIT — see [LICENSE](LICENSE).
