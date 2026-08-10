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
7. (Later stages) trigger deploy from the tag

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
