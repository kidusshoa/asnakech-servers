# Response envelope

Every JSON API response uses this shape:

```json
{
  "success": true,
  "data": {},
  "error": null,
  "meta": {}
}
```

| Field | When present | Meaning |
|-------|--------------|---------|
| `success` | always | `true` on 2xx business success; `false` on errors |
| `data` | success (and some error bodies with context) | Payload |
| `error` | failures | `{ "code": "...", "message": "..." }` |
| `meta` | optional | Pagination, `request_id`, etc. |

## Error codes

Stable machine-readable `error.code` values (see `internal/apperr`):

| Code | Typical HTTP |
|------|--------------|
| `validation_error` | 400 |
| `unauthorized` | 401 |
| `forbidden` | 403 |
| `not_found` | 404 |
| `conflict` | 409 |
| `internal_error` | 500 |
| `not_ready` | 503 |

## Request ID

- Send optional `X-Request-ID`
- If omitted, the server generates one
- Echoed on every response; included in `meta.request_id` on errors

## Pagination (upcoming)

List endpoints will use query params `page` / `per_page` and return:

```json
{
  "success": true,
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100
  }
}
```
