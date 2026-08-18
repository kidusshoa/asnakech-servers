# RBAC permission matrix

Roles are stored in `roles` and assigned on `users.role_id`. Access tokens embed `role` for middleware checks.

## Roles

| Code | Purpose |
|------|---------|
| `student` | Default registration role; learns |
| `teacher` | Authors and delivers courses |
| `parent` | Guardian linked to students (Stage 8+) |
| `admin` | Platform / school administration |

## Permissions

| Permission | student | teacher | parent | admin |
|------------|:-------:|:-------:|:------:|:-----:|
| `profile:read` | ✓ | ✓ | ✓ | ✓ |
| `profile:write` | ✓ | ✓ | ✓ | ✓ |
| `roles:read` | ✓ | ✓ | ✓ | ✓ |
| `users:read` | | | | ✓ |
| `users:manage` | | | | ✓ |
| `courses:read` | ✓ | ✓ | ✓ | ✓ |
| `courses:write` | | ✓ | | ✓ |
| `courses:manage` | | | | ✓ |
| `orgs:create` | ✓ | ✓ | ✓ | ✓ |
| `orgs:read` | ✓ | ✓ | ✓ | ✓ |

Org-scoped actions (invite, update settings) additionally require membership `owner`/`admin` — see [organizations.md](./organizations.md).

Source of truth: `internal/rbac/rbac.go`.

## Middleware

```go
middleware.BearerAuth(tokens)
middleware.RequireRoles(domain.RoleAdmin)
middleware.RequirePermission(rbac.PermUsersManage)
```

## Bootstrap an admin (local)

```sql
UPDATE users
SET role_id = (SELECT id FROM roles WHERE code = 'admin')
WHERE email = 'you@example.com';
```

Then log in again so the JWT picks up the new role.

## Profile & admin APIs

| Method | Path | Permission |
|--------|------|------------|
| `GET` | `/api/v1/users/me` | authenticated |
| `PATCH` | `/api/v1/users/me` | `profile:write` |
| `PUT` | `/api/v1/users/me/avatar` | `profile:write` |
| `GET` | `/api/v1/users/me/avatar/upload-intent` | `profile:write` (stub) |
| `GET` | `/api/v1/admin/users` | `users:manage` |
| `GET` | `/api/v1/admin/users/:id` | `users:manage` |
| `PATCH` | `/api/v1/admin/users/:id` | `users:manage` |
| `DELETE` | `/api/v1/admin/users/:id` | `users:manage` |

Analytics and reports (`users:manage`): see [analytics.md](./analytics.md).
