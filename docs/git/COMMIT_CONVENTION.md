# Commit convention

We use [Conventional Commits](https://www.conventionalcommits.org/). This keeps history readable and feeds the changelog / release process.

## Format

```text
<type>(optional-scope): <short summary>

[optional body]

[optional footer(s)]
```

- Summary in **imperative mood**, lowercase, no trailing period
- Keep the first line ≤ ~72 characters
- Body explains *why*, not a file dump

## Types

| Type | When |
|------|------|
| `feat` | New user-facing capability or API endpoint |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting; no code meaning change |
| `refactor` | Restructure without behavior change |
| `perf` | Performance improvement |
| `test` | Adding or fixing tests |
| `chore` | Build, tooling, deps, CI config |
| `ci` | CI-specific changes |
| `build` | Build system or external dependencies |
| `revert` | Revert a previous commit |

## Scopes (optional, preferred)

Use a short area name when helpful:

`auth`, `config`, `cors`, `health`, `middleware`, `server`, `api`, `db`, `docs`, `git`

Examples:

```text
feat(auth): add JWT login and refresh endpoints
fix(cors): stop pairing wildcard origin with credentials
docs(git): add branching and release guides
chore(config): add APP_VERSION and CORS env vars
test(middleware): cover request-id generation
```

## Breaking changes

Mark breaking API or behavior changes in one of two ways:

1. `!` after type/scope:

```text
feat(api)!: rename enrollment status enum values
```

2. Or a footer:

```text
BREAKING CHANGE: /api/v1/courses now requires Authorization
```

Breaking changes bump the **major** version — see [RELEASE.md](./RELEASE.md).

## What not to commit

- Secrets (`.env`, tokens, keys)
- Generated binaries under `bin/`
- Large unrelated reformatting mixed with features

## Amending

Only amend if the commit has **not** been pushed, or you are coordinating a force-with-lease on your own topic branch. Never rewrite `master` history.
