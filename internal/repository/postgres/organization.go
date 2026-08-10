package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	const q = `
		INSERT INTO organizations (name, slug, description, logo_url, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		org.Name,
		org.Slug,
		org.Description,
		org.LogoURL,
		org.CreatedBy,
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("organization slug already exists")
		}
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	const q = `
		SELECT id::text, name, slug, description, logo_url, created_by::text, created_at, updated_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL`

	org, err := scanOrg(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("organization not found")
		}
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return org, nil
}

func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	const q = `
		SELECT id::text, name, slug, description, logo_url, created_by::text, created_at, updated_at
		FROM organizations
		WHERE lower(slug) = lower($1) AND deleted_at IS NULL`

	org, err := scanOrg(r.pool.QueryRow(ctx, q, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("organization not found")
		}
		return nil, fmt.Errorf("get organization by slug: %w", err)
	}
	return org, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, id string, patch domain.OrganizationUpdate) (*domain.Organization, error) {
	current, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	name := current.Name
	desc := current.Description
	logo := current.LogoURL
	if patch.Name != nil {
		name = strings.TrimSpace(*patch.Name)
	}
	if patch.Description != nil {
		desc = strings.TrimSpace(*patch.Description)
	}
	if patch.LogoURL != nil {
		logo = strings.TrimSpace(*patch.LogoURL)
	}

	const q = `
		UPDATE organizations
		SET name = $2, description = $3, logo_url = $4
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, name, desc, logo)
	if err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("organization not found")
	}
	return r.GetByID(ctx, id)
}

func (r *OrganizationRepository) SoftDelete(ctx context.Context, id string, at time.Time) error {
	const q = `
		UPDATE organizations
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, at)
	if err != nil {
		return fmt.Errorf("delete organization: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("organization not found")
	}
	return nil
}

func (r *OrganizationRepository) ListForUser(ctx context.Context, userID string) ([]domain.Organization, error) {
	const q = `
		SELECT o.id::text, o.name, o.slug, o.description, o.logo_url, o.created_by::text, o.created_at, o.updated_at
		FROM organizations o
		JOIN organization_members m ON m.organization_id = o.id
		WHERE m.user_id = $1 AND o.deleted_at IS NULL
		ORDER BY o.name ASC`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Organization, 0)
	for rows.Next() {
		org, err := scanOrg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *org)
	}
	return out, rows.Err()
}

func scanOrg(row scannable) (*domain.Organization, error) {
	var org domain.Organization
	var createdBy *string
	if err := row.Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Description,
		&org.LogoURL,
		&createdBy,
		&org.CreatedAt,
		&org.UpdatedAt,
	); err != nil {
		return nil, err
	}
	org.CreatedBy = createdBy
	return &org, nil
}

type OrganizationMemberRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationMemberRepository(pool *pgxpool.Pool) *OrganizationMemberRepository {
	return &OrganizationMemberRepository{pool: pool}
}

func (r *OrganizationMemberRepository) Add(ctx context.Context, member *domain.OrganizationMember) error {
	const q = `
		INSERT INTO organization_members (organization_id, user_id, org_role)
		VALUES ($1, $2, $3)
		RETURNING id::text, joined_at, created_at`

	err := r.pool.QueryRow(ctx, q, member.OrganizationID, member.UserID, string(member.OrgRole)).
		Scan(&member.ID, &member.JoinedAt, &member.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("user is already a member")
		}
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

func (r *OrganizationMemberRepository) Get(ctx context.Context, orgID, userID string) (*domain.OrganizationMember, error) {
	const q = `
		SELECT m.id::text, m.organization_id::text, m.user_id::text, m.org_role, m.joined_at, m.created_at,
		       u.email, u.full_name
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1 AND m.user_id = $2 AND u.deleted_at IS NULL`

	member, err := scanMember(r.pool.QueryRow(ctx, q, orgID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.Forbidden("not a member of this organization")
		}
		return nil, fmt.Errorf("get member: %w", err)
	}
	return member, nil
}

func (r *OrganizationMemberRepository) ListByOrg(ctx context.Context, orgID string) ([]domain.OrganizationMember, error) {
	const q = `
		SELECT m.id::text, m.organization_id::text, m.user_id::text, m.org_role, m.joined_at, m.created_at,
		       u.email, u.full_name
		FROM organization_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1 AND u.deleted_at IS NULL
		ORDER BY m.joined_at ASC`

	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	out := make([]domain.OrganizationMember, 0)
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *OrganizationMemberRepository) UpdateRole(ctx context.Context, orgID, userID string, role domain.OrgRole) error {
	const q = `
		UPDATE organization_members
		SET org_role = $3
		WHERE organization_id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, q, orgID, userID, string(role))
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("member not found")
	}
	return nil
}

func (r *OrganizationMemberRepository) Remove(ctx context.Context, orgID, userID string) error {
	const q = `
		DELETE FROM organization_members
		WHERE organization_id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, q, orgID, userID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("member not found")
	}
	return nil
}

func (r *OrganizationMemberRepository) CountOwners(ctx context.Context, orgID string) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM organization_members
		WHERE organization_id = $1 AND org_role = 'owner'`

	var n int
	if err := r.pool.QueryRow(ctx, q, orgID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}
	return n, nil
}

func scanMember(row scannable) (*domain.OrganizationMember, error) {
	var m domain.OrganizationMember
	var role string
	if err := row.Scan(
		&m.ID,
		&m.OrganizationID,
		&m.UserID,
		&role,
		&m.JoinedAt,
		&m.CreatedAt,
		&m.UserEmail,
		&m.UserFullName,
	); err != nil {
		return nil, err
	}
	m.OrgRole = domain.OrgRole(role)
	return &m, nil
}

type OrganizationInviteRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationInviteRepository(pool *pgxpool.Pool) *OrganizationInviteRepository {
	return &OrganizationInviteRepository{pool: pool}
}

func (r *OrganizationInviteRepository) Create(ctx context.Context, invite *domain.OrganizationInvite) error {
	const q = `
		INSERT INTO organization_invites (organization_id, email, org_role, token_hash, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at`

	err := r.pool.QueryRow(ctx, q,
		invite.OrganizationID,
		strings.ToLower(strings.TrimSpace(invite.Email)),
		string(invite.OrgRole),
		invite.TokenHash,
		invite.InvitedBy,
		invite.ExpiresAt,
	).Scan(&invite.ID, &invite.CreatedAt)
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

func (r *OrganizationInviteRepository) GetByHash(ctx context.Context, hash string) (*domain.OrganizationInvite, error) {
	const q = `
		SELECT id::text, organization_id::text, email, org_role, token_hash, invited_by::text,
		       expires_at, accepted_at, revoked_at, created_at
		FROM organization_invites
		WHERE token_hash = $1`

	invite, err := scanInvite(r.pool.QueryRow(ctx, q, hash))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.Unauthorized("invalid invite token")
		}
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return invite, nil
}

func (r *OrganizationInviteRepository) GetByID(ctx context.Context, id string) (*domain.OrganizationInvite, error) {
	const q = `
		SELECT id::text, organization_id::text, email, org_role, token_hash, invited_by::text,
		       expires_at, accepted_at, revoked_at, created_at
		FROM organization_invites
		WHERE id = $1`

	invite, err := scanInvite(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("invite not found")
		}
		return nil, fmt.Errorf("get invite by id: %w", err)
	}
	return invite, nil
}

func (r *OrganizationInviteRepository) ListPendingByOrg(ctx context.Context, orgID string) ([]domain.OrganizationInvite, error) {
	const q = `
		SELECT id::text, organization_id::text, email, org_role, token_hash, invited_by::text,
		       expires_at, accepted_at, revoked_at, created_at
		FROM organization_invites
		WHERE organization_id = $1
		  AND accepted_at IS NULL
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	out := make([]domain.OrganizationInvite, 0)
	for rows.Next() {
		inv, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

func (r *OrganizationInviteRepository) MarkAccepted(ctx context.Context, id string, at time.Time) error {
	const q = `
		UPDATE organization_invites
		SET accepted_at = $2
		WHERE id = $1 AND accepted_at IS NULL AND revoked_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, at)
	if err != nil {
		return fmt.Errorf("accept invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.Conflict("invite already used or revoked")
	}
	return nil
}

func (r *OrganizationInviteRepository) Revoke(ctx context.Context, id string, at time.Time) error {
	const q = `
		UPDATE organization_invites
		SET revoked_at = $2
		WHERE id = $1 AND accepted_at IS NULL AND revoked_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, at)
	if err != nil {
		return fmt.Errorf("revoke invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("invite not found or already closed")
	}
	return nil
}

func scanInvite(row scannable) (*domain.OrganizationInvite, error) {
	var inv domain.OrganizationInvite
	var role string
	var invitedBy *string
	if err := row.Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&role,
		&inv.TokenHash,
		&invitedBy,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.RevokedAt,
		&inv.CreatedAt,
	); err != nil {
		return nil, err
	}
	inv.OrgRole = domain.OrgRole(role)
	inv.InvitedBy = invitedBy
	return &inv, nil
}
