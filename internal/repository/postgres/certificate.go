package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CertificateRepository struct {
	pool *pgxpool.Pool
}

func NewCertificateRepository(pool *pgxpool.Pool) *CertificateRepository {
	return &CertificateRepository{pool: pool}
}

const certificateSelect = `
	c.id::text, c.course_id::text, c.user_id::text, c.verification_code,
	c.learner_name, c.course_title, c.storage_key, c.public_url,
	c.issued_at, c.revoked_at, c.created_at, c.updated_at,
	co.slug, u.email`

func (r *CertificateRepository) Create(ctx context.Context, c *domain.Certificate) error {
	const sql = `
		INSERT INTO certificates (
			course_id, user_id, verification_code, learner_name, course_title,
			storage_key, public_url, issued_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id::text, created_at, updated_at`
	err := r.pool.QueryRow(ctx, sql,
		c.CourseID, c.UserID, c.VerificationCode, c.LearnerName, c.CourseTitle,
		c.StorageKey, c.PublicURL, c.IssuedAt,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.Conflict("certificate already issued for this course")
		}
		return fmt.Errorf("create certificate: %w", err)
	}
	return nil
}

func (r *CertificateRepository) GetByID(ctx context.Context, id string) (*domain.Certificate, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.id = $1`, certificateSelect)
	cert, err := scanCertificate(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("certificate not found")
		}
		return nil, fmt.Errorf("get certificate: %w", err)
	}
	return cert, nil
}

func (r *CertificateRepository) GetByVerificationCode(ctx context.Context, code string) (*domain.Certificate, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.verification_code = $1`, certificateSelect)
	cert, err := scanCertificate(r.pool.QueryRow(ctx, q, code))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("certificate not found")
		}
		return nil, fmt.Errorf("verify certificate: %w", err)
	}
	return cert, nil
}

func (r *CertificateRepository) GetByCourseUser(ctx context.Context, courseID, userID string) (*domain.Certificate, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.course_id = $1 AND c.user_id = $2`, certificateSelect)
	cert, err := scanCertificate(r.pool.QueryRow(ctx, q, courseID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("certificate not found")
		}
		return nil, fmt.Errorf("get certificate: %w", err)
	}
	return cert, nil
}

func (r *CertificateRepository) ListByUser(ctx context.Context, userID string) ([]domain.Certificate, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.user_id = $1
		ORDER BY c.issued_at DESC`, certificateSelect)

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Certificate, 0)
	for rows.Next() {
		c, err := scanCertificate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *CertificateRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.Certificate, error) {
	q := fmt.Sprintf(`
		SELECT %s FROM certificates c
		JOIN courses co ON co.id = c.course_id
		JOIN users u ON u.id = c.user_id
		WHERE c.course_id = $1
		ORDER BY c.issued_at DESC`, certificateSelect)

	rows, err := r.pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, fmt.Errorf("list course certificates: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Certificate, 0)
	for rows.Next() {
		c, err := scanCertificate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *CertificateRepository) Revoke(ctx context.Context, id string) (*domain.Certificate, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE certificates SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL`, id)
	if err != nil {
		return nil, fmt.Errorf("revoke certificate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.NotFound("certificate not found or already revoked")
	}
	return r.GetByID(ctx, id)
}

func (r *CertificateRepository) UpdateStorage(ctx context.Context, id, storageKey, publicURL string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE certificates SET storage_key = $2, public_url = $3 WHERE id = $1`,
		id, storageKey, publicURL)
	if err != nil {
		return fmt.Errorf("update certificate storage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("certificate not found")
	}
	return nil
}

func scanCertificate(row pgx.Row) (*domain.Certificate, error) {
	var c domain.Certificate
	err := row.Scan(
		&c.ID, &c.CourseID, &c.UserID, &c.VerificationCode,
		&c.LearnerName, &c.CourseTitle, &c.StorageKey, &c.PublicURL,
		&c.IssuedAt, &c.RevokedAt, &c.CreatedAt, &c.UpdatedAt,
		&c.CourseSlug, &c.UserEmail,
	)
	return &c, err
}
