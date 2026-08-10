# Organizations (schools)

Light multi-tenant tenancy: an **organization** is a school or workspace. Courses (later stages) will hang off orgs.

## Org roles

| Role | Capabilities |
|------|----------------|
| `owner` | Full control, delete org, manage members/invites |
| `admin` | Update org, manage members/invites |
| `member` | Read org and member list |

Platform `admin` users can manage any organization without membership.

## Endpoints

| Method | Path | Notes |
|--------|------|-------|
| `POST` | `/api/v1/organizations` | Create; caller becomes `owner` |
| `GET` | `/api/v1/organizations` | List orgs you belong to |
| `GET` | `/api/v1/organizations/:id` | Member or platform admin |
| `PATCH` | `/api/v1/organizations/:id` | Owner/admin |
| `DELETE` | `/api/v1/organizations/:id` | Owner (or platform admin) |
| `GET` | `/api/v1/organizations/:id/members` | Members |
| `PATCH` | `/api/v1/organizations/:id/members/:userId` | Change org role |
| `DELETE` | `/api/v1/organizations/:id/members/:userId` | Remove member |
| `POST` | `/api/v1/organizations/:id/invites` | Invite by email |
| `GET` | `/api/v1/organizations/:id/invites` | Pending invites |
| `DELETE` | `/api/v1/organizations/:id/invites/:inviteId` | Revoke |
| `POST` | `/api/v1/organizations/invites/accept` | Accept with token |

Invites expire in 7 days. Accepting requires the authenticated user's email to match the invite. In development, create-invite may return `token` for local testing (email delivery comes later).

## Platform permissions

All authenticated roles have `orgs:create` and `orgs:read`. Fine-grained org actions are enforced by **membership org_role** inside the service layer.
