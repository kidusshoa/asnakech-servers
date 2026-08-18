# API deprecation policy

How we evolve the Asnakech HTTP API without breaking clients unexpectedly.

## Versioning model

- Current stable surface: **`/api/v1`**
- Breaking changes require a new major path (`/api/v2`) or explicit client migration windows documented here and in [CHANGELOG.md](./CHANGELOG.md).
- Non-breaking additions (new optional fields, new endpoints, new query params) ship in v1 without a version bump.

See also [versioning.md](./versioning.md).

## What counts as breaking

| Breaking | Non-breaking |
|----------|--------------|
| Removing or renaming fields clients rely on | Adding optional JSON fields |
| Changing field types or semantics | Adding endpoints |
| Removing endpoints | Adding enum values (when clients ignore unknown) |
| Tightening validation on existing inputs | Stricter validation on **new** inputs only |
| Changing auth requirements for existing routes | New optional headers |

## Deprecation workflow

1. **Announce** in [CHANGELOG.md](./CHANGELOG.md) under `Deprecated` with target removal version/date.
2. **Document** the replacement (endpoint, field, or behavior).
3. **Observe** usage via logs/metrics if possible.
4. **Sunset** after at least **90 days** for public endpoints (longer for mobile clients with slow upgrade cycles).
5. **Remove** in a **minor** release before v1.0 lock, or a **major** release after 1.0.

## Response headers (future)

When an endpoint or field is deprecated, responses may include:

```http
Deprecation: true
Sunset: Sat, 01 Jan 2027 00:00:00 GMT
Link: </api/v1/new-path>; rel="successor-version"
```

Clients should log these headers and migrate before the sunset date.

## v2 planning notes

A future `/api/v2` should address:

- Consistent pagination cursor (`meta.next_cursor`) on all list endpoints
- Unified error detail shape for validation (`error.details[]`)
- Webhook event schema registry
- Optional OpenAPI-driven client SDK generation

v1 remains supported until v2 reaches feature parity and clients complete migration. v1 will receive security fixes only during the overlap period.

## Client guidance

- Pin integrations to documented paths under `/api/v1`.
- Treat unknown JSON fields as forward-compatible.
- Send `X-Request-ID` on mutating calls for support correlation.
- Subscribe to [API CHANGELOG](./CHANGELOG.md) and GitHub releases.
