# Asnakech API documentation

Interactive OpenAPI UI (Swagger) is served by the running API:

| URL | Description |
|-----|-------------|
| [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html) | Swagger UI |
| [http://localhost:8080/swagger/doc.json](http://localhost:8080/swagger/doc.json) | OpenAPI JSON |

Generated specs live in [`docs/swagger/`](../swagger/) (`swagger.json`, `swagger.yaml`, `docs.go`).

## Guides

| Doc | Topic |
|-----|-------|
| [envelope.md](./envelope.md) | Success/error JSON shape |
| [versioning.md](./versioning.md) | `/api/v1` policy |
| [auth.md](./auth.md) | Register, login, JWT, password reset |
| [rbac.md](./rbac.md) | Roles, permissions, admin APIs |
| [organizations.md](./organizations.md) | Schools, members, invites |
| [courses.md](./courses.md) | Catalog, categories, publish |
| [curriculum.md](./curriculum.md) | Modules, lessons, content blocks |
| [enrollment.md](./enrollment.md) | Enroll, capacity, invite codes, access |
| [progress.md](./progress.md) | Lesson/course progress, completion |
| [assessments.md](./assessments.md) | Quizzes, assignments, gradebook |
| [CHANGELOG.md](./CHANGELOG.md) | API-facing changes |

## Regenerating OpenAPI

After adding or changing swag annotations on handlers:

```bash
make docs
```

Then commit the updated files under `docs/swagger/`.

## Annotating new endpoints

1. Add swag comments on the handler (`@Summary`, `@Tags`, `@Router`, `@Success`, `@Failure`, …)
2. Prefer exported response models in `internal/handlers/swagger_models.go`
3. Use full paths in `@Router` (BasePath is `/`) — e.g. `/api/v1/courses`
4. Run `make docs`
5. Note client-visible changes in [CHANGELOG.md](./CHANGELOG.md)

Auth (Stage 6+): protect routes with `BearerAuth` and annotate:

```go
// @Security BearerAuth
```
