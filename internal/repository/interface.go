package repository

import "github.com/zalhui/URLShortener/internal/entity"

type URLRepository interface {
	Save(url *entity.URL) error
	GetByShortID(shortID string) (*entity.URL, error)
	GetByOriginalURL(originalURL string) (*entity.URL, error)
	Close() error
}
