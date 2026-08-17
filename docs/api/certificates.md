# Certificates & transcripts

Completion certificates with PDF download and public verification codes, plus exportable transcripts.

## Certificate issuance

| Actor | Rule |
|-------|------|
| Student | Course progress must be **100%** with `completed_at` set |
| Teacher / admin | May issue for any enrolled learner (`user_id` in body) |

One certificate per course per user. Revoked certificates fail verification and download.

## Verification

Public, no auth:

`GET /api/v1/certificates/verify/:code`

Returns `{ valid, learner_name, course_title, issued_at, ... }`.

## PDF

Generated on download from stored metadata (learner, course, code, date). Not streamed through upload APIs.

## Transcript

JSON summary of enrollments with progress, quiz scores, assignment grades, and certificate refs.

## Endpoints

| Method | Path | Access |
|--------|------|--------|
| `POST` | `/api/v1/courses/:id/certificate` | self (complete) or teacher |
| `GET` | `/api/v1/courses/:id/certificates` | teacher |
| `GET` | `/api/v1/courses/:id/transcript/:userId` | teacher |
| `GET` | `/api/v1/me/certificates` | owner |
| `GET` | `/api/v1/me/transcript` | owner |
| `GET` | `/api/v1/certificates/:id` | owner or teacher |
| `GET` | `/api/v1/certificates/:id/download` | owner or teacher |
| `POST` | `/api/v1/certificates/:id/revoke` | teacher |
| `GET` | `/api/v1/certificates/verify/:code` | public |
