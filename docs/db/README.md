# Database conventions

PostgreSQL is the system of record for Asnakech Servers.

## Tooling

- Migrations live in [`/migrations`](../../migrations) as numbered SQL pairs (`*.up.sql` / `*.down.sql`)
- Applied with [golang-migrate](https://github.com/golang-migrate/migrate) via Makefile:

```bash
make up-infra          # start Postgres
make migrate-up        # apply all pending migrations
make migrate-down      # roll back one step
make migrate-version   # show current version
make migrate-create NAME=add_users
```

`DATABASE_URL` must be set (see `.env.example`).

## Schema rules

| Rule | Guidance |
|------|----------|
| Primary keys | `UUID` with `gen_random_uuid()` (`pgcrypto`) |
| Timestamps | `created_at` / `updated_at` as `TIMESTAMPTZ`, default `NOW()` |
| `updated_at` | Attach `set_updated_at` trigger on mutable tables |
| Soft deletes | Prefer `deleted_at TIMESTAMPTZ NULL` when history matters; omit for pure reference data (e.g. `roles`) |
| Naming | `snake_case` tables/columns; plural table names |
| Money | Later: integer minor units + currency code (never float) |
| FKs | Explicit `ON DELETE` behavior; prefer restrict for important refs |

## Layering

```
handlers → service → repository (interface) → repository/postgres
```

- Domain types live in `internal/domain`
- Interfaces in `internal/repository`
- SQL implementations in `internal/repository/postgres`
- Shared pool from `internal/database`

## Seeded roles

Migration `000002` seeds:

| code | name |
|------|------|
| `student` | Student |
| `teacher` | Teacher |
| `admin` | Admin |
| `parent` | Parent |

List via `GET /api/v1/roles` when `DATABASE_URL` is configured and migrations are applied.
