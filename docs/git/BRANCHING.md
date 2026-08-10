# Branching model

Asnakech Servers uses a simple **trunk-based** flow with short-lived topic branches.

## Default branch

- **`master`** — protected integration branch; always deployable / buildable
- Direct commits to `master` are discouraged; use Pull Requests

> Renaming to `main` later is fine; update this file and GitHub default-branch settings together.

## Branch naming

| Prefix | Use for | Example |
|--------|---------|---------|
| `feat/` | New capability | `feat/auth-jwt` |
| `fix/` | Bug fix | `fix/cors-credentials` |
| `docs/` | Documentation only | `docs/git-workflow` |
| `chore/` | Tooling, deps, cleanup | `chore/makefile-air` |
| `refactor/` | Internal restructuring, no behavior change | `refactor/server-wiring` |
| `test/` | Tests only | `test/middleware-cors` |
| `stage/` | Roadmap stage checkpoints (optional) | `stage/02-git-docs` |

Rules:

- Lowercase, kebab-case after the prefix
- No personal long-lived branches (`john/wip`) that accumulate unrelated work
- Delete the branch after the PR is merged

## Workflow

```text
master ──► feat/my-change ──► PR (squash) ──► master
```

1. `git checkout master && git pull`
2. `git checkout -b feat/short-description`
3. Commit with Conventional Commits
4. `git push -u origin HEAD`
5. Open PR → review → squash-merge
6. `git checkout master && git pull` and delete local branch

## Long-running work

If a stage spans multiple PRs:

- Land foundational PR first (e.g. interfaces / migrations)
- Follow with feature PRs that build on it
- Optional: open a tracking issue titled `Stage N: …` and link PRs to it

## Hotfixes

Use `fix/` from the latest `master` (or from a release tag if you are patching a release — see [RELEASE.md](./RELEASE.md)).
