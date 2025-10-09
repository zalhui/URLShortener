package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/zalhui/URLShortener/internal/entity"
	"github.com/zalhui/URLShortener/internal/repository"
)

type ShortenerService struct {
	repo    repository.URLRepository
	baseURL string
}

func NewShortenerService(repo repository.URLRepository, baseURL string) *ShortenerService {
	return &ShortenerService{
		repo:    repo,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func (s *ShortenerService) ShortenURL(originalURL string) (*entity.URL, error) {
	if exists, _ := s.repo.GetByOriginalURL(originalURL); exists != nil {
		return exists, nil
	}

	shortID, err := generateShortID()
	if err != nil {
		return nil, err
	}

	for {
		if exists, _ := s.repo.GetByShortID(shortID); exists == nil {
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
		OriginalURL: originalURL,
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
