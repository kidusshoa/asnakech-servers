# Releases

How we version, tag, and publish Asnakech Servers.

## Versioning

We follow [Semantic Versioning](https://semver.org/):

| Bump | When | Conventional Commits signal |
|------|------|-----------------------------|
| **MAJOR** `X.0.0` | Breaking API / behavior | `BREAKING CHANGE` or `type!:` |
| **MINOR** `0.X.0` | Backwards-compatible features | `feat` |
| **PATCH** `0.0.X` | Backwards-compatible fixes | `fix`, `perf` (non-breaking) |

Pre-1.0 (`0.y.z`): the API may still evolve; prefer clear changelog notes for any client-visible change.

Current app version is also exposed via `APP_VERSION` / health endpoint — keep it in sync with the git tag when cutting a release.

## Release checklist

1. Ensure `master` is green (`make test`, `make build`)
2. Update [CHANGELOG.md](../../CHANGELOG.md):
   - Move items from `[Unreleased]` into a new version section
   - Date the section (`## [0.2.0] - YYYY-MM-DD`)
3. Bump default `APP_VERSION` in `.env.example` (and docs if needed)
4. Commit: `chore(release): prepare v0.2.0`
5. Tag annotated release:

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin master
git push origin v0.2.0
```

6. Create a GitHub Release from the tag; paste the changelog section as the body
7. Trigger deploy from the tag (see [CI/CD](#cicd) below)

## CI/CD

**Continuous integration** runs on every push/PR to `master` via [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml):

| Step | Command / action |
|------|------------------|
| Format | `gofmt` — no diffs |
| Vet | `go vet ./...` |
| Test | `go test -race ./...` |
| Build | `go build ./cmd/api` |
| OpenAPI | `make docs-check` |
| Migrations | up → down all → up against Postgres 16 |

Local equivalent: `make ci` (requires Postgres for `docs-check` only if swagger changed; migration smoke is CI-only unless you run migrate manually).

**Continuous delivery (sketch)**

1. Tag `vX.Y.Z` on `master` after CI is green
2. Build and push container (`Dockerfile`, Go 1.25)
3. Run `make migrate-up` as a release job
4. Rolling deploy; watch `/ready` and scrape `/metrics`

Operational details: [docs/ops/RUNBOOK.md](../ops/RUNBOOK.md).

**API deprecation:** [docs/api/deprecation.md](../api/deprecation.md) — sunset windows before removing v1 endpoints.

## Changelog rules

- Keep [CHANGELOG.md](../../CHANGELOG.md) in [Keep a Changelog](https://keepachangelog.com/) format
- Group entries: Added, Changed, Fixed, Deprecated, Removed, Security
- Prefer user/API impact over internal file lists
- Link PRs/issues when available (`(#12)`)

## Hotfix releases

1. Branch `fix/...` from the release tag (or from `master` if still linear)
2. Fix + test
3. PR → merge
4. Cut a **patch** release (`v0.2.1`) with changelog entry under Fixed

## Stage milestones (optional)

Roadmap stages may be marked with lightweight tags for checkpoints:

```bash
git tag stage-02-git-docs
git push origin stage-02-git-docs
```

These are not SemVer releases; they are progress markers only.
