package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zalhui/URLShortener/internal/database"
	"github.com/zalhui/URLShortener/internal/entity"
)

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(ctx context.Context, url *entity.URL) error {
	const query = `
INSERT INTO short_urls (uuid, user_id, original_url, short_url, created_at)
VALUES ($1, $2, $3, $4, $5)
`

	if _, err := r.db.Pool.Exec(ctx, query, url.UUID, url.UserID, url.OriginalURL, url.ShortURL, url.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "short_urls_user_id_original_url_key":
				return ErrDuplicateOriginalURL
			case "short_urls_pkey", "short_urls_short_url_key":
				return ErrDuplicateShortID
			}
		}

		return fmt.Errorf("failed to save url: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByShortID(ctx context.Context, shortID string) (*entity.URL, error) {
	const query = `
SELECT uuid, user_id, original_url, short_url, created_at
FROM short_urls
WHERE uuid = $1
`

	url := &entity.URL{}
	err := r.db.Pool.QueryRow(ctx, query, shortID).Scan(
		&url.UUID,
		&url.UserID,
		&url.OriginalURL,
		&url.ShortURL,
		&url.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get url by short id: %w", err)
	}

	return url, nil
}

func (r *PostgresRepository) GetByOriginalURL(ctx context.Context, userID, originalURL string) (*entity.URL, error) {
	const query = `
SELECT uuid, user_id, original_url, short_url, created_at
FROM short_urls
WHERE user_id = $1 AND original_url = $2
`

	url := &entity.URL{}
	err := r.db.Pool.QueryRow(ctx, query, userID, originalURL).Scan(
		&url.UUID,
		&url.UserID,
		&url.OriginalURL,
		&url.ShortURL,
		&url.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get url by original url: %w", err)
	}

	return url, nil
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID string) ([]*entity.URL, error) {
	const query = `
SELECT uuid, user_id, original_url, short_url, created_at
FROM short_urls
WHERE user_id = $1
ORDER BY created_at DESC
`

	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list urls by user: %w", err)
	}
	defer rows.Close()

	urls := make([]*entity.URL, 0)
	for rows.Next() {
		url := &entity.URL{}
		if err := rows.Scan(&url.UUID, &url.UserID, &url.OriginalURL, &url.ShortURL, &url.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan url: %w", err)
		}

		urls = append(urls, url)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate urls: %w", err)
	}

	return urls, nil
}

func (r *PostgresRepository) DeleteByShortID(ctx context.Context, userID, shortID string) error {
	const query = `
DELETE FROM short_urls
WHERE user_id = $1 AND uuid = $2
`

	if _, err := r.db.Pool.Exec(ctx, query, userID, shortID); err != nil {
		return fmt.Errorf("failed to delete url: %w", err)
	}

	return nil
}

func (r *PostgresRepository) Close() error {
	return nil
}
