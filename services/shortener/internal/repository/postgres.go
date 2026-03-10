package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/zalhui/URLShortener/internal/database"
	"github.com/zalhui/URLShortener/internal/entity"
)

const dbTimeout = 5 * time.Second

type PostgresRepository struct {
	db *database.DB
}

func NewPostgresRepository(db *database.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(url *entity.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	const query = `
INSERT INTO short_urls (uuid, original_url, short_url, created_at)
VALUES ($1, $2, $3, $4)
`

	if _, err := r.db.Pool.Exec(ctx, query, url.UUID, url.OriginalURL, url.ShortURL, url.CreatedAt); err != nil {
		return fmt.Errorf("failed to save url: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByShortID(shortID string) (*entity.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	const query = `
SELECT uuid, original_url, short_url, created_at
FROM short_urls
WHERE uuid = $1
`

	url := &entity.URL{}
	err := r.db.Pool.QueryRow(ctx, query, shortID).Scan(
		&url.UUID,
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

func (r *PostgresRepository) GetByOriginalURL(originalURL string) (*entity.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	const query = `
SELECT uuid, original_url, short_url, created_at
FROM short_urls
WHERE original_url = $1
`

	url := &entity.URL{}
	err := r.db.Pool.QueryRow(ctx, query, originalURL).Scan(
		&url.UUID,
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

func (r *PostgresRepository) Close() error {
	return nil
}
