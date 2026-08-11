package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	const q = `
		SELECT id::text, name, slug, description, created_at, updated_at
		FROM categories
		ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Category, 0)
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	const q = `
		SELECT id::text, name, slug, description, created_at, updated_at
		FROM categories WHERE id = $1`

	var c domain.Category
	err := r.pool.QueryRow(ctx, q, id).Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("category not found")
		}
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	const q = `
		SELECT id::text, name, slug, description, created_at, updated_at
		FROM categories WHERE lower(slug) = lower($1)`

	var c domain.Category
	err := r.pool.QueryRow(ctx, q, slug).Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("category not found")
		}
		return nil, fmt.Errorf("get category by slug: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepository) Create(ctx context.Context, cat *domain.Category) error {
	const q = `
		INSERT INTO categories (name, slug, description)
		VALUES ($1, $2, $3)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q, cat.Name, cat.Slug, cat.Description).
		Scan(&cat.ID, &cat.CreatedAt, &cat.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("category slug already exists")
		}
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

type TagRepository struct {
	pool *pgxpool.Pool
}

func NewTagRepository(pool *pgxpool.Pool) *TagRepository {
	return &TagRepository{pool: pool}
}

func (r *TagRepository) GetOrCreateByNames(ctx context.Context, names []string) ([]domain.Tag, error) {
	out := make([]domain.Tag, 0, len(names))
	seen := map[string]struct{}{}
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}

		const q = `
			INSERT INTO tags (name, slug)
			VALUES ($1, $2)
			ON CONFLICT ((lower(slug))) DO UPDATE SET name = EXCLUDED.name
			RETURNING id::text, name, slug, created_at`

		// Unique index is on lower(slug) but ON CONFLICT needs constraint name or columns matching unique index.
		// Postgres ON CONFLICT for expression indexes: ON CONFLICT ((lower(slug))) works on PG15+.
		var tag domain.Tag
		err := r.pool.QueryRow(ctx, q, name, slug).Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.CreatedAt)
		if err != nil {
			// Fallback path if expression conflict target unsupported: select then insert.
			const sel = `SELECT id::text, name, slug, created_at FROM tags WHERE lower(slug) = lower($1)`
			err2 := r.pool.QueryRow(ctx, sel, slug).Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.CreatedAt)
			if err2 != nil {
				const ins = `
					INSERT INTO tags (name, slug) VALUES ($1, $2)
					RETURNING id::text, name, slug, created_at`
				if err3 := r.pool.QueryRow(ctx, ins, name, slug).Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.CreatedAt); err3 != nil {
					var pgErr *pgconn.PgError
					if errors.As(err3, &pgErr) && pgErr.Code == "23505" {
						if err4 := r.pool.QueryRow(ctx, sel, slug).Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.CreatedAt); err4 != nil {
							return nil, fmt.Errorf("get or create tag: %w", err4)
						}
					} else {
						return nil, fmt.Errorf("get or create tag: %w", err3)
					}
				}
			}
		}
		out = append(out, tag)
	}
	return out, nil
}

func (r *TagRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.Tag, error) {
	const q = `
		SELECT t.id::text, t.name, t.slug, t.created_at
		FROM tags t
		JOIN course_tags ct ON ct.tag_id = t.id
		WHERE ct.course_id = $1
		ORDER BY t.name ASC`

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list course tags: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Tag, 0)
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TagRepository) ReplaceCourseTags(ctx context.Context, courseID string, tagIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tag replace: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM course_tags WHERE course_id = $1`, courseID); err != nil {
		return fmt.Errorf("clear course tags: %w", err)
	}
	for _, id := range tagIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO course_tags (course_id, tag_id) VALUES ($1, $2)`, courseID, id); err != nil {
			return fmt.Errorf("insert course tag: %w", err)
		}
	}
	return tx.Commit(ctx)
}
