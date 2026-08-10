# Asnakech School Servers

Backend API for the Asnakech online education platform. Built with Go and Gin as a **modular monolith** (domain → service → repository → handler).

> Status: early foundation. Stage 1 architecture is in place; education domains land in later stages.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

```bash
git clone https://github.com/kidusshoa/asnakech-servers.git
cd asnakech-servers
cp .env.example .env
make install
```

### Running

```bash
make dev
```

Server listens on `http://localhost:8080` by default.

## Project Structure

```
.
├── cmd/api/                 # Process entrypoint
├── internal/
│   ├── apperr/              # Typed application errors
│   ├── config/              # Environment-based configuration
│   ├── domain/              # Business entities (growing)
│   ├── handlers/            # HTTP handlers (thin)
│   ├── middleware/          # CORS, request ID, …
│   ├── repository/          # Persistence interfaces (growing)
│   ├── response/            # Standard JSON envelope
│   ├── server/              # HTTP server wiring & routes
│   └── service/             # Use-cases / workflows (growing)
├── docs/git/                # Contributing, branching, commits, releases
├── .github/                 # PR / issue templates, CODEOWNERS
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
| `make clean` | Remove build artifacts |
| `make deps` | Refresh dependencies |
| `make install` | Download modules |
| `make dev` | Run with `go run` |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness / version |
| `GET` | `/api/v1/` | Welcome stub |

Responses use a shared envelope:

```json
{
  "success": true,
  "data": { },
  "error": null,
  "meta": { }
}
```

Every response includes an `X-Request-ID` header (generated if the client did not send one).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `ENV` | `development` | `development` or `production` |
| `APP_VERSION` | `0.1.0` | Reported in health/welcome |
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated origins |
| `CORS_ALLOW_CREDENTIALS` | `false` | Must stay false when origin is `*` |

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
