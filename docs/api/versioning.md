# API versioning

## Current

- Versioned business routes live under **`/api/v1`**
- Operational probes stay at the root: `/health`, `/ready`
- Interactive docs: `/swagger/*`

## Rules

1. **Additive changes** in v1 are preferred (new fields, new endpoints)
2. **Breaking changes** require a new major path (`/api/v2`) or a clearly announced deprecation window
3. Breaking = removed fields, renamed fields, changed meaning, stricter auth, or incompatible status codes for the same operation
4. Document every client-visible change in [`CHANGELOG.md`](./CHANGELOG.md)

## Deprecation (future)

When retiring a v1 endpoint:

1. Mark deprecated in OpenAPI (`@Deprecated`)
2. Note removal date in the API changelog
3. Keep behavior until the announced date
4. Remove only in a major bump

## Auth header (from Stage 6)

```http
Authorization: Bearer <access_token>
```

Declared in OpenAPI as `BearerAuth`.
