# Media & content delivery

Binary files never pass through the API. Clients request a **presigned PUT**, upload directly to S3/MinIO, then call **complete**.

## Flow

1. `POST /api/v1/media/uploads` → `{ upload_url, headers, asset }`
2. Client `PUT` bytes to `upload_url` with `Content-Type`
3. `POST /api/v1/media/:id/complete` (optional video `duration_seconds` / `width` / `height`)
4. Use `public_url` in courses, lessons, assignments, avatars

## Purposes & limits

| Purpose | Max size | Allowed types |
|---------|----------|---------------|
| `avatar` | 5 MiB | jpeg, png, webp, gif |
| `course_cover` | 10 MiB | `image/*` |
| `lesson_media` | 500 MiB | video/audio/image, pdf |
| `assignment_attachment` | 50 MiB | pdf, images, text, zip |
| `general` | 25 MiB | image/pdf/video/audio |

## Virus scan hook

On complete, `SkipScanner` marks `scan_status=skipped` (ready). Admins (or a future worker) can POST:

`POST /api/v1/media/:id/scan-result` `{ "scan_status": "clean"|"infected"|"skipped", "note": "..." }`

Infected assets are rejected and the object is deleted.

## CDN URLs

Set `S3_PUBLIC_BASE_URL` (e.g. `https://cdn.example.com/asnakech`) for CDN-friendly `public_url`. Otherwise path-style `{S3_ENDPOINT}/{bucket}/{key}`.

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST` | `/api/v1/media/uploads` | authenticated |
| `POST` | `/api/v1/media/:id/complete` | owner |
| `GET` | `/api/v1/media/:id` | owner / course teacher / admin |
| `DELETE` | `/api/v1/media/:id` | owner |
| `GET` | `/api/v1/me/media` | list mine |
| `POST` | `/api/v1/media/:id/scan-result` | admin |
| `GET` | `/api/v1/users/me/avatar/upload-intent` | real presign when S3 configured |

Requires `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`.
