package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/repository"
)

var (
	ErrNotFound     = errors.New("url not found")
	ErrInvalidURL   = errors.New("invalid url")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
)

type Pinger interface {
	Ping(context.Context) error
}

type ShortenerService struct {
	pinger  Pinger
	repo    repository.URLRepository
	baseURL string
}

func NewShortenerService(repo repository.URLRepository, baseURL string, pinger Pinger) *ShortenerService {
	return &ShortenerService{
		repo:    repo,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		pinger:  pinger,
	}
}

func (s *ShortenerService) Ping(ctx context.Context) error {
	if s.pinger == nil {
		return nil
	}

	return s.pinger.Ping(ctx)
}

func (s *ShortenerService) ShortenURL(ctx context.Context, userID, originalURL string) (*entity.URL, error) {
	if userID == "" {
		return nil, ErrUnauthorized
	}

	normalizedURL, err := normalizeURL(originalURL)
	if err != nil {
		return nil, err
	}

	if exists, err := s.repo.GetByOriginalURL(ctx, userID, normalizedURL); err != nil {
		return nil, err
	} else if exists != nil {
		return exists, nil
	}

	for {
		shortID, err := generateShortID()
		if err != nil {
			return nil, err
		}

		exists, err := s.repo.GetByShortID(ctx, shortID)
		if err != nil {
			return nil, err
		}
		if exists != nil {
			continue
		}

		url := &entity.URL{
			UUID:        shortID,
			UserID:      userID,
			ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, shortID),
			OriginalURL: normalizedURL,
			CreatedAt:   time.Now().UTC(),
		}

		err = s.repo.Save(ctx, url)
		switch {
		case err == nil:
			return url, nil
		case errors.Is(err, repository.ErrDuplicateOriginalURL):
			existingURL, getErr := s.repo.GetByOriginalURL(ctx, userID, normalizedURL)
			if getErr != nil {
				return nil, getErr
			}
			if existingURL != nil {
				return existingURL, nil
			}
		case errors.Is(err, repository.ErrDuplicateShortID):
			continue
		default:
			return nil, err
		}
	}
}

func (s *ShortenerService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	url, err := s.repo.GetByShortID(ctx, shortID)
	if err != nil {
		return "", err
	}
	if url == nil {
		return "", ErrNotFound
	}

	return url.OriginalURL, nil
}

func (s *ShortenerService) ListUserURLs(ctx context.Context, userID string) ([]*entity.URL, error) {
	if userID == "" {
		return nil, ErrUnauthorized
	}

	return s.repo.ListByUser(ctx, userID)
}

func (s *ShortenerService) DeleteURL(ctx context.Context, userID, shortID string) error {
	if userID == "" {
		return ErrUnauthorized
	}

	url, err := s.repo.GetByShortID(ctx, shortID)
	if err != nil {
		return err
	}
	if url == nil {
		return ErrNotFound
	}
	if url.UserID != userID {
		return ErrForbidden
	}

	return s.repo.DeleteByShortID(ctx, userID, shortID)
}

func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate short ID: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func normalizeURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", ErrInvalidURL
	}

	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil || parsed.Host == "" {
		return "", ErrInvalidURL
	}

	return parsed.String(), nil
}
