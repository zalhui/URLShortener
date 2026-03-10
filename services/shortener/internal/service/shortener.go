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
	ErrNotFound   = errors.New("url not found")
	ErrInvalidURL = errors.New("invalid url")
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

func (s *ShortenerService) ShortenURL(originalURL string) (*entity.URL, error) {
	normalizedURL, err := normalizeURL(originalURL)
	if err != nil {
		return nil, err
	}

	if exists, err := s.repo.GetByOriginalURL(normalizedURL); err != nil {
		return nil, err
	} else if exists != nil {
		return exists, nil
	}

	shortID, err := generateShortID()
	if err != nil {
		return nil, err
	}

	for {
		exists, err := s.repo.GetByShortID(shortID)
		if err != nil {
			return nil, err
		}
		if exists == nil {
			break
		}

		shortID, err = generateShortID()
		if err != nil {
			return nil, err
		}
	}

	url := &entity.URL{
		UUID:        shortID,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, shortID),
		OriginalURL: normalizedURL,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.Save(url); err != nil {
		return nil, err
	}

	return url, nil
}

func (s *ShortenerService) GetOriginalURL(shortID string) (string, error) {
	url, err := s.repo.GetByShortID(shortID)
	if err != nil {
		return "", err
	}
	if url == nil {
		return "", ErrNotFound
	}

	return url.OriginalURL, nil
}

func generateShortID() (string, error) {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate short ID: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
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
