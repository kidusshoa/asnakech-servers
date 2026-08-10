# Contributing to Asnakech Servers

Thanks for contributing. This repo is a Go modular-monolith API for the Asnakech education platform. Please read this guide before opening a PR.

## Quick start

1. Fork (or clone) the repository
2. Create a branch from `master` — see [BRANCHING.md](./BRANCHING.md)
3. Make changes; keep PRs focused on one concern
4. Commit with [Conventional Commits](./COMMIT_CONVENTION.md)
5. Push and open a Pull Request using the template
6. Address review feedback; we squash-merge into `master`

```bash
git clone https://github.com/kidusshoa/asnakech-servers.git
cd asnakech-servers
cp .env.example .env
make install
make test
make dev
```

## What we expect in a PR

- **Scope:** one feature, fix, or docs change — avoid drive-by refactors
- **Tests:** add or update tests for non-trivial logic (`apperr`, middleware, services, repos)
- **Build:** `make test` and `make build` pass locally
- **API shape:** new/changed endpoints use the shared `response` envelope and `apperr` codes
- **Docs:** update README or `docs/` when behavior or workflow changes
- **Secrets:** never commit `.env`, tokens, private keys, or credentials

## Layering (do not break)

| Layer | May depend on | Must not depend on |
|-------|---------------|--------------------|
| `handlers` | `service`, `response`, `apperr` | SQL / repo implementations |
| `service` | `domain`, repository interfaces, `apperr` | Gin / HTTP |
| `repository` | `domain` | Gin / handlers |
| `domain` | stdlib | Gin, SQL, services |

## Code style

- Follow standard Go formatting (`gofmt` / `go fmt ./...`)
- Prefer clear names over abbreviations
- Keep handlers thin: bind → call service → map response
- Prefer small, reviewable commits that still compile

## Review & merge

- Prefer squash-merge to keep `master` history linear
- At least one approving review when collaborators are active
- CI (when added in Stage 21) must be green before merge

## Releases

See [RELEASE.md](./RELEASE.md) for versioning, tags, and changelog updates.

## Questions

Open a GitHub Discussion or Issue if something in these docs is unclear.
