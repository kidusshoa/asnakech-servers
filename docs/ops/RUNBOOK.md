# Operations runbook

Day-2 operations for Asnakech Servers: deploy, migrations, backups, incidents, and observability.

## Health probes

| Endpoint | Purpose | Expected |
|----------|---------|----------|
| `GET /health` | Liveness — process is up | `200`, `status: ok` |
| `GET /ready` | Readiness — dependencies reachable | `200` when Postgres (and configured Redis/S3) OK; `503` otherwise |
| `GET /metrics` | Prometheus scrape target | `200` text exposition (restrict by network/firewall) |

Kubernetes example:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  periodSeconds: 10
  failureThreshold: 3
```

## Deploy (container)

1. Build image from tagged release (`vX.Y.Z`) — Dockerfile uses Go **1.25** to match `go.mod`.
2. Run migrations **before** or as a one-shot init container (recommended: init job, then roll API).
3. Set required production env:
   - `ENV=production`
   - `JWT_SECRET` (strong random, rotated via secret manager)
   - `DATABASE_URL`
   - `APP_VERSION` matching the git tag
4. Optional hardening:
   - `SECURITY_HSTS=true` when TLS terminates at the edge
   - `TRUSTED_PROXIES=10.0.0.0/8` (adjust for your LB CIDRs)
   - `METRICS_ENABLED=true` but scrape only from internal network
   - Tune `RATE_LIMIT_GLOBAL_RPS` / `RATE_LIMIT_AUTH_RPS`

```bash
docker build -t asnakech-api:v0.2.0 .
docker run --rm -e DATABASE_URL=... -e JWT_SECRET=... -p 8080:8080 asnakech-api:v0.2.0
```

## Database migrations

Apply on every release that ships SQL under `migrations/`:

```bash
export DATABASE_URL='postgres://user:pass@host:5432/asnakech?sslmode=require'
make migrate-up
make migrate-version
```

Rollback one step (use with care in production):

```bash
make migrate-down
```

If a migration fails mid-deploy, fix forward with a new migration rather than editing applied files. Use `make migrate-force VERSION=N` only when the migrate version table is out of sync after manual recovery.

## Backups

**Postgres**

- Daily logical dumps (`pg_dump -Fc`) retained 30 days minimum.
- Test restore quarterly to a staging cluster.
- Point-in-time recovery (WAL archiving) recommended for production.

**Object storage (S3/MinIO)**

- Enable bucket versioning and lifecycle rules.
- Media assets are not in Postgres — backup the bucket independently.

**Redis** (if used for sessions/cache later)

- Treat as ephemeral unless persistence is explicitly enabled.

## Secrets rotation

| Secret | Rotation |
|--------|----------|
| `JWT_SECRET` | Invalidates all access tokens — plan maintenance window; users re-login |
| `STRIPE_*` / `CHAPA_*` | Rotate in provider dashboard; update env; replay webhooks if needed |
| DB credentials | Rotate via managed DB; update `DATABASE_URL`; rolling restart API |
| `PAYMENT_WEBHOOK_SECRET` | Update provider + env; old signatures rejected immediately |

Never commit `.env` or real keys. Use a secret manager (AWS SM, GCP SM, Vault, K8s secrets).

## Observability

- **Logs:** JSON in production (`LOG_LEVEL=info`). Correlate via `X-Request-ID`.
- **Metrics:** scrape `GET /metrics` (counters + latency histograms by route template).
- **Tracing:** not wired yet — add OpenTelemetry exporter when APM is chosen.

Suggested alerts:

- `/ready` failing > 2 min
- 5xx rate > 1% over 5 min
- p95 latency > 2s on `/api/v1/*`
- auth 429 spike (credential stuffing)

## Incident response (short)

1. Confirm scope: `/ready`, error logs, recent deploy/migration.
2. Roll back API image if deploy-related; do **not** run `migrate-down` unless the migration itself is broken and you have a tested down script.
3. Scale horizontally if CPU-bound; check Postgres connections and slow queries.
4. Communicate via status page; post-mortem for SEV-1/2.

## CI/CD sketch

GitHub Actions (`.github/workflows/ci.yml`) runs on PR/push:

- `gofmt`, `go vet`, `go test -race`, build
- OpenAPI freshness (`make docs-check`)
- Postgres service: migrate up → down all → up

Production CD (manual for now):

1. Merge to `master`
2. Tag release per [RELEASE.md](../git/RELEASE.md)
3. Build & push container
4. Run migrations job
5. Rolling deploy; watch `/ready` and `/metrics`

## Local troubleshooting

```bash
make up-infra && make migrate-up && make dev
curl -s localhost:8080/health | jq
curl -s localhost:8080/ready | jq
curl -s localhost:8080/metrics | head
```

Port conflicts: adjust `POSTGRES_HOST_PORT`, `REDIS_HOST_PORT`, `MINIO_*` in `.env` and keep URLs aligned.
