# Admin, analytics & reporting

Platform KPIs for administrators and course-level analytics for teachers. All metrics are computed from existing tables — no separate analytics warehouse in v1.

## Admin dashboard

`GET /api/v1/admin/overview` returns:

- User totals by role
- Course totals by status
- Enrollment totals by status
- Organizations and certificates
- Paid order count + revenue
- 7-day trends (enrollments, revenue, new users)

Requires `users:manage` (platform admin).

## Reports

Time-bounded JSON reports. Default range: **last 30 days** when `from`/`to` omitted.

| Endpoint | Groups by |
|----------|-----------|
| `GET /api/v1/admin/reports/enrollments` | day, course, status |
| `GET /api/v1/admin/reports/revenue` | day, course (paid orders) |
| `GET /api/v1/admin/reports/users` | day, role |

Query params:

| Param | Format |
|-------|--------|
| `from` | `YYYY-MM-DD` or RFC3339 |
| `to` | `YYYY-MM-DD` or RFC3339 |

## Course analytics (teacher)

`GET /api/v1/courses/:id/analytics` — enrollment breakdown, completion rate, average progress, revenue, certificates, published quizzes/assignments.

Requires course teacher or platform admin (`courses:write`).

## Bootstrap admin (local)

```bash
TUID=<user-uuid> go run ./scripts/promote_admin.go
```

Re-login so the JWT carries the `admin` role.

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `GET` | `/api/v1/admin/overview` | admin |
| `GET` | `/api/v1/admin/reports/enrollments` | admin |
| `GET` | `/api/v1/admin/reports/revenue` | admin |
| `GET` | `/api/v1/admin/reports/users` | admin |
| `GET` | `/api/v1/courses/:id/analytics` | teacher / admin |

Existing admin user CRUD remains under `/api/v1/admin/users` — see [rbac.md](./rbac.md).
