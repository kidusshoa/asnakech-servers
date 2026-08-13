package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepository struct {
	pool *pgxpool.Pool
}

func NewMediaRepository(pool *pgxpool.Pool) *MediaRepository {
	return &MediaRepository{pool: pool}
}

func (r *MediaRepository) Create(ctx context.Context, a *domain.MediaAsset) error {
	const q = `
		INSERT INTO media_assets (
			owner_id, course_id, purpose, status, content_type, original_filename,
			size_bytes, storage_key, public_url, duration_seconds, width, height,
			scan_status, scan_note
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id::text, created_at, updated_at`

	err := r.pool.QueryRow(ctx, q,
		a.OwnerID, a.CourseID, string(a.Purpose), string(a.Status), a.ContentType, a.OriginalFilename,
		a.SizeBytes, a.StorageKey, a.PublicURL, a.DurationSeconds, a.Width, a.Height,
		string(a.ScanStatus), a.ScanNote,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("media asset already exists")
		}
		return fmt.Errorf("create media asset: %w", err)
	}
	return nil
}

func (r *MediaRepository) GetByID(ctx context.Context, id string) (*domain.MediaAsset, error) {
	const q = `
		SELECT id::text, owner_id::text, course_id::text, purpose, status, content_type,
		       original_filename, size_bytes, storage_key, public_url, duration_seconds,
		       width, height, scan_status, scan_note, created_at, updated_at, deleted_at
		FROM media_assets
		WHERE id = $1 AND deleted_at IS NULL`

	a, err := scanMedia(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("media asset not found")
		}
		return nil, fmt.Errorf("get media asset: %w", err)
	}
	return a, nil
}

func (r *MediaRepository) Update(ctx context.Context, a *domain.MediaAsset) (*domain.MediaAsset, error) {
	const q = `
		UPDATE media_assets SET
			status = $2, content_type = $3, original_filename = $4, size_bytes = $5,
			public_url = $6, duration_seconds = $7, width = $8, height = $9,
			scan_status = $10, scan_note = $11
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q,
		a.ID, string(a.Status), a.ContentType, a.OriginalFilename, a.SizeBytes,
		a.PublicURL, a.DurationSeconds, a.Width, a.Height,
		string(a.ScanStatus), a.ScanNote,
	)
	if err != nil {
		return nil, fmt.Errorf("update media asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("media asset not found")
	}
	return r.GetByID(ctx, a.ID)
}

func (r *MediaRepository) ListByOwner(ctx context.Context, ownerID string, page, perPage int) ([]domain.MediaAsset, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM media_assets WHERE owner_id = $1 AND deleted_at IS NULL`, ownerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count media assets: %w", err)
	}

	const q = `
		SELECT id::text, owner_id::text, course_id::text, purpose, status, content_type,
		       original_filename, size_bytes, storage_key, public_url, duration_seconds,
		       width, height, scan_status, scan_note, created_at, updated_at, deleted_at
		FROM media_assets
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, q, ownerID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list media assets: %w", err)
	}
	defer rows.Close()

	out := make([]domain.MediaAsset, 0, perPage)
	for rows.Next() {
		a, err := scanMedia(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *a)
	}
	return out, total, rows.Err()
}

func (r *MediaRepository) SoftDelete(ctx context.Context, id string) error {
	const q = `
		UPDATE media_assets
		SET deleted_at = $2, status = 'deleted'
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("delete media asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("media asset not found")
	}
	return nil
}

func scanMedia(row scannable) (*domain.MediaAsset, error) {
	var a domain.MediaAsset
	var purpose, status, scanStatus string
	var courseID *string
	if err := row.Scan(
		&a.ID, &a.OwnerID, &courseID, &purpose, &status, &a.ContentType,
		&a.OriginalFilename, &a.SizeBytes, &a.StorageKey, &a.PublicURL, &a.DurationSeconds,
		&a.Width, &a.Height, &scanStatus, &a.ScanNote, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
	); err != nil {
		return nil, err
	}
	a.CourseID = courseID
	a.Purpose = domain.MediaPurpose(purpose)
	a.Status = domain.MediaStatus(status)
	a.ScanStatus = domain.MediaScanStatus(scanStatus)
	return &a, nil
}
