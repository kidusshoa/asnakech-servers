# Authentication

JWT Bearer auth is available under `/api/v1/auth/*`.

## Endpoints

| Method | Path | Auth |
|--------|------|------|
| `POST` | `/api/v1/auth/register` | public (rate-limited) |
| `POST` | `/api/v1/auth/login` | public (rate-limited) |
| `POST` | `/api/v1/auth/refresh` | public (rate-limited) |
| `POST` | `/api/v1/auth/logout` | refresh token body |
| `GET` | `/api/v1/auth/me` | `Authorization: Bearer <access>` |
| `POST` | `/api/v1/auth/forgot-password` | public (rate-limited) |
| `POST` | `/api/v1/auth/reset-password` | reset token |
| `POST` | `/api/v1/auth/verify-email` | verification token |

## Tokens

- **Access token** — short-lived JWT (`JWT_ACCESS_TTL`, default 15m). Send as `Authorization: Bearer …`
- **Refresh token** — opaque random string, stored hashed in Postgres (`JWT_REFRESH_TTL`, default 168h). Rotated on refresh.

## Development helpers

When `ENV != production`:

- Register responses may include `verification_token`
- Forgot-password responses may include `reset_token` when the email exists

Email delivery is not wired yet (notifications stage). Use the returned tokens locally.

## Defaults

New accounts receive the `student` role.
