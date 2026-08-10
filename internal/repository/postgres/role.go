// Package postgres implements repository interfaces against PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleRepository is a Postgres-backed RoleRepository.
type RoleRepository struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{pool: pool}
}

func (r *RoleRepository) List(ctx context.Context) ([]domain.Role, error) {
	const q = `
		SELECT id::text, code, name, description, created_at, updated_at
		FROM roles
		ORDER BY code ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	roles := make([]domain.Role, 0, 4)
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list roles iterate: %w", err)
	}
	return roles, nil
}

func (r *RoleRepository) GetByCode(ctx context.Context, code domain.RoleCode) (*domain.Role, error) {
	const q = `
		SELECT id::text, code, name, description, created_at, updated_at
		FROM roles
		WHERE code = $1`

	row := r.pool.QueryRow(ctx, q, string(code))
	role, err := scanRole(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("role not found")
		}
		return nil, fmt.Errorf("get role by code: %w", err)
	}
	return &role, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanRole(row scannable) (domain.Role, error) {
	var role domain.Role
	var code string
	if err := row.Scan(
		&role.ID,
		&code,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	); err != nil {
		return domain.Role{}, err
	}
	role.Code = domain.RoleCode(code)
	return role, nil
}
