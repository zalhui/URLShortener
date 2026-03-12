package repository

import (
	"context"
	"errors"

	"github.com/zalhui/URLShortener/internal/entity"
)

var (
	ErrDuplicateOriginalURL = errors.New("original url already exists")
	ErrDuplicateShortID     = errors.New("short id already exists")
)

type URLRepository interface {
	Save(ctx context.Context, url *entity.URL) error
	GetByShortID(ctx context.Context, shortID string) (*entity.URL, error)
	GetByOriginalURL(ctx context.Context, userID, originalURL string) (*entity.URL, error)
	ListByUser(ctx context.Context, userID string) ([]*entity.URL, error)
	DeleteByShortID(ctx context.Context, userID, shortID string) error
	Close() error
}
